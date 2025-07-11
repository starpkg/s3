package s3

import (
	"time"
)

// Configuration key constants
const (
	configKeyServiceType    = "service_type"
	configKeyAccessKey      = "access_key"
	configKeySecretKey      = "secret_key"
	configKeySessionToken   = "session_token"
	configKeyRegion         = "region"
	configKeyEndpoint       = "endpoint"
	configKeyForcePathStyle = "force_path_style"
	configKeyUseSSL         = "use_ssl"
	configKeyTimeout        = "timeout"
	configKeyMaxRetries     = "max_retries"
	configKeyPartSize       = "part_size"
	configKeyConcurrency    = "concurrency"
	configKeyEnableLogging  = "enable_logging"
	configKeyUserAgent      = "user_agent"
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

// ValidateConfig validates the client configuration
func (c *ClientConfig) ValidateConfig() error {
	if c.ServiceType == "" {
		c.ServiceType = "auto"
	}

	if c.Region == "" {
		c.Region = "us-east-1"
	}

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
		c.UserAgent = "starlark-s3/1.0"
	}

	// Auto-detect service type if needed
	if c.ServiceType == "auto" {
		c.ServiceType = c.detectServiceType()
	}

	return nil
}

// detectServiceType attempts to detect the service type from the endpoint
func (c *ClientConfig) detectServiceType() string {
	if c.Endpoint == "" {
		return "aws_s3"
	}

	endpoint := c.Endpoint
	switch {
	case contains(endpoint, "amazonaws.com"):
		return "aws_s3"
	case contains(endpoint, "r2.cloudflarestorage.com"):
		return "cloudflare_r2"
	case contains(endpoint, "backblazeb2.com"):
		return "backblaze_b2"
	case contains(endpoint, "digitaloceanspaces.com"):
		return "digitalocean_spaces"
	case contains(endpoint, "localhost") || contains(endpoint, "127.0.0.1"):
		return "minio"
	default:
		return "aws_s3"
	}
}

// contains checks if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		(len(str) > len(substr) &&
			(str[:len(substr)] == substr ||
				str[len(str)-len(substr):] == substr ||
				indexOf(str, substr) >= 0)))
}

// indexOf returns the index of the first occurrence of substr in str, or -1 if not found
func indexOf(str, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(str) < len(substr) {
		return -1
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetTimeout returns the timeout as a time.Duration
func (c *ClientConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// GetEndpointURL returns the endpoint URL, generating one if not set
func (c *ClientConfig) GetEndpointURL() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}

	// Generate default endpoint based on service type
	switch c.ServiceType {
	case "aws_s3":
		return "" // Use default AWS SDK endpoint
	case "cloudflare_r2":
		return "" // R2 requires custom endpoint
	case "digitalocean_spaces":
		return "https://" + c.Region + ".digitaloceanspaces.com"
	case "backblaze_b2":
		return "https://s3." + c.Region + ".backblazeb2.com"
	case "minio":
		return "http://localhost:9000"
	default:
		return ""
	}
}
