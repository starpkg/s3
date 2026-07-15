package s3

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Configuration key constants
const (
	configKeyServiceType          = "service_type"
	configKeyAccessKey            = "access_key"
	configKeySecretKey            = "secret_key"
	configKeySessionToken         = "session_token"
	configKeyRegion               = "region"
	configKeyEndpoint             = "endpoint"
	configKeyForcePathStyle       = "force_path_style"
	configKeyUseSSL               = "use_ssl"
	configKeyTimeout              = "timeout"
	configKeyMaxRetries           = "max_retries"
	configKeyPartSize             = "part_size"
	configKeyConcurrency          = "concurrency"
	configKeyEnableLogging        = "enable_logging"
	configKeyUserAgent            = "user_agent"
	configKeyFileRoot             = "file_root"
	configKeyAllowUnsafeFilePaths = "allow_unsafe_file_paths"
)

// ClientConfig contains configuration for an S3 client
type ClientConfig struct {
	// Service configuration
	ServiceType    string // Service type (aws_s3, cloudflare_r2, etc.)
	Endpoint       string // Custom endpoint URL
	Region         string // AWS region or equivalent
	ForcePathStyle bool   // Use path-style addressing
	UseSSL         bool   // Enable/disable SSL

	// Authentication
	AccessKey    string // Access key ID
	SecretKey    string // Secret access key
	SessionToken string // Session token for temporary credentials

	// Performance and reliability
	Timeout     int   // Request timeout in seconds
	MaxRetries  int   // Maximum retry attempts
	PartSize    int64 // Multi-part upload part size in bytes
	Concurrency int   // Concurrent operations

	// Advanced options
	EnableLogging bool   // Enable request logging
	UserAgent     string // Custom user agent
}

// Validate validates the client configuration and sets defaults.
func (c *ClientConfig) Validate() error {
	if c.ServiceType == "" {
		c.ServiceType = "auto"
	}

	// if c.Region == "" {
	// 	c.Region = "us-east-1"
	// }

	if c.Timeout <= 0 {
		c.Timeout = 30
	}

	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}

	if c.PartSize <= 0 {
		c.PartSize = 5 * 1024 * 1024 // 5MB default
	}

	if c.Concurrency <= 0 {
		c.Concurrency = 3
	}

	if c.UserAgent == "" {
		c.UserAgent = "Starlark-S3/1.0"
	}

	// Auto-detect service type if needed
	if c.ServiceType == "auto" {
		c.ServiceType = c.detectServiceType()
	}

	return nil
}

// detectServiceType attempts to detect the service type from available information
func (c *ClientConfig) detectServiceType() string {
	return DetectProviderFromConfig(c)
}

// ObjectOptions contains options for object operations
type ObjectOptions struct {
	ContentType        *string
	Metadata           *map[string]string
	Tags               *map[string]string
	CacheControl       *string
	ContentEncoding    *string
	ContentDisposition *string
	ContentLanguage    *string
	Expires            *time.Time
}

// NewObjectOptions creates a new ObjectOptions instance
func NewObjectOptions() *ObjectOptions {
	return &ObjectOptions{}
}

// Validate returns true if the options contain any non-nil values
func (o *ObjectOptions) Validate() bool {
	return o.ContentType != nil ||
		o.Metadata != nil ||
		o.Tags != nil ||
		o.CacheControl != nil ||
		o.ContentEncoding != nil ||
		o.ContentDisposition != nil ||
		o.ContentLanguage != nil ||
		o.Expires != nil
}

// ApplyToPutObject applies the options to a PutObjectInput
func (o *ObjectOptions) ApplyToPutObject(input *s3.PutObjectInput) {
	if o.ContentType != nil {
		input.ContentType = o.ContentType
	}
	if o.Metadata != nil {
		input.Metadata = *o.Metadata
	}
	if o.CacheControl != nil {
		input.CacheControl = o.CacheControl
	}
	if o.ContentEncoding != nil {
		input.ContentEncoding = o.ContentEncoding
	}
	if o.ContentDisposition != nil {
		input.ContentDisposition = o.ContentDisposition
	}
	if o.ContentLanguage != nil {
		input.ContentLanguage = o.ContentLanguage
	}
	if o.Expires != nil {
		input.Expires = o.Expires
	}
	if o.Tags != nil {
		// Convert tags to URL-encoded string format
		var tagPairs []string
		for k, v := range *o.Tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		input.Tagging = aws.String(strings.Join(tagPairs, "&"))
	}
}

// ApplyToCopyObject applies the options to a CopyObjectInput and sets metadata directive
func (o *ObjectOptions) ApplyToCopyObject(input *s3.CopyObjectInput) {
	needsReplace := false

	if o.ContentType != nil {
		input.ContentType = o.ContentType
		needsReplace = true
	}
	if o.Metadata != nil {
		input.Metadata = *o.Metadata
		needsReplace = true
	}
	if o.CacheControl != nil {
		input.CacheControl = o.CacheControl
		needsReplace = true
	}
	if o.ContentEncoding != nil {
		input.ContentEncoding = o.ContentEncoding
		needsReplace = true
	}
	if o.ContentDisposition != nil {
		input.ContentDisposition = o.ContentDisposition
		needsReplace = true
	}
	if o.ContentLanguage != nil {
		input.ContentLanguage = o.ContentLanguage
		needsReplace = true
	}
	if o.Expires != nil {
		input.Expires = o.Expires
		needsReplace = true
	}
	if o.Tags != nil {
		// Convert tags to URL-encoded string format for copy operations
		var tagPairs []string
		for k, v := range *o.Tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		input.Tagging = aws.String(strings.Join(tagPairs, "&"))
		// Set TaggingDirective to REPLACE to ensure tags are replaced
		input.TaggingDirective = types.TaggingDirectiveReplace
	}

	// Only set metadata directive if we're actually changing something
	if needsReplace {
		input.MetadataDirective = types.MetadataDirectiveReplace
	}
}

// ListObjectsOptions configures ListObjects operations
type ListObjectsOptions struct {
	Prefix            *string
	Delimiter         *string
	MaxKeys           *int
	ContinuationToken *string
}

// NewListObjectsOptions creates a new ListObjectsOptions instance
func NewListObjectsOptions() *ListObjectsOptions {
	return &ListObjectsOptions{}
}

// Validate returns true if the options contain any non-nil values
func (o *ListObjectsOptions) Validate() bool {
	return o.Prefix != nil ||
		o.Delimiter != nil ||
		o.MaxKeys != nil ||
		o.ContinuationToken != nil
}

// ApplyToListObjects applies the options to a ListObjectsV2Input
func (o *ListObjectsOptions) ApplyToListObjects(input *s3.ListObjectsV2Input) {
	if o.Prefix != nil {
		input.Prefix = o.Prefix
	}
	if o.Delimiter != nil {
		input.Delimiter = o.Delimiter
	}
	if o.MaxKeys != nil {
		input.MaxKeys = aws.Int32(int32(*o.MaxKeys))
	}
	if o.ContinuationToken != nil {
		input.ContinuationToken = o.ContinuationToken
	}
}

// DetectionRule represents a rule for detecting a specific provider
type DetectionRule struct {
	// Priority determines the order of evaluation (lower = higher priority)
	Priority int
	// DetectFunc returns true if this provider matches the config
	DetectFunc func(config *ClientConfig) bool
	// Description explains what this rule detects
	Description string
}

// Detection helper functions
func hasEndpointPattern(pattern string) func(*ClientConfig) bool {
	compiledPattern := regexp.MustCompile(pattern)
	return func(config *ClientConfig) bool {
		if config.Endpoint == "" {
			return false
		}
		testURL := "https://" + config.Endpoint + "/test"
		return compiledPattern.MatchString(testURL)
	}
}

func hasRegionPattern(pattern string) func(*ClientConfig) bool {
	compiledPattern := regexp.MustCompile(pattern)
	return func(config *ClientConfig) bool {
		return config.Region != "" && compiledPattern.MatchString(config.Region)
	}
}

func hasRegionInList(regions []string) func(*ClientConfig) bool {
	return func(config *ClientConfig) bool {
		if config.Region == "" {
			return false
		}
		regionLower := strings.ToLower(config.Region)
		for _, region := range regions {
			if regionLower == strings.ToLower(region) {
				return true
			}
		}
		return false
	}
}

func hasAccessKeyPattern(pattern string) func(*ClientConfig) bool {
	compiledPattern := regexp.MustCompile(pattern)
	return func(config *ClientConfig) bool {
		return config.AccessKey != "" && compiledPattern.MatchString(config.AccessKey)
	}
}

func hasAccessKeyInList(keys []string) func(*ClientConfig) bool {
	return func(config *ClientConfig) bool {
		if config.AccessKey == "" {
			return false
		}
		for _, key := range keys {
			if config.AccessKey == key {
				return true
			}
		}
		return false
	}
}

func hasExactRegion(region string) func(*ClientConfig) bool {
	return func(config *ClientConfig) bool {
		return strings.ToLower(config.Region) == strings.ToLower(region)
	}
}

func hasEndpointContaining(substring string) func(*ClientConfig) bool {
	return func(config *ClientConfig) bool {
		return config.Endpoint != "" && strings.Contains(strings.ToLower(config.Endpoint), strings.ToLower(substring))
	}
}
