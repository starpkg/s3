package s3

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.starlark.net/starlark"
)

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

// ServiceConfig contains configuration for known S3-compatible services
type ServiceConfig struct {
	Name            string
	DefaultRegion   string
	EndpointPattern string
	ForcePathStyle  bool
	DefaultPort     string
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

// convertMetadataDict converts a Starlark dictionary to metadata map
func convertMetadataDict(dict *starlark.Dict) (map[string]string, error) {
	return convertStarlarkDict(dict)
}
