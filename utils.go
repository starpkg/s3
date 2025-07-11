package s3

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// parseS3URL parses an S3 URL and returns bucket and key components
func parseS3URL(s3URL string) (bucket, key string, err error) {
	if s3URL == "" {
		return "", "", fmt.Errorf("S3 URL cannot be empty")
	}

	// Handle s3:// format
	if strings.HasPrefix(s3URL, "s3://") {
		s3URL = strings.TrimPrefix(s3URL, "s3://")
		parts := strings.SplitN(s3URL, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			return "", "", fmt.Errorf("invalid S3 URL: missing bucket name")
		}
		bucket = parts[0]
		if len(parts) > 1 {
			key = parts[1]
		}
		return bucket, key, nil
	}

	// Handle HTTP/HTTPS URLs
	if strings.HasPrefix(s3URL, "http://") || strings.HasPrefix(s3URL, "https://") {
		u, err := url.Parse(s3URL)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}

		// For virtual-hosted-style URLs (bucket.s3.amazonaws.com)
		if strings.Contains(u.Host, ".s3.") || strings.Contains(u.Host, ".s3-") {
			parts := strings.Split(u.Host, ".")
			if len(parts) > 0 {
				bucket = parts[0]
				key = strings.TrimPrefix(u.Path, "/")
				return bucket, key, nil
			}
		}

		// For path-style URLs (s3.amazonaws.com/bucket/key)
		if strings.Contains(u.Host, "s3.") || strings.Contains(u.Host, "s3-") {
			pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				bucket = pathParts[0]
				if len(pathParts) > 1 {
					key = strings.Join(pathParts[1:], "/")
				}
				return bucket, key, nil
			}
		}

		return "", "", fmt.Errorf("unable to parse S3 URL: %s", s3URL)
	}

	return "", "", fmt.Errorf("unsupported URL format: %s", s3URL)
}

// generateS3URL generates an S3 URL from bucket and key
func generateS3URL(bucket, key string) string {
	if key == "" {
		return fmt.Sprintf("s3://%s", bucket)
	}
	return fmt.Sprintf("s3://%s/%s", bucket, key)
}

// getPublicURL generates a public HTTP URL for an S3 object
func getPublicURL(bucket, key, region, endpoint string, useSSL bool) string {
	if endpoint != "" {
		// Custom endpoint
		scheme := "https"
		if !useSSL {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, key)
	}

	// AWS S3 URL
	scheme := "https"
	if !useSSL {
		scheme = "http"
	}

	if region == "" || region == "us-east-1" {
		return fmt.Sprintf("%s://s3.amazonaws.com/%s/%s", scheme, bucket, key)
	}

	return fmt.Sprintf("%s://s3.%s.amazonaws.com/%s/%s", scheme, region, bucket, key)
}

// validateBucketName validates S3 bucket name according to AWS rules
func validateBucketName(bucket string) error {
	if bucket == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	if len(bucket) < 3 || len(bucket) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters")
	}

	// Check for valid characters and patterns
	validName := regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)
	if !validName.MatchString(bucket) {
		return fmt.Errorf("bucket name must start and end with a letter or number, and contain only lowercase letters, numbers, dots, and hyphens")
	}

	// Check for consecutive dots
	if strings.Contains(bucket, "..") {
		return fmt.Errorf("bucket name cannot contain consecutive dots")
	}

	// Check for IP address format
	ipPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	if ipPattern.MatchString(bucket) {
		return fmt.Errorf("bucket name cannot be formatted as an IP address")
	}

	// Check for invalid prefixes
	if strings.HasPrefix(bucket, "xn--") {
		return fmt.Errorf("bucket name cannot start with 'xn--'")
	}

	if strings.HasSuffix(bucket, "-s3alias") {
		return fmt.Errorf("bucket name cannot end with '-s3alias'")
	}

	return nil
}

// validateObjectKey validates S3 object key
func validateObjectKey(key string) error {
	if key == "" {
		return fmt.Errorf("object key cannot be empty")
	}

	if len(key) > 1024 {
		return fmt.Errorf("object key cannot exceed 1024 characters")
	}

	// Check for invalid characters
	invalidChars := []string{"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07", "\x08", "\x09", "\x0A", "\x0B", "\x0C", "\x0D", "\x0E", "\x0F"}
	for _, char := range invalidChars {
		if strings.Contains(key, char) {
			return fmt.Errorf("object key contains invalid control characters")
		}
	}

	return nil
}

// getSupportedServices returns a list of supported S3-compatible services
func getSupportedServices() []string {
	return []string{
		"aws",
		"minio",
		"digitalocean",
		"linode",
		"wasabi",
		"backblaze",
		"cloudflare",
		"scaleway",
		"alibaba",
		"google",
		"oracle",
		"ibm",
		"custom",
	}
}

// ServiceConfig contains configuration for known S3-compatible services
type ServiceConfig struct {
	Name            string
	DefaultRegion   string
	EndpointPattern string
	ForcePathStyle  bool
	DefaultPort     string
}

// getServiceConfig returns configuration for known S3-compatible services
func getServiceConfig(serviceType string) *ServiceConfig {
	configs := map[string]*ServiceConfig{
		"aws": {
			Name:            "Amazon S3",
			DefaultRegion:   "us-east-1",
			EndpointPattern: "s3.{region}.amazonaws.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"minio": {
			Name:            "MinIO",
			DefaultRegion:   "us-east-1",
			EndpointPattern: "localhost:9000",
			ForcePathStyle:  true,
			DefaultPort:     "9000",
		},
		"digitalocean": {
			Name:            "DigitalOcean Spaces",
			DefaultRegion:   "nyc3",
			EndpointPattern: "{region}.digitaloceanspaces.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"linode": {
			Name:            "Linode Object Storage",
			DefaultRegion:   "us-east-1",
			EndpointPattern: "{region}.linodeobjects.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"wasabi": {
			Name:            "Wasabi Hot Cloud Storage",
			DefaultRegion:   "us-east-1",
			EndpointPattern: "s3.{region}.wasabisys.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"backblaze": {
			Name:            "Backblaze B2",
			DefaultRegion:   "us-west-000",
			EndpointPattern: "s3.{region}.backblazeb2.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"cloudflare": {
			Name:            "Cloudflare R2",
			DefaultRegion:   "auto",
			EndpointPattern: "{account_id}.r2.cloudflarestorage.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"scaleway": {
			Name:            "Scaleway Object Storage",
			DefaultRegion:   "fr-par",
			EndpointPattern: "s3.{region}.scw.cloud",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"alibaba": {
			Name:            "Alibaba Cloud OSS",
			DefaultRegion:   "oss-cn-hangzhou",
			EndpointPattern: "oss-{region}.aliyuncs.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"google": {
			Name:            "Google Cloud Storage",
			DefaultRegion:   "us-central1",
			EndpointPattern: "storage.googleapis.com",
			ForcePathStyle:  true,
			DefaultPort:     "443",
		},
		"oracle": {
			Name:            "Oracle Cloud Infrastructure",
			DefaultRegion:   "us-ashburn-1",
			EndpointPattern: "{namespace}.compat.objectstorage.{region}.oraclecloud.com",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
		"ibm": {
			Name:            "IBM Cloud Object Storage",
			DefaultRegion:   "us-south",
			EndpointPattern: "s3.{region}.cloud-object-storage.appdomain.cloud",
			ForcePathStyle:  false,
			DefaultPort:     "443",
		},
	}

	if config, exists := configs[serviceType]; exists {
		return config
	}

	// Return custom configuration for unknown services
	return &ServiceConfig{
		Name:            "Custom S3 Service",
		DefaultRegion:   "us-east-1",
		EndpointPattern: "localhost:9000",
		ForcePathStyle:  true,
		DefaultPort:     "9000",
	}
}
