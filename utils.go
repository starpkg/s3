package s3

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go.starlark.net/starlark"
)

// parseS3URL parses an S3 URL and returns bucket and key components
// Supports various S3-compatible services including AWS, MinIO, DigitalOcean, Cloudflare R2, etc.
func parseS3URL(s3URL string) (bucket, key string, err error) {
	if s3URL == "" {
		return "", "", fmt.Errorf("S3 URL cannot be empty")
	}

	// Handle s3:// format (standard across all providers)
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

	// Handle HTTP/HTTPS URLs for various providers
	if strings.HasPrefix(s3URL, "http://") || strings.HasPrefix(s3URL, "https://") {
		u, err := url.Parse(s3URL)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}

		// AWS S3 patterns
		if strings.Contains(u.Host, ".s3.amazonaws.com") || strings.Contains(u.Host, ".s3-") ||
			(strings.Contains(u.Host, ".s3.") && strings.Contains(u.Host, ".amazonaws.com")) {
			// Virtual-hosted-style: bucket.s3.amazonaws.com, bucket.s3-region.amazonaws.com, or bucket.s3.region.amazonaws.com
			if strings.Contains(u.Host, ".s3.") || strings.Contains(u.Host, ".s3-") {
				parts := strings.Split(u.Host, ".")
				if len(parts) > 0 {
					bucket = parts[0]
					key = strings.TrimPrefix(u.Path, "/")
					return bucket, key, nil
				}
			}
		}

		// AWS S3 path-style: s3.amazonaws.com/bucket/key or s3.region.amazonaws.com/bucket/key
		if strings.Contains(u.Host, "s3.amazonaws.com") {
			pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				bucket = pathParts[0]
				if len(pathParts) > 1 {
					key = strings.Join(pathParts[1:], "/")
				}
				return bucket, key, nil
			}
		}

		// DigitalOcean Spaces: bucket.region.digitaloceanspaces.com/key
		if strings.Contains(u.Host, ".digitaloceanspaces.com") {
			parts := strings.Split(u.Host, ".")
			if len(parts) >= 3 {
				bucket = parts[0]
				key = strings.TrimPrefix(u.Path, "/")
				return bucket, key, nil
			}
		}

		// Linode Object Storage: bucket.region.linodeobjects.com/key
		if strings.Contains(u.Host, ".linodeobjects.com") {
			parts := strings.Split(u.Host, ".")
			if len(parts) >= 3 {
				bucket = parts[0]
				key = strings.TrimPrefix(u.Path, "/")
				return bucket, key, nil
			}
		}

		// Wasabi: bucket.s3.region.wasabisys.com/key or s3.region.wasabisys.com/bucket/key
		if strings.Contains(u.Host, ".wasabisys.com") {
			if strings.HasPrefix(u.Host, "s3.") {
				// Path-style
				pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
				if len(pathParts) > 0 && pathParts[0] != "" {
					bucket = pathParts[0]
					if len(pathParts) > 1 {
						key = strings.Join(pathParts[1:], "/")
					}
					return bucket, key, nil
				}
			} else {
				// Virtual-hosted-style
				parts := strings.Split(u.Host, ".")
				if len(parts) > 0 {
					bucket = parts[0]
					key = strings.TrimPrefix(u.Path, "/")
					return bucket, key, nil
				}
			}
		}

		// Backblaze B2: bucket.s3.region.backblazeb2.com/key or s3.region.backblazeb2.com/bucket/key
		if strings.Contains(u.Host, ".backblazeb2.com") {
			if strings.HasPrefix(u.Host, "s3.") {
				// Path-style
				pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
				if len(pathParts) > 0 && pathParts[0] != "" {
					bucket = pathParts[0]
					if len(pathParts) > 1 {
						key = strings.Join(pathParts[1:], "/")
					}
					return bucket, key, nil
				}
			} else {
				// Virtual-hosted-style
				parts := strings.Split(u.Host, ".")
				if len(parts) > 0 {
					bucket = parts[0]
					key = strings.TrimPrefix(u.Path, "/")
					return bucket, key, nil
				}
			}
		}

		// Cloudflare R2: account_id.r2.cloudflarestorage.com/bucket/key
		if strings.Contains(u.Host, ".r2.cloudflarestorage.com") {
			pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				bucket = pathParts[0]
				if len(pathParts) > 1 {
					key = strings.Join(pathParts[1:], "/")
				}
				return bucket, key, nil
			}
		}

		// Scaleway: bucket.s3.region.scw.cloud/key or s3.region.scw.cloud/bucket/key
		if strings.Contains(u.Host, ".scw.cloud") {
			if strings.HasPrefix(u.Host, "s3.") {
				// Path-style
				pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
				if len(pathParts) > 0 && pathParts[0] != "" {
					bucket = pathParts[0]
					if len(pathParts) > 1 {
						key = strings.Join(pathParts[1:], "/")
					}
					return bucket, key, nil
				}
			} else {
				// Virtual-hosted-style
				parts := strings.Split(u.Host, ".")
				if len(parts) > 0 {
					bucket = parts[0]
					key = strings.TrimPrefix(u.Path, "/")
					return bucket, key, nil
				}
			}
		}

		// Google Cloud Storage: storage.googleapis.com/bucket/key
		if strings.Contains(u.Host, "storage.googleapis.com") {
			pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
			if len(pathParts) > 0 && pathParts[0] != "" {
				bucket = pathParts[0]
				if len(pathParts) > 1 {
					key = strings.Join(pathParts[1:], "/")
				}
				return bucket, key, nil
			}
		}

		// Alibaba Cloud OSS: bucket.oss-region.aliyuncs.com/key or oss-region.aliyuncs.com/bucket/key
		if strings.Contains(u.Host, ".aliyuncs.com") {
			if strings.Contains(u.Host, "oss-") && !strings.HasPrefix(u.Host, "oss-") {
				// Virtual-hosted-style: bucket.oss-region.aliyuncs.com
				parts := strings.Split(u.Host, ".")
				if len(parts) > 0 {
					bucket = parts[0]
					key = strings.TrimPrefix(u.Path, "/")
					return bucket, key, nil
				}
			} else if strings.HasPrefix(u.Host, "oss-") {
				// Path-style: oss-region.aliyuncs.com/bucket/key
				pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
				if len(pathParts) > 0 && pathParts[0] != "" {
					bucket = pathParts[0]
					if len(pathParts) > 1 {
						key = strings.Join(pathParts[1:], "/")
					}
					return bucket, key, nil
				}
			}
		}

		// MinIO or custom endpoints: try path-style first
		pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(pathParts) > 0 && pathParts[0] != "" {
			bucket = pathParts[0]
			if len(pathParts) > 1 {
				key = strings.Join(pathParts[1:], "/")
			}
			return bucket, key, nil
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
func getPublicURL(bucket, key, region, endpoint string, useSSL bool, serviceType string) string {
	scheme := "https"
	if !useSSL {
		scheme = "http"
	}

	// If endpoint is explicitly provided, use it directly
	if endpoint != "" {
		// For path-style addressing
		return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, key)
	}

	// Use service configuration if available
	if serviceType != "" && serviceType != "custom" {
		serviceConfig := getServiceConfig(serviceType)
		if serviceConfig != nil {
			endpointPattern := serviceConfig.EndpointPattern

			// Handle different endpoint patterns
			switch serviceType {
			case "aws":
				if region == "" || region == "us-east-1" {
					if serviceConfig.ForcePathStyle {
						return fmt.Sprintf("%s://s3.amazonaws.com/%s/%s", scheme, bucket, key)
					}
					return fmt.Sprintf("%s://s3.amazonaws.com/%s/%s", scheme, bucket, key)
				}
				// Replace {region} placeholder
				endpointPattern = strings.ReplaceAll(endpointPattern, "{region}", region)
				if serviceConfig.ForcePathStyle {
					return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)
				}
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)

			case "digitalocean", "linode", "wasabi", "backblaze", "scaleway":
				// Replace {region} placeholder
				endpointPattern = strings.ReplaceAll(endpointPattern, "{region}", region)
				if serviceConfig.ForcePathStyle {
					return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)
				}
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)

			case "cloudflare":
				// For Cloudflare R2, we typically use the account-specific endpoint provided
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)

			case "google":
				// Google Cloud Storage uses path-style addressing
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)

			case "alibaba":
				// Replace {region} placeholder
				endpointPattern = strings.ReplaceAll(endpointPattern, "{region}", region)
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)

			case "minio":
				// MinIO typically uses path-style addressing with custom endpoints
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)

			default:
				// Default to path-style addressing
				endpointPattern = strings.ReplaceAll(endpointPattern, "{region}", region)
				return fmt.Sprintf("%s://%s/%s/%s", scheme, endpointPattern, bucket, key)
			}
		}
	}

	// Fallback to AWS S3 format if service type is unknown
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

// convertStarlarkDict converts a Starlark dictionary to a Go map[string]string
func convertStarlarkDict(dict *starlark.Dict) (map[string]string, error) {
	if dict == nil || dict.Len() == 0 {
		return nil, nil
	}

	result := make(map[string]string, dict.Len())
	for _, item := range dict.Items() {
		keyStr, ok := item[0].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("dictionary key must be a string, got %T", item[0])
		}

		valueStr, ok := item[1].(starlark.String)
		if !ok {
			return nil, fmt.Errorf("dictionary value must be a string, got %T", item[1])
		}

		result[keyStr.GoString()] = valueStr.GoString()
	}
	return result, nil
}

// convertStarlarkStringToTime converts a string to time.Time using RFC3339 format
func convertStarlarkStringToTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time format, expected RFC3339: %w", err)
	}
	return t, nil
}

// convertStringToBytes converts a string to []byte
func convertStringToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

// convertMetadataDict converts a Starlark dictionary to metadata map
func convertMetadataDict(dict *starlark.Dict) (map[string]string, error) {
	return convertStarlarkDict(dict)
}

// convertTagsDict converts a Starlark dictionary to tags map
func convertTagsDict(dict *starlark.Dict) (map[string]string, error) {
	return convertStarlarkDict(dict)
}

// validateNonEmptyString validates that a string is not empty
func validateNonEmptyString(s string, fieldName string) error {
	if s == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// validateStringLength validates that a string meets length requirements
func validateStringLength(s string, fieldName string, minLen, maxLen int) error {
	if len(s) < minLen {
		return fmt.Errorf("%s must be at least %d characters long", fieldName, minLen)
	}
	if len(s) > maxLen {
		return fmt.Errorf("%s must be no more than %d characters long", fieldName, maxLen)
	}
	return nil
}

// convertOptionalString converts a string parameter that might be empty
func convertOptionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// convertOptionalInt converts an integer parameter that might be zero
func convertOptionalInt(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// convertOptionalInt64 converts an int64 parameter that might be zero
func convertOptionalInt64(i int64) *int64 {
	if i == 0 {
		return nil
	}
	return &i
}

// convertOptionalBool converts a boolean parameter with explicit flag
func convertOptionalBool(b bool, hasValue bool) *bool {
	if !hasValue {
		return nil
	}
	return &b
}

// detectServiceTypeFromURL detects the S3-compatible service type from URL patterns
func detectServiceTypeFromURL(s3URL string) string {
	if s3URL == "" {
		return "unknown"
	}

	// s3:// URLs don't contain service-specific information
	if strings.HasPrefix(s3URL, "s3://") {
		return "unknown"
	}

	// Parse HTTP/HTTPS URLs to detect service
	if strings.HasPrefix(s3URL, "http://") || strings.HasPrefix(s3URL, "https://") {
		u, err := url.Parse(s3URL)
		if err != nil {
			return "unknown"
		}

		host := strings.ToLower(u.Host)

		// AWS S3
		if strings.Contains(host, ".s3.amazonaws.com") || strings.Contains(host, "s3.amazonaws.com") ||
			strings.Contains(host, ".s3-") || (strings.Contains(host, ".s3.") && strings.Contains(host, ".amazonaws.com")) ||
			(strings.HasPrefix(host, "s3.") && strings.Contains(host, ".amazonaws.com")) {
			return "aws"
		}

		// DigitalOcean Spaces
		if strings.Contains(host, ".digitaloceanspaces.com") {
			return "digitalocean"
		}

		// Linode Object Storage
		if strings.Contains(host, ".linodeobjects.com") {
			return "linode"
		}

		// Wasabi
		if strings.Contains(host, ".wasabisys.com") {
			return "wasabi"
		}

		// Backblaze B2
		if strings.Contains(host, ".backblazeb2.com") {
			return "backblaze"
		}

		// Cloudflare R2
		if strings.Contains(host, ".r2.cloudflarestorage.com") {
			return "cloudflare"
		}

		// Scaleway
		if strings.Contains(host, ".scw.cloud") {
			return "scaleway"
		}

		// Google Cloud Storage
		if strings.Contains(host, "storage.googleapis.com") {
			return "google"
		}

		// Alibaba Cloud OSS
		if strings.Contains(host, ".aliyuncs.com") {
			return "alibaba"
		}

		// Check for localhost or IP addresses (likely MinIO)
		if strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
			return "minio"
		}

		// Oracle Cloud Infrastructure Object Storage
		if strings.Contains(host, ".oraclecloud.com") {
			return "oracle"
		}

		// IBM Cloud Object Storage
		if strings.Contains(host, ".cloud.ibm.com") {
			return "ibm"
		}
	}

	// Default to unknown if we can't detect the service
	return "unknown"
}
