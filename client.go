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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

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

	// Validate configuration
	if err := clientConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client configuration: %w", err)
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
		if info.Region == "" {
			info.Region = "us-east-1" // Default region
		}
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

	// First, get the current object info
	currentInfo, err := c.GetObjectInfo(ctx, bucket, key)
	if err != nil {
		return fmt.Errorf("failed to get current object info: %w", err)
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

	_, err = c.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to set object info for %s/%s: %w", bucket, key, err)
	}

	// Handle tags separately if provided
	if opts.Tags != nil && len(*opts.Tags) > 0 {
		// Convert tags to the format expected by PutObjectTagging
		tagSet := make([]types.Tag, 0, len(*opts.Tags))
		for key, value := range *opts.Tags {
			tagSet = append(tagSet, types.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
		}

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

	_ = currentInfo // Suppress unused variable warning

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

// ObjectInfo contains information about an S3 object
type ObjectInfo struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
	ETag         string            `json:"etag"`
	ContentType  string            `json:"content_type,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

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
