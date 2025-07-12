package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Client wraps the AWS S3 client with configuration
type S3Client struct {
	client *s3.Client
	config *ClientConfig
	mu     sync.RWMutex
}

// NewS3Client creates a new S3 client with the given configuration
func NewS3Client(ctx context.Context, clientConfig *ClientConfig) (*S3Client, error) {
	if clientConfig == nil {
		return nil, fmt.Errorf("client configuration is required")
	}

	// Validate configuration
	if err := clientConfig.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create AWS configuration
	awsConfig, err := createAWSConfig(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		// Set custom endpoint if provided
		if clientConfig.Endpoint != "" {
			o.BaseEndpoint = aws.String(clientConfig.Endpoint)
		}

		// Set path-style addressing if required
		if clientConfig.ForcePathStyle {
			o.UsePathStyle = true
		}

		// Set custom user agent
		if clientConfig.UserAgent != "" {
			o.AppID = clientConfig.UserAgent
		}
	})

	return &S3Client{
		client: s3Client,
		config: clientConfig,
	}, nil
}

// createAWSConfig creates AWS SDK configuration from client config
func createAWSConfig(ctx context.Context, clientConfig *ClientConfig) (aws.Config, error) {
	// Create configuration options
	var opts []func(*config.LoadOptions) error

	// Set region
	if clientConfig.Region != "" {
		opts = append(opts, config.WithRegion(clientConfig.Region))
	}

	// Set credentials if provided
	if clientConfig.AccessKey != "" && clientConfig.SecretKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			clientConfig.AccessKey,
			clientConfig.SecretKey,
			clientConfig.SessionToken,
		)
		opts = append(opts, config.WithCredentialsProvider(creds))
	}

	// Set retry configuration
	if clientConfig.MaxRetries > 0 {
		opts = append(opts, config.WithRetryMaxAttempts(clientConfig.MaxRetries))
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return cfg, nil
}

// GetConfig returns the client configuration
func (c *S3Client) GetConfig() *ClientConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// CreateBucket creates a new bucket
func (c *S3Client) CreateBucket(ctx context.Context, bucket string, region ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Set region if provided
	if len(region) > 0 && region[0] != "" {
		// Only set CreateBucketConfiguration for regions other than us-east-1
		if region[0] != "us-east-1" {
			input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
				LocationConstraint: types.BucketLocationConstraint(region[0]),
			}
		}
	}

	_, err := c.client.CreateBucket(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
	}

	return nil
}

// DeleteBucket deletes a bucket
func (c *S3Client) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If force is true, delete all objects first
	if force {
		if err := c.deleteAllObjects(ctx, bucket); err != nil {
			return fmt.Errorf("failed to delete objects in bucket %s: %w", bucket, err)
		}
	}

	_, err := c.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket %s: %w", bucket, err)
	}

	return nil
}

// ListBuckets lists all buckets
func (c *S3Client) ListBuckets(ctx context.Context) ([]BucketInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, err := c.client.ListBuckets(ctx, &s3.ListBucketsInput{})
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
func (c *S3Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchBucket") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return true, nil
}

// GetBucketInfo gets comprehensive information about a bucket
func (c *S3Client) GetBucketInfo(ctx context.Context, bucket string) (*BucketInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Initialize bucket info
	bucketInfo := &BucketInfo{
		Name: bucket,
	}

	// Get bucket location
	locationResult, err := c.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket location: %w", err)
	}

	// Handle empty location constraint (means us-east-1)
	region := string(locationResult.LocationConstraint)
	if region == "" {
		region = "us-east-1"
	}
	bucketInfo.Region = region

	// Get bucket creation date from list buckets (gracefully handle errors)
	listResult, err := c.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err == nil {
		for _, b := range listResult.Buckets {
			if aws.ToString(b.Name) == bucket {
				bucketInfo.CreationDate = aws.ToTime(b.CreationDate)
				break
			}
		}
	}
	// If ListBuckets fails, we don't treat it as an error - just leave CreationDate as zero value

	// Get bucket versioning status
	versioningResult, err := c.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		bucketInfo.VersioningStatus = string(versioningResult.Status)
	}

	// Get public access block settings
	publicAccessResult, err := c.client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucket),
	})
	if err == nil && publicAccessResult.PublicAccessBlockConfiguration != nil {
		config := publicAccessResult.PublicAccessBlockConfiguration
		bucketInfo.PublicAccessBlocked = aws.ToBool(config.BlockPublicAcls) &&
			aws.ToBool(config.BlockPublicPolicy) &&
			aws.ToBool(config.IgnorePublicAcls) &&
			aws.ToBool(config.RestrictPublicBuckets)
	}

	// Check if bucket has a policy
	_, err = c.client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	bucketInfo.HasPolicy = err == nil

	// Get bucket encryption
	_, err = c.client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucket),
	})
	bucketInfo.EncryptionEnabled = err == nil

	// Get object count and total size (approximate)
	objectsResult, err := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err == nil {
		bucketInfo.ObjectCount = int64(len(objectsResult.Contents))
		totalSize := int64(0)
		for _, obj := range objectsResult.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}
		bucketInfo.TotalSize = totalSize
	}

	return bucketInfo, nil
}

// PutObject uploads an object to S3
func (c *S3Client) PutObject(ctx context.Context, bucket, key string, body io.Reader, options ...PutObjectOption) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}

	// Apply options
	for _, opt := range options {
		opt(input)
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object %s/%s: %w", bucket, key, err)
	}

	return nil
}

// PutObjectFromFile uploads a file to S3
func (c *S3Client) PutObjectFromFile(ctx context.Context, bucket, key, filePath string, options ...PutObjectOption) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	for _, opt := range options {
		opt(input)
	}

	_, err = c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object %s/%s from file %s: %w", bucket, key, filePath, err)
	}

	return nil
}

// GetObject downloads an object from S3
func (c *S3Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s/%s: %w", bucket, key, err)
	}

	return result.Body, nil
}

// GetObjectToFile downloads an object from S3 to a local file
func (c *S3Client) GetObjectToFile(ctx context.Context, bucket, key, filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object %s/%s: %w", bucket, key, err)
	}
	defer result.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to copy object %s/%s to file %s: %w", bucket, key, filePath, err)
	}

	return nil
}

// DeleteObject deletes an object from S3
func (c *S3Client) DeleteObject(ctx context.Context, bucket, key string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s/%s: %w", bucket, key, err)
	}

	return nil
}

// ListObjects lists objects in a bucket
func (c *S3Client) ListObjects(ctx context.Context, bucket string, options ...ListObjectsOption) (*ListObjectsResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	// Apply options
	for _, opt := range options {
		opt(input)
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

// ObjectExists checks if an object exists
func (c *S3Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetObjectInfo gets metadata about an object
func (c *S3Client) GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
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

// SetObjectInfo sets metadata and other object properties by copying it with new settings
func (c *S3Client) SetObjectInfo(ctx context.Context, bucket, key string, options ...SetObjectInfoOption) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Copy object to itself with new metadata/settings
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		CopySource: aws.String(bucket + "/" + key),
	}

	// Set metadata directive to replace existing metadata
	input.MetadataDirective = "REPLACE"

	// Apply all options
	var tags map[string]string
	for _, opt := range options {
		tags = opt(input, tags)
	}

	_, err := c.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set object info for %s/%s: %w", bucket, key, err)
	}

	// Set tags if provided
	if len(tags) > 0 {
		var tagSet []types.Tag
		for k, v := range tags {
			tagSet = append(tagSet, types.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}

		_, err = c.client.PutObjectTagging(ctx, &s3.PutObjectTaggingInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Tagging: &types.Tagging{
				TagSet: tagSet,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to set tags for %s/%s: %w", bucket, key, err)
		}
	}

	return nil
}

// SetObjectInfoOption configures SetObjectInfo operations
type SetObjectInfoOption func(*s3.CopyObjectInput, map[string]string) map[string]string

// WithObjectMetadata sets metadata for SetObjectInfo
func WithObjectMetadata(metadata map[string]string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.Metadata = metadata
		return tags
	}
}

// WithObjectTags sets tags for SetObjectInfo
func WithObjectTags(newTags map[string]string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		if tags == nil {
			tags = make(map[string]string)
		}
		for k, v := range newTags {
			tags[k] = v
		}
		return tags
	}
}

// WithObjectContentType sets content type for SetObjectInfo
func WithObjectContentType(contentType string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.ContentType = aws.String(contentType)
		return tags
	}
}

// WithObjectCacheControl sets cache control for SetObjectInfo
func WithObjectCacheControl(cacheControl string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.CacheControl = aws.String(cacheControl)
		return tags
	}
}

// WithObjectContentEncoding sets content encoding for SetObjectInfo
func WithObjectContentEncoding(encoding string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.ContentEncoding = aws.String(encoding)
		return tags
	}
}

// WithObjectContentDisposition sets content disposition for SetObjectInfo
func WithObjectContentDisposition(disposition string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.ContentDisposition = aws.String(disposition)
		return tags
	}
}

// WithObjectContentLanguage sets content language for SetObjectInfo
func WithObjectContentLanguage(language string) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.ContentLanguage = aws.String(language)
		return tags
	}
}

// WithObjectExpires sets expires header for SetObjectInfo
func WithObjectExpires(expires time.Time) SetObjectInfoOption {
	return func(input *s3.CopyObjectInput, tags map[string]string) map[string]string {
		input.Expires = &expires
		return tags
	}
}

// CopyObject copies an object from one location to another
func (c *S3Client) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string, options ...PutObjectOption) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(dstBucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(srcBucket + "/" + srcKey),
	}

	// Create a dummy PutObjectInput to apply options, then transfer relevant fields
	dummyPutInput := &s3.PutObjectInput{}
	for _, opt := range options {
		opt(dummyPutInput)
	}

	// Transfer applicable fields from PutObjectInput to CopyObjectInput
	if dummyPutInput.ContentType != nil {
		input.ContentType = dummyPutInput.ContentType
		input.MetadataDirective = "REPLACE"
	}
	if dummyPutInput.Metadata != nil {
		input.Metadata = dummyPutInput.Metadata
		input.MetadataDirective = "REPLACE"
	}
	if dummyPutInput.CacheControl != nil {
		input.CacheControl = dummyPutInput.CacheControl
		input.MetadataDirective = "REPLACE"
	}
	if dummyPutInput.ContentEncoding != nil {
		input.ContentEncoding = dummyPutInput.ContentEncoding
		input.MetadataDirective = "REPLACE"
	}

	_, err := c.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to copy object from %s/%s to %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}

	return nil
}

// deleteAllObjects deletes all objects in a bucket (for force delete)
func (c *S3Client) deleteAllObjects(ctx context.Context, bucket string) error {
	// List all objects
	result, err := c.ListObjects(ctx, bucket)
	if err != nil {
		return err
	}

	// Delete objects in batches
	for len(result.Contents) > 0 {
		// Prepare delete input
		var objects []types.ObjectIdentifier
		for _, obj := range result.Contents {
			objects = append(objects, types.ObjectIdentifier{
				Key: aws.String(obj.Key),
			})
		}

		// Delete batch
		_, err := c.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &types.Delete{
				Objects: objects,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}

		// Check if there are more objects
		if !result.IsTruncated {
			break
		}

		// List next batch
		result, err = c.ListObjects(ctx, bucket, WithContinuationToken(result.NextMarker))
		if err != nil {
			return err
		}
	}

	return nil
}

// Data structures

// BucketInfo represents information about a bucket
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

// ObjectInfo represents information about an object
type ObjectInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ListObjectsResult represents the result of listing objects
type ListObjectsResult struct {
	Contents       []ObjectInfo `json:"contents"`
	CommonPrefixes []string     `json:"common_prefixes,omitempty"`
	IsTruncated    bool         `json:"is_truncated"`
	NextMarker     string       `json:"next_marker,omitempty"`
	MaxKeys        int          `json:"max_keys"`
	Prefix         string       `json:"prefix,omitempty"`
	Delimiter      string       `json:"delimiter,omitempty"`
}

// Option types for methods

// PutObjectOption configures PutObject operations
type PutObjectOption func(*s3.PutObjectInput)

// WithContentType sets the content type for PutObject
func WithContentType(contentType string) PutObjectOption {
	return func(input *s3.PutObjectInput) {
		input.ContentType = aws.String(contentType)
	}
}

// WithMetadata sets metadata for PutObject
func WithMetadata(metadata map[string]string) PutObjectOption {
	return func(input *s3.PutObjectInput) {
		input.Metadata = metadata
	}
}

// WithCacheControl sets cache control for PutObject
func WithCacheControl(cacheControl string) PutObjectOption {
	return func(input *s3.PutObjectInput) {
		input.CacheControl = aws.String(cacheControl)
	}
}

// WithContentEncoding sets content encoding for PutObject
func WithContentEncoding(encoding string) PutObjectOption {
	return func(input *s3.PutObjectInput) {
		input.ContentEncoding = aws.String(encoding)
	}
}

// ListObjectsOption configures ListObjects operations
type ListObjectsOption func(*s3.ListObjectsV2Input)

// WithPrefix sets the prefix for ListObjects
func WithPrefix(prefix string) ListObjectsOption {
	return func(input *s3.ListObjectsV2Input) {
		input.Prefix = aws.String(prefix)
	}
}

// WithDelimiter sets the delimiter for ListObjects
func WithDelimiter(delimiter string) ListObjectsOption {
	return func(input *s3.ListObjectsV2Input) {
		input.Delimiter = aws.String(delimiter)
	}
}

// WithMaxKeys sets the maximum number of keys for ListObjects
func WithMaxKeys(maxKeys int) ListObjectsOption {
	return func(input *s3.ListObjectsV2Input) {
		input.MaxKeys = aws.Int32(int32(maxKeys))
	}
}

// WithContinuationToken sets the continuation token for ListObjects
func WithContinuationToken(token string) ListObjectsOption {
	return func(input *s3.ListObjectsV2Input) {
		if token != "" {
			input.ContinuationToken = aws.String(token)
		}
	}
}
