package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/1set/starlet/dataconv"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

// Helper function to get owner display name from AWS Owner type
func getOwnerDisplayName(owner *types.Owner) string {
	if owner == nil {
		return ""
	}
	if owner.DisplayName != nil {
		return aws.ToString(owner.DisplayName)
	}
	if owner.ID != nil {
		return aws.ToString(owner.ID)
	}
	return ""
}

// convertAWSBucketToBucketInfo converts an AWS Bucket to our BucketInfo struct
func convertAWSBucketToBucketInfo(bucket types.Bucket) BucketInfo {
	return BucketInfo{
		Name:         aws.ToString(bucket.Name),
		CreationDate: aws.ToTime(bucket.CreationDate),
	}
}

// convertAWSObjectToObjectInfo converts an AWS Object to our ObjectInfo struct
func convertAWSObjectToObjectInfo(obj types.Object) ObjectInfo {
	objInfo := ObjectInfo{
		Key:          aws.ToString(obj.Key),
		Size:         aws.ToInt64(obj.Size),
		LastModified: aws.ToTime(obj.LastModified),
		ETag:         aws.ToString(obj.ETag),
		StorageClass: string(obj.StorageClass),
		Owner:        getOwnerDisplayName(obj.Owner),
	}

	// Report the object's checksum algorithm as its own field. (A ListObjectsV2
	// entry carries no VersionId — that comes only from ListObjectVersions — so
	// version_id is deliberately left empty here rather than being overwritten
	// with the checksum algorithm.)
	if len(obj.ChecksumAlgorithm) > 0 {
		objInfo.ChecksumAlgorithm = string(obj.ChecksumAlgorithm[0])
	}

	return objInfo
}

// Utility helper functions for Starlark conversions

// timeToStarlark converts time.Time to starlark.Value, returning None for zero time
func timeToStarlark(t time.Time) starlark.Value {
	if t.IsZero() {
		return starlark.None
	}
	return startime.Time(t)
}

// stringMapToStarlark converts map[string]string to starlark.Dict
func stringMapToStarlark(m map[string]string) *starlark.Dict {
	dict := &starlark.Dict{}
	for k, v := range m {
		dict.SetKey(starlark.String(k), starlark.String(v))
	}
	return dict
}

// stringSliceToStarlark converts []string to starlark.List
func stringSliceToStarlark(s []string) *starlark.List {
	values := make([]starlark.Value, len(s))
	for i, str := range s {
		values[i] = starlark.String(str)
	}
	return starlark.NewList(values)
}

// tagsToAWSTagSet converts a map[string]string to AWS TagSet
func tagsToAWSTagSet(tags map[string]string) []types.Tag {
	if tags == nil || len(tags) == 0 {
		return nil
	}

	tagSet := make([]types.Tag, 0, len(tags))
	for key, value := range tags {
		tagSet = append(tagSet, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return tagSet
}

// Client wraps the AWS S3 client with configuration
type Client struct {
	client *s3.Client
	config *ClientConfig
	mu     sync.RWMutex
}

// NewClient creates a new S3 client with the provided configuration
func NewClient(ctx context.Context, clientConfig *ClientConfig) (*Client, error) {
	if clientConfig == nil {
		return nil, fmt.Errorf("client config cannot be nil")
	}

	// Create AWS configuration
	cfg, err := createAWSConfig(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	// Create S3 client with custom endpoint if provided
	var s3Client *s3.Client
	if clientConfig.Endpoint != "" {
		s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(clientConfig.Endpoint)
			o.UsePathStyle = clientConfig.ForcePathStyle
		})
	} else {
		s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = clientConfig.ForcePathStyle
		})
	}

	return &Client{
		client: s3Client,
		config: clientConfig,
	}, nil
}

// createAWSConfig creates the AWS configuration from client configuration
func createAWSConfig(ctx context.Context, clientConfig *ClientConfig) (aws.Config, error) {
	// Create options for AWS config
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(clientConfig.Region),
	}

	// Add credentials if provided
	if clientConfig.AccessKey != "" && clientConfig.SecretKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			clientConfig.AccessKey,
			clientConfig.SecretKey,
			clientConfig.SessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	// Apply the client-behavior settings (timeout, retries, logging, user agent)
	// so the configured values actually reach the SDK rather than being dropped.
	opts = append(opts, clientBehaviorOptions(clientConfig)...)

	// Load configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
}

// clientBehaviorOptions maps the resolved timeout / max-retries / logging /
// user-agent settings onto AWS config load options. Without this the SDK client
// silently ignores those config values. Each lever is applied only when set, so
// an unset value keeps the SDK default (the historical behavior).
func clientBehaviorOptions(cc *ClientConfig) []func(*config.LoadOptions) error {
	var opts []func(*config.LoadOptions) error

	// Per-request timeout: a bounded HTTP client so a hung request can't block
	// the caller forever. Capped so an absurd value can't overflow the
	// nanosecond duration into a negative one, which http.Client would treat as
	// "no deadline" and silently disable the bound.
	if cc.Timeout > 0 {
		opts = append(opts, config.WithHTTPClient(
			awshttp.NewBuildableClient().WithTimeout(timeoutDuration(cc.Timeout))))
	}

	// Maximum attempts (including the first try; the SDK default is 3). A
	// positive value takes precedence over any ambient AWS_MAX_ATTEMPTS; a
	// non-positive value is skipped so the SDK/env default applies.
	if cc.MaxRetries > 0 {
		opts = append(opts, config.WithRetryMaxAttempts(cc.MaxRetries))
	}

	// Request/response logging when explicitly enabled.
	if cc.EnableLogging {
		opts = append(opts, config.WithClientLogMode(aws.LogRequest|aws.LogResponse))
	}

	// Custom user-agent token appended to the SDK's own user agent. A
	// "name/version" string is added as a proper key/value pair; AddUserAgentKey
	// alone would sanitize the '/' into '-' (turning "Starlark-S3/1.0" into
	// "Starlark-S3-1.0").
	if cc.UserAgent != "" {
		opts = append(opts, config.WithAPIOptions([]func(*smithymiddleware.Stack) error{
			userAgentOption(cc.UserAgent),
		}))
	}

	return opts
}

// maxRequestTimeoutSeconds caps the per-request timeout at one day. Any positive
// timeout is honored up to this bound; the cap only exists to keep the
// nanosecond conversion from overflowing into a negative (unbounded) duration.
const maxRequestTimeoutSeconds = 24 * 60 * 60

// timeoutDuration converts a positive timeout in seconds to a duration, clamped
// to maxRequestTimeoutSeconds so the conversion cannot overflow.
func timeoutDuration(seconds int) time.Duration {
	if seconds > maxRequestTimeoutSeconds {
		seconds = maxRequestTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

// splitUserAgent decides how a configured user-agent string maps onto the SDK's
// user-agent API: a "name/version" form is split so the '/' is preserved as a
// key/value pair, while a bare string is a single key. Returned as a pure
// decision so the routing is unit-testable without running a middleware stack.
func splitUserAgent(ua string) (name, version string, pair bool) {
	return strings.Cut(ua, "/")
}

// userAgentOption builds the user-agent middleware for a configured agent
// string. A "name/version" form is registered as a key/value pair (so the '/'
// survives; AddUserAgentKey alone would sanitize it to '-'); a bare string is
// added as a single key.
func userAgentOption(ua string) func(*smithymiddleware.Stack) error {
	if name, version, pair := splitUserAgent(ua); pair {
		return awsmiddleware.AddUserAgentKeyValue(name, version)
	}
	return awsmiddleware.AddUserAgentKey(ua)
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *ClientConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// CreateBucket creates a new S3 bucket
func (c *Client) CreateBucket(ctx context.Context, bucket string, region ...string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Set region if provided
	if len(region) > 0 && region[0] != "" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region[0]),
		}
	}

	_, err := c.client.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
	}

	return nil
}

// DeleteBucket deletes an S3 bucket
func (c *Client) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	// If force is true, delete all objects in the bucket first
	if force {
		if err := c.deleteAllObjects(ctx, bucket); err != nil {
			return fmt.Errorf("failed to delete objects in bucket %s: %w", bucket, err)
		}
	}

	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := c.client.DeleteBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete bucket %s: %w", bucket, err)
	}

	return nil
}

// ListBuckets lists all S3 buckets
func (c *Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	input := &s3.ListBucketsInput{}

	result, err := c.client.ListBuckets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	buckets := make([]BucketInfo, len(result.Buckets))
	for i, bucket := range result.Buckets {
		buckets[i] = convertAWSBucketToBucketInfo(bucket)
	}

	return buckets, nil
}

// BucketExists checks if a bucket exists
func (c *Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	}

	_, err := c.client.HeadBucket(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchBucket") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return true, nil
}

// GetBucketInfo gets information about a bucket
func (c *Client) GetBucketInfo(ctx context.Context, bucket string) (*BucketInfo, error) {
	// Check if bucket exists
	exists, err := c.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", bucket)
	}

	info := &BucketInfo{
		Name: bucket,
	}

	// Get bucket location
	locationInput := &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	}
	locationResult, err := c.client.GetBucketLocation(ctx, locationInput)
	if err == nil {
		region := string(locationResult.LocationConstraint)
		info.Region = region
		info.Location = region
	}

	// Get bucket versioning
	versioningInput := &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	}
	versioningResult, err := c.client.GetBucketVersioning(ctx, versioningInput)
	if err == nil {
		info.VersioningStatus = string(versioningResult.Status)
	}

	// Try to get bucket policy to check if it has one
	policyInput := &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	}
	_, err = c.client.GetBucketPolicy(ctx, policyInput)
	if err == nil {
		info.HasPolicy = true
	}

	// Try to get bucket encryption
	encryptionInput := &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucket),
	}
	encryptionResult, err := c.client.GetBucketEncryption(ctx, encryptionInput)
	if err == nil && encryptionResult.ServerSideEncryptionConfiguration != nil {
		info.EncryptionEnabled = true
		// Get encryption type from the first rule
		if len(encryptionResult.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := encryptionResult.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				info.EncryptionType = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
			}
		}
	}

	// Try to get bucket CORS configuration
	corsInput := &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	}
	_, err = c.client.GetBucketCors(ctx, corsInput)
	if err == nil {
		info.HasCors = true
	}

	// Try to get public access block
	publicAccessInput := &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucket),
	}
	publicAccessResult, err := c.client.GetPublicAccessBlock(ctx, publicAccessInput)
	if err == nil && publicAccessResult.PublicAccessBlockConfiguration != nil {
		config := publicAccessResult.PublicAccessBlockConfiguration
		info.PublicAccessBlocked = aws.ToBool(config.BlockPublicAcls) &&
			aws.ToBool(config.BlockPublicPolicy) &&
			aws.ToBool(config.IgnorePublicAcls) &&
			aws.ToBool(config.RestrictPublicBuckets)
	}

	// Get object count and total size (this is expensive for large buckets)
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	var objectCount int64
	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(c.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			break // Don't fail the entire operation if we can't get object stats
		}

		objectCount += int64(len(page.Contents))
		for _, obj := range page.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}
	}

	info.ObjectCount = objectCount
	info.TotalSize = totalSize

	// Get bucket tags with a separate API call
	tagsInput := &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	}

	tagsResult, err := c.client.GetBucketTagging(ctx, tagsInput)
	if err != nil {
		// Tags might not be accessible or bucket might not have tags
		// Don't fail the entire operation, just initialize empty tags
		info.Tags = make(map[string]string)
	} else {
		// Convert AWS tags to map
		info.Tags = make(map[string]string)
		for _, tag := range tagsResult.TagSet {
			if tag.Key != nil && tag.Value != nil {
				info.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return info, nil
}

// PutObject uploads an object to S3
func (c *Client) PutObject(ctx context.Context, bucket, key string, body io.Reader, opts *ObjectOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}

	// Apply options
	if opts != nil {
		opts.ApplyToPutObject(input)
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object %s/%s: %w", bucket, key, err)
	}

	return nil
}

// PutObjectFromFile uploads a file to S3
func (c *Client) PutObjectFromFile(ctx context.Context, bucket, key, filePath string, opts *ObjectOptions) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	}

	// Apply options
	if opts != nil {
		opts.ApplyToPutObject(input)
	}

	_, err = c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object from file %s to %s/%s: %w", filePath, bucket, key, err)
	}

	return nil
}

// GetObject downloads an object from S3
func (c *Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s/%s: %w", bucket, key, err)
	}

	return result.Body, nil
}

// GetObjectToFile downloads an object from S3 to a file
func (c *Client) GetObjectToFile(ctx context.Context, bucket, key, filePath string) error {
	// Get the object
	body, err := c.GetObject(ctx, bucket, key)
	if err != nil {
		return err
	}
	defer body.Close()

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Copy the content
	_, err = io.Copy(file, body)
	if err != nil {
		return fmt.Errorf("failed to copy object to file %s: %w", filePath, err)
	}

	return nil
}

// DeleteObject deletes an object from S3
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object %s/%s: %w", bucket, key, err)
	}

	return nil
}

// ListObjects lists objects in a bucket
func (c *Client) ListObjects(ctx context.Context, bucket string, opts *ListObjectsOptions) (*ListObjectsResult, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	// Apply options
	if opts != nil {
		opts.ApplyToListObjects(input)
	}

	// max_keys bounds the total number of items returned across pages (default
	// 1000, the historical single-page size). S3 returns at most 1000 items per
	// request, so a larger total is auto-paginated instead of being silently
	// clamped to one page. Memory use is bounded by the caller's max_keys.
	maxTotal := listMaxTotal(opts)
	objects, commonPrefixes, nextToken, truncated, err := paginateListObjects(ctx, c.client, input, maxTotal)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	return &ListObjectsResult{
		Contents:       objects,
		CommonPrefixes: commonPrefixes,
		IsTruncated:    truncated,
		NextMarker:     nextToken,
		MaxKeys:        maxTotal,
		Prefix:         aws.ToString(input.Prefix),
		Delimiter:      aws.ToString(input.Delimiter),
	}, nil
}

// s3MaxKeysPerPage is the maximum number of items S3 returns in a single
// ListObjectsV2 page.
const s3MaxKeysPerPage = 1000

// listMaxTotal returns the total number of items (objects plus grouped common
// prefixes) a listing should return across auto-paginated pages. max_keys
// (default 1000) bounds it; S3 counts objects and common prefixes together
// against MaxKeys, so the loop must bound on their sum, not objects alone.
func listMaxTotal(opts *ListObjectsOptions) int {
	if opts != nil && opts.MaxKeys != nil && *opts.MaxKeys > 0 {
		return *opts.MaxKeys
	}
	return 1000
}

// listObjectsV2API is the one S3 call paginateListObjects makes; *s3.Client
// satisfies it, and a fake exercises the paging loop in tests without a live
// client.
type listObjectsV2API interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// paginateListObjects lists objects (and common prefixes) across pages, never
// requesting or keeping more than maxTotal items combined. Each page requests
// exactly the remaining budget (bounded by the S3 per-page maximum), so a page
// can never overshoot the cap — which means the returned nextToken always
// resumes precisely after the returned set, and a delimiter listing that yields
// mostly common prefixes is still bounded (S3 counts prefixes against MaxKeys,
// and so does this loop). truncated is true when more items remain beyond the
// returned set.
func paginateListObjects(ctx context.Context, api listObjectsV2API, input *s3.ListObjectsV2Input, maxTotal int) (objects []ObjectInfo, commonPrefixes []string, nextToken string, truncated bool, err error) {
	// Honor a caller-supplied starting cursor (opts may set one) rather than
	// silently restarting from the beginning.
	token := input.ContinuationToken
	for len(objects)+len(commonPrefixes) < maxTotal {
		pageLimit := maxTotal - (len(objects) + len(commonPrefixes))
		if pageLimit > s3MaxKeysPerPage {
			pageLimit = s3MaxKeysPerPage
		}
		input.MaxKeys = aws.Int32(int32(pageLimit))
		input.ContinuationToken = token

		out, perr := api.ListObjectsV2(ctx, input)
		if perr != nil {
			return nil, nil, "", false, perr
		}
		for _, obj := range out.Contents {
			objects = append(objects, convertAWSObjectToObjectInfo(obj))
		}
		for _, prefix := range out.CommonPrefixes {
			commonPrefixes = append(commonPrefixes, aws.ToString(prefix.Prefix))
		}

		next, keepGoing := advanceToken(out, token)
		if !keepGoing {
			return objects, commonPrefixes, aws.ToString(next), aws.ToBool(out.IsTruncated), nil
		}
		token = next
	}
	// Stopped with the budget full while S3 still has more: the token resumes
	// exactly after the returned set because no page overshot.
	return objects, commonPrefixes, aws.ToString(token), true, nil
}

// advanceToken decides the cursor to resume from after a page and whether paging
// should continue. It stops (keepGoing=false) when the page was the last one
// (not truncated) or when the cursor fails to advance — an empty or unchanged
// continuation token on a truncated page, which would otherwise spin forever
// against a misbehaving endpoint.
func advanceToken(out *s3.ListObjectsV2Output, current *string) (next *string, keepGoing bool) {
	if !aws.ToBool(out.IsTruncated) {
		return nil, false
	}
	n := out.NextContinuationToken
	if aws.ToString(n) == "" || aws.ToString(n) == aws.ToString(current) {
		return n, false
	}
	return n, true
}

// ObjectExists checks if an object exists in S3
func (c *Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.HeadObject(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetObjectInfo gets information about an object
func (c *Client) GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info for %s/%s: %w", bucket, key, err)
	}

	objInfo := &ObjectInfo{
		Key:                key,
		Size:               aws.ToInt64(result.ContentLength),
		LastModified:       aws.ToTime(result.LastModified),
		ETag:               aws.ToString(result.ETag),
		ContentType:        aws.ToString(result.ContentType),
		ContentEncoding:    aws.ToString(result.ContentEncoding),
		ContentDisposition: aws.ToString(result.ContentDisposition),
		ContentLanguage:    aws.ToString(result.ContentLanguage),
		CacheControl:       aws.ToString(result.CacheControl),
		StorageClass:       string(result.StorageClass),
		VersionID:          aws.ToString(result.VersionId),
		Metadata:           result.Metadata,
	}

	// Handle expires field if present
	if result.Expires != nil {
		objInfo.Expires = result.Expires
	}

	// Get object tags with a separate API call
	tagsInput := &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	tagsResult, err := c.client.GetObjectTagging(ctx, tagsInput)
	if err != nil {
		// Tags might not be accessible or object might not have tags
		// Don't fail the entire operation, just log and continue
		objInfo.Tags = make(map[string]string)
	} else {
		// Convert AWS tags to map
		objInfo.Tags = make(map[string]string)
		for _, tag := range tagsResult.TagSet {
			if tag.Key != nil && tag.Value != nil {
				objInfo.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return objInfo, nil
}

// SetObjectInfo sets metadata and other properties for an existing object
func (c *Client) SetObjectInfo(ctx context.Context, bucket, key string, opts *ObjectOptions) error {
	// Return early if no options provided
	if opts == nil || !opts.Validate() {
		return nil
	}

	// Create copy input
	copySource := fmt.Sprintf("%s/%s", bucket, key)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		CopySource: aws.String(copySource),
	}

	// Apply options
	opts.ApplyToCopyObject(input)

	// If no metadata directive was set, we need to preserve existing metadata
	if input.MetadataDirective == "" {
		input.MetadataDirective = types.MetadataDirectiveCopy
	}

	_, err := c.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set object info for %s/%s: %w", bucket, key, err)
	}

	return nil
}

// CopyObject copies an object from one location to another
func (c *Client) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string, opts *ObjectOptions) error {
	copySource := fmt.Sprintf("%s/%s", srcBucket, srcKey)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(copySource),
	}

	// Apply options
	if opts != nil {
		opts.ApplyToCopyObject(input)
	}

	_, err := c.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to copy object from %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}

	return nil
}

// PresignURL generates a pre-signed URL for temporary access to an object
func (c *Client) PresignURL(ctx context.Context, bucket, key string, expiresInSec int, method string) (string, error) {
	// Create presign client
	presignClient := s3.NewPresignClient(c.client)

	// Set expiration duration
	expiration := time.Duration(expiresInSec) * time.Second

	var req *v4.PresignedHTTPRequest
	var err error

	switch strings.ToUpper(method) {
	case "GET":
		getReq := &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		req, err = presignClient.PresignGetObject(ctx, getReq, func(opts *s3.PresignOptions) {
			opts.Expires = expiration
		})
	case "PUT":
		putReq := &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		req, err = presignClient.PresignPutObject(ctx, putReq, func(opts *s3.PresignOptions) {
			opts.Expires = expiration
		})
	case "HEAD":
		headReq := &s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		req, err = presignClient.PresignHeadObject(ctx, headReq, func(opts *s3.PresignOptions) {
			opts.Expires = expiration
		})
	default:
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return "", fmt.Errorf("failed to presign URL for %s/%s: %w", bucket, key, err)
	}

	return req.URL, nil
}

// deleteAllObjects deletes all objects in a bucket (helper for force delete)
func (c *Client) deleteAllObjects(ctx context.Context, bucket string) error {
	// List all objects
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	paginator := s3.NewListObjectsV2Paginator(c.client, listInput)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects for deletion: %w", err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		// Prepare objects for deletion
		objects := make([]types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = types.ObjectIdentifier{
				Key: obj.Key,
			}
		}

		// Delete objects in batch
		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{
				Objects: objects,
			},
		}

		output, err := c.client.DeleteObjects(ctx, deleteInput)
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}

		// DeleteObjects returns HTTP 200 even when individual objects fail
		// (object lock, governance, permissions), reporting them in .Errors
		// rather than as a request error. Surfacing them here keeps a force
		// delete honest: without this, the batch call "succeeds" while objects
		// remain and the subsequent DeleteBucket fails with "bucket not empty".
		if err := deleteObjectsPartialError(output.Errors); err != nil {
			return err
		}
	}

	return nil
}

// deleteObjectsPartialError turns the per-object failures of a DeleteObjects
// batch (returned in the response body alongside HTTP 200) into an error, so a
// partially-failed force delete is reported as a failure rather than a success.
// It returns nil when no objects failed.
func deleteObjectsPartialError(errs []types.Error) error {
	if len(errs) == 0 {
		return nil
	}
	first := errs[0]
	return fmt.Errorf("failed to delete %d object(s), first: key %q: %s: %s",
		len(errs), aws.ToString(first.Key), aws.ToString(first.Code), aws.ToString(first.Message))
}

// BucketInfo contains comprehensive information about an S3 bucket
type BucketInfo struct {
	Name                string            `json:"name"`
	CreationDate        time.Time         `json:"creation_date,omitempty"`
	Region              string            `json:"region,omitempty"`
	Location            string            `json:"location,omitempty"`
	VersioningStatus    string            `json:"versioning_status,omitempty"`
	PublicAccessBlocked bool              `json:"public_access_blocked,omitempty"`
	HasPolicy           bool              `json:"has_policy,omitempty"`
	HasCors             bool              `json:"has_cors,omitempty"`
	EncryptionEnabled   bool              `json:"encryption_enabled,omitempty"`
	EncryptionType      string            `json:"encryption_type,omitempty"`
	ObjectCount         int64             `json:"object_count,omitempty"`
	TotalSize           int64             `json:"total_size,omitempty"`
	StorageClass        string            `json:"storage_class,omitempty"`
	Tags                map[string]string `json:"tags,omitempty"`
	Owner               string            `json:"owner,omitempty"`
	BucketType          string            `json:"bucket_type,omitempty"`
}

// MarshalStarlark implements the Marshaler interface for BucketInfo
func (b *BucketInfo) MarshalStarlark() (starlark.Value, error) {
	dict := starlark.NewDict(16)

	dict.SetKey(starlark.String("name"), starlark.String(b.Name))
	dict.SetKey(starlark.String("creation_date"), timeToStarlark(b.CreationDate))
	dict.SetKey(starlark.String("region"), starlark.String(b.Region))
	dict.SetKey(starlark.String("location"), starlark.String(b.Location))
	dict.SetKey(starlark.String("versioning_status"), starlark.String(b.VersioningStatus))
	dict.SetKey(starlark.String("public_access_blocked"), starlark.Bool(b.PublicAccessBlocked))
	dict.SetKey(starlark.String("has_policy"), starlark.Bool(b.HasPolicy))
	dict.SetKey(starlark.String("has_cors"), starlark.Bool(b.HasCors))
	dict.SetKey(starlark.String("encryption_enabled"), starlark.Bool(b.EncryptionEnabled))
	dict.SetKey(starlark.String("encryption_type"), starlark.String(b.EncryptionType))
	dict.SetKey(starlark.String("object_count"), starlark.MakeInt64(b.ObjectCount))
	dict.SetKey(starlark.String("total_size"), starlark.MakeInt64(b.TotalSize))
	dict.SetKey(starlark.String("storage_class"), starlark.String(b.StorageClass))
	dict.SetKey(starlark.String("tags"), stringMapToStarlark(b.Tags))
	dict.SetKey(starlark.String("owner"), starlark.String(b.Owner))
	dict.SetKey(starlark.String("bucket_type"), starlark.String(b.BucketType))

	return dict, nil
}

// Ensure BucketInfo implements dataconv.Marshaler
var _ dataconv.Marshaler = (*BucketInfo)(nil)

// ObjectInfo contains comprehensive information about an S3 object
type ObjectInfo struct {
	Key                string            `json:"key"`
	Size               int64             `json:"size"`
	LastModified       time.Time         `json:"last_modified"`
	ETag               string            `json:"etag"`
	ContentType        string            `json:"content_type,omitempty"`
	ContentEncoding    string            `json:"content_encoding,omitempty"`
	ContentDisposition string            `json:"content_disposition,omitempty"`
	ContentLanguage    string            `json:"content_language,omitempty"`
	CacheControl       string            `json:"cache_control,omitempty"`
	Expires            *time.Time        `json:"expires,omitempty"`
	StorageClass       string            `json:"storage_class,omitempty"`
	ChecksumAlgorithm  string            `json:"checksum_algorithm,omitempty"`
	VersionID          string            `json:"version_id,omitempty"`
	IsLatest           bool              `json:"is_latest,omitempty"`
	Owner              string            `json:"owner,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
}

// MarshalStarlark implements the Marshaler interface for ObjectInfo
func (o *ObjectInfo) MarshalStarlark() (starlark.Value, error) {
	dict := starlark.NewDict(17)

	dict.SetKey(starlark.String("key"), starlark.String(o.Key))
	dict.SetKey(starlark.String("size"), starlark.MakeInt64(o.Size))
	dict.SetKey(starlark.String("last_modified"), timeToStarlark(o.LastModified))
	dict.SetKey(starlark.String("etag"), starlark.String(o.ETag))
	dict.SetKey(starlark.String("content_type"), starlark.String(o.ContentType))
	dict.SetKey(starlark.String("content_encoding"), starlark.String(o.ContentEncoding))
	dict.SetKey(starlark.String("content_disposition"), starlark.String(o.ContentDisposition))
	dict.SetKey(starlark.String("content_language"), starlark.String(o.ContentLanguage))
	dict.SetKey(starlark.String("cache_control"), starlark.String(o.CacheControl))

	// Handle nullable expires field
	if o.Expires != nil {
		dict.SetKey(starlark.String("expires"), timeToStarlark(*o.Expires))
	} else {
		dict.SetKey(starlark.String("expires"), starlark.None)
	}

	dict.SetKey(starlark.String("storage_class"), starlark.String(o.StorageClass))
	dict.SetKey(starlark.String("checksum_algorithm"), starlark.String(o.ChecksumAlgorithm))
	dict.SetKey(starlark.String("version_id"), starlark.String(o.VersionID))
	dict.SetKey(starlark.String("is_latest"), starlark.Bool(o.IsLatest))
	dict.SetKey(starlark.String("owner"), starlark.String(o.Owner))
	dict.SetKey(starlark.String("metadata"), stringMapToStarlark(o.Metadata))
	dict.SetKey(starlark.String("tags"), stringMapToStarlark(o.Tags))

	return dict, nil
}

// Ensure ObjectInfo implements dataconv.Marshaler
var _ dataconv.Marshaler = (*ObjectInfo)(nil)

// ListObjectsResult contains the result of a list objects operation
type ListObjectsResult struct {
	Contents       []ObjectInfo `json:"contents"`
	CommonPrefixes []string     `json:"common_prefixes,omitempty"`
	IsTruncated    bool         `json:"is_truncated"`
	NextMarker     string       `json:"next_marker,omitempty"`
	MaxKeys        int          `json:"max_keys"`
	Prefix         string       `json:"prefix,omitempty"`
	Delimiter      string       `json:"delimiter,omitempty"`
}

// MarshalStarlark implements the Marshaler interface for ListObjectsResult
// Returns a dict with all ListObjectsResult fields
func (l *ListObjectsResult) MarshalStarlark() (starlark.Value, error) {
	// Convert Contents slice to Starlark list
	contentsList := starlark.NewList(make([]starlark.Value, len(l.Contents)))
	for i, obj := range l.Contents {
		objValue, err := obj.MarshalStarlark()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object info: %w", err)
		}
		contentsList.SetIndex(i, objValue)
	}
	return contentsList, nil
}

// Ensure ListObjectsResult implements dataconv.Marshaler
var _ dataconv.Marshaler = (*ListObjectsResult)(nil)
