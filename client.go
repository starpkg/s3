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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

	// Handle checksum algorithms if present
	if len(obj.ChecksumAlgorithm) > 0 {
		// Store the first checksum algorithm as a string representation
		objInfo.VersionID = string(obj.ChecksumAlgorithm[0])
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

	// Load configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
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

	result, err := c.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	// Convert to our result format
	objects := make([]ObjectInfo, len(result.Contents))
	for i, obj := range result.Contents {
		objects[i] = convertAWSObjectToObjectInfo(obj)
	}

	commonPrefixes := make([]string, len(result.CommonPrefixes))
	for i, prefix := range result.CommonPrefixes {
		commonPrefixes[i] = aws.ToString(prefix.Prefix)
	}

	return &ListObjectsResult{
		Contents:       objects,
		CommonPrefixes: commonPrefixes,
		IsTruncated:    aws.ToBool(result.IsTruncated),
		NextMarker:     aws.ToString(result.NextContinuationToken),
		MaxKeys:        int(aws.ToInt32(result.MaxKeys)),
		Prefix:         aws.ToString(result.Prefix),
		Delimiter:      aws.ToString(result.Delimiter),
	}, nil
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

		_, err = c.client.DeleteObjects(ctx, deleteInput)
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	return nil
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
	VersionID          string            `json:"version_id,omitempty"`
	IsLatest           bool              `json:"is_latest,omitempty"`
	Owner              string            `json:"owner,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
}

// MarshalStarlark implements the Marshaler interface for ObjectInfo
func (o *ObjectInfo) MarshalStarlark() (starlark.Value, error) {
	dict := starlark.NewDict(16)

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
