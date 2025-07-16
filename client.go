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
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
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
		// For regions other than us-east-1, we need to set the location constraint
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
		buckets[i] = BucketInfo{
			Name:         aws.ToString(bucket.Name),
			CreationDate: aws.ToTime(bucket.CreationDate),
		}
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
		info.Region = string(locationResult.LocationConstraint)
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
	_, err = c.client.GetBucketEncryption(ctx, encryptionInput)
	if err == nil {
		info.EncryptionEnabled = true
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
		objects[i] = ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
		}
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

	return &ObjectInfo{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         aws.ToString(result.ETag),
		ContentType:  aws.ToString(result.ContentType),
		Metadata:     result.Metadata,
	}, nil
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

	// Handle tags separately if provided
	if opts.Tags != nil && len(*opts.Tags) > 0 {
		tagSet := tagsToAWSTagSet(*opts.Tags)
		if tagSet != nil {
			tagInput := &s3.PutObjectTaggingInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
				Tagging: &types.Tagging{
					TagSet: tagSet,
				},
			}

			_, err = c.client.PutObjectTagging(ctx, tagInput)
			if err != nil {
				return fmt.Errorf("failed to set object tags for %s/%s: %w", bucket, key, err)
			}
		}
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

// GetPublicURL generates a public HTTP URL for an object using client configuration
func (c *Client) GetPublicURL(bucket, key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return GenerateURLWithProvider(bucket, key, c.config.Region, c.config.Endpoint, c.config.UseSSL, c.config.ServiceType)
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

// BucketInfo contains information about an S3 bucket
type BucketInfo struct {
	Name                string    `json:"name"`
	CreationDate        time.Time `json:"creation_date,omitempty"`
	Region              string    `json:"region,omitempty"`
	VersioningStatus    string    `json:"versioning_status,omitempty"`
	PublicAccessBlocked bool      `json:"public_access_blocked,omitempty"`
	HasPolicy           bool      `json:"has_policy,omitempty"`
	EncryptionEnabled   bool      `json:"encryption_enabled,omitempty"`
	ObjectCount         int64     `json:"object_count,omitempty"`
	TotalSize           int64     `json:"total_size,omitempty"`
}

// MarshalStarlark implements the Marshaler interface for BucketInfo
func (b *BucketInfo) MarshalStarlark() (starlark.Value, error) {
	dict := starlark.NewDict(9)

	dict.SetKey(starlark.String("name"), starlark.String(b.Name))
	dict.SetKey(starlark.String("creation_date"), timeToStarlark(b.CreationDate))
	dict.SetKey(starlark.String("region"), starlark.String(b.Region))
	dict.SetKey(starlark.String("versioning_status"), starlark.String(b.VersioningStatus))
	dict.SetKey(starlark.String("public_access_blocked"), starlark.Bool(b.PublicAccessBlocked))
	dict.SetKey(starlark.String("has_policy"), starlark.Bool(b.HasPolicy))
	dict.SetKey(starlark.String("encryption_enabled"), starlark.Bool(b.EncryptionEnabled))
	dict.SetKey(starlark.String("object_count"), starlark.MakeInt64(b.ObjectCount))
	dict.SetKey(starlark.String("total_size"), starlark.MakeInt64(b.TotalSize))

	return dict, nil
}

// Ensure BucketInfo implements dataconv.Marshaler
var _ dataconv.Marshaler = (*BucketInfo)(nil)

// ObjectInfo contains information about an S3 object
type ObjectInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// MarshalStarlark implements the Marshaler interface for ObjectInfo
func (o *ObjectInfo) MarshalStarlark() (starlark.Value, error) {
	dict := starlark.NewDict(6)

	dict.SetKey(starlark.String("key"), starlark.String(o.Key))
	dict.SetKey(starlark.String("size"), starlark.MakeInt64(o.Size))
	dict.SetKey(starlark.String("last_modified"), timeToStarlark(o.LastModified))
	dict.SetKey(starlark.String("etag"), starlark.String(o.ETag))
	dict.SetKey(starlark.String("content_type"), starlark.String(o.ContentType))
	dict.SetKey(starlark.String("metadata"), stringMapToStarlark(o.Metadata))

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
func (l *ListObjectsResult) MarshalStarlark() (starlark.Value, error) {
	dict := starlark.NewDict(7)

	// Convert Contents slice to Starlark list
	contentsList := starlark.NewList(make([]starlark.Value, len(l.Contents)))
	for i, obj := range l.Contents {
		objValue, err := obj.MarshalStarlark()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal object info: %w", err)
		}
		contentsList.SetIndex(i, objValue)
	}
	dict.SetKey(starlark.String("contents"), contentsList)

	// Convert CommonPrefixes slice to Starlark list using utility function
	dict.SetKey(starlark.String("common_prefixes"), stringSliceToStarlark(l.CommonPrefixes))

	dict.SetKey(starlark.String("is_truncated"), starlark.Bool(l.IsTruncated))
	dict.SetKey(starlark.String("next_marker"), starlark.String(l.NextMarker))
	dict.SetKey(starlark.String("max_keys"), starlark.MakeInt(l.MaxKeys))
	dict.SetKey(starlark.String("prefix"), starlark.String(l.Prefix))
	dict.SetKey(starlark.String("delimiter"), starlark.String(l.Delimiter))

	return dict, nil
}

// Ensure ListObjectsResult implements dataconv.Marshaler
var _ dataconv.Marshaler = (*ListObjectsResult)(nil)
