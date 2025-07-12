// Package s3 provides constants and configuration for S3-compatible service providers
package s3

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// S3-compatible service provider constants
const (
	// ProviderAWS is Amazon S3
	ProviderAWS = "aws"
	// ProviderMinIO is MinIO
	ProviderMinIO = "minio"
	// ProviderDigitalOcean is DigitalOcean Spaces
	ProviderDigitalOcean = "digitalocean"
	// ProviderLinode is Linode Object Storage
	ProviderLinode = "linode"
	// ProviderWasabi is Wasabi Hot Cloud Storage
	ProviderWasabi = "wasabi"
	// ProviderBackblaze is Backblaze B2
	ProviderBackblaze = "backblaze"
	// ProviderCloudflare is Cloudflare R2
	ProviderCloudflare = "cloudflare"
	// ProviderScaleway is Scaleway Object Storage
	ProviderScaleway = "scaleway"
	// ProviderAlibaba is Alibaba Cloud OSS
	ProviderAlibaba = "alibaba"
	// ProviderGoogle is Google Cloud Storage
	ProviderGoogle = "google"
	// ProviderOracle is Oracle Cloud Infrastructure
	ProviderOracle = "oracle"
	// ProviderIBM is IBM Cloud Object Storage
	ProviderIBM = "ibm"
	// ProviderUnknown is for unknown providers (e.g., s3:// URLs)
	ProviderUnknown = "unknown"
	// ProviderCustom is for custom providers
	ProviderCustom = "custom"
)

// URLStyle represents different URL addressing styles
type URLStyle int

const (
	// URLStyleVirtualHosted uses virtual-hosted-style URLs: bucket.s3.amazonaws.com/key
	URLStyleVirtualHosted URLStyle = iota
	// URLStylePath uses path-style URLs: s3.amazonaws.com/bucket/key
	URLStylePath
	// URLStyleBoth supports both virtual-hosted and path-style URLs
	URLStyleBoth
)

// URLPattern represents a URL pattern for parsing or generating URLs
type URLPattern struct {
	// Pattern is a regexp pattern for matching URLs
	Pattern *regexp.Regexp
	// ParseFunc extracts bucket and key from URL components
	ParseFunc func(host, path string) (bucket, key string, ok bool)
	// GenerateFunc generates a URL from bucket, key, and other parameters
	GenerateFunc func(bucket, key, region, endpoint string, useSSL bool) string
}

// ProviderConfig contains comprehensive configuration for S3-compatible service providers
type ProviderConfig struct {
	// Basic information
	Name          string
	DisplayName   string
	DefaultRegion string
	DefaultPort   string

	// Connection settings
	ForcePathStyle bool
	URLStyle       URLStyle

	// Endpoint configuration
	EndpointPattern string

	// URL patterns for parsing different URL formats
	URLPatterns []URLPattern

	// URL generation function
	GenerateURL func(bucket, key, region, endpoint string, useSSL bool) string

	// Provider-specific settings
	SupportsVirtualHosted bool
	SupportsPathStyle     bool
	RequiresAccountID     bool
	RequiresNamespace     bool
}

// GetProviderConfig returns the configuration for a specific provider
func GetProviderConfig(provider string) *ProviderConfig {
	if config, exists := providerConfigs[provider]; exists {
		return config
	}
	return providerConfigs[ProviderCustom]
}

// GetAllProviders returns a list of all supported provider names
func GetAllProviders() []string {
	providers := make([]string, 0, len(providerConfigs))
	for name := range providerConfigs {
		if name != ProviderCustom && name != ProviderUnknown {
			providers = append(providers, name)
		}
	}
	return providers
}

// DetectProviderFromURL attempts to detect the provider from a URL
func DetectProviderFromURL(s3URL string) string {
	for provider, config := range providerConfigs {
		if provider == ProviderCustom {
			continue
		}

		for _, pattern := range config.URLPatterns {
			if pattern.Pattern.MatchString(s3URL) {
				return provider
			}
		}
	}
	return ProviderCustom
}

// ParseURLWithProvider parses an S3 URL using provider-specific logic
func ParseURLWithProvider(s3URL string, provider string) (bucket, key string, detectedProvider string, err error) {
	// Handle s3:// format (universal)
	if strings.HasPrefix(s3URL, "s3://") {
		s3URL = strings.TrimPrefix(s3URL, "s3://")
		parts := strings.SplitN(s3URL, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			return "", "", "", fmt.Errorf("invalid S3 URL: missing bucket name")
		}
		bucket = parts[0]
		if len(parts) > 1 {
			key = parts[1]
		}
		// s3:// URLs don't contain provider-specific information
		detectedProvider = ProviderUnknown
		if provider != "" {
			detectedProvider = provider
		}
		return bucket, key, detectedProvider, nil
	}

	// If provider is specified, use its patterns first
	if provider != "" && provider != ProviderCustom {
		config := GetProviderConfig(provider)
		bucket, key, err := parseURLWithConfig(s3URL, config)
		if err == nil {
			return bucket, key, provider, nil
		}
	}

	// Try to detect provider and parse
	detectedProvider = DetectProviderFromURL(s3URL)
	config := GetProviderConfig(detectedProvider)
	bucket, key, err = parseURLWithConfig(s3URL, config)
	if err != nil {
		return "", "", detectedProvider, err
	}

	return bucket, key, detectedProvider, nil
}

// parseURLWithConfig parses a URL using a specific provider configuration
func parseURLWithConfig(s3URL string, config *ProviderConfig) (bucket, key string, err error) {
	if !strings.HasPrefix(s3URL, "http://") && !strings.HasPrefix(s3URL, "https://") {
		return "", "", fmt.Errorf("unsupported URL format: %s", s3URL)
	}

	u, err := url.Parse(s3URL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	// Try each URL pattern for this provider
	for _, pattern := range config.URLPatterns {
		if pattern.Pattern.MatchString(s3URL) {
			bucket, key, ok := pattern.ParseFunc(u.Host, u.Path)
			if ok {
				return bucket, key, nil
			}
		}
	}

	return "", "", fmt.Errorf("unable to parse URL with provider %s: %s", config.Name, s3URL)
}

// GenerateURLWithProvider generates a public URL using provider-specific logic
func GenerateURLWithProvider(bucket, key, region, endpoint string, useSSL bool, provider string) string {
	config := GetProviderConfig(provider)

	// If a custom endpoint is provided, use it
	if endpoint != "" {
		scheme := "https"
		if !useSSL {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, key)
	}

	// Use provider-specific URL generation
	return config.GenerateURL(bucket, key, region, endpoint, useSSL)
}

// Helper functions for common URL patterns

// parseVirtualHostedURL parses virtual-hosted-style URLs
func parseVirtualHostedURL(host, path string) (bucket, key string, ok bool) {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", "", false
	}
	bucket = parts[0]
	key = strings.TrimPrefix(path, "/")
	return bucket, key, true
}

// parsePathStyleURL parses path-style URLs
func parsePathStyleURL(host, path string) (bucket, key string, ok bool) {
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(pathParts) < 1 || pathParts[0] == "" {
		return "", "", false
	}
	bucket = pathParts[0]
	if len(pathParts) > 1 {
		key = strings.Join(pathParts[1:], "/")
	}
	return bucket, key, true
}

// generateStandardURL generates a standard S3-style URL
func generateStandardURL(bucket, key, region, endpoint string, useSSL bool, endpointPattern string, forcePathStyle bool) string {
	scheme := "https"
	if !useSSL {
		scheme = "http"
	}

	// Replace region placeholder
	finalEndpoint := strings.ReplaceAll(endpointPattern, "{region}", region)

	if forcePathStyle {
		return fmt.Sprintf("%s://%s/%s/%s", scheme, finalEndpoint, bucket, key)
	}

	// Virtual-hosted style
	return fmt.Sprintf("%s://%s.%s/%s", scheme, bucket, finalEndpoint, key)
}

// Provider configurations
var providerConfigs = map[string]*ProviderConfig{
	ProviderAWS: {
		Name:                  ProviderAWS,
		DisplayName:           "Amazon S3",
		DefaultRegion:         "us-east-1",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleBoth,
		EndpointPattern:       "s3.{region}.amazonaws.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.amazonaws\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3-[^/]+\.amazonaws\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.[^/]+\.amazonaws\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://s3\.amazonaws\.com/`),
				ParseFunc: parsePathStyleURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://s3\.[^/]+\.amazonaws\.com/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			if region == "" || region == "us-east-1" {
				return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.amazonaws.com", false)
			}
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.amazonaws.com", false)
		},
	},

	ProviderMinIO: {
		Name:                  ProviderMinIO,
		DisplayName:           "MinIO",
		DefaultRegion:         "us-east-1",
		DefaultPort:           "9000",
		ForcePathStyle:        true,
		URLStyle:              URLStylePath,
		EndpointPattern:       "localhost:9000",
		SupportsVirtualHosted: false,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://localhost:[0-9]+/`),
				ParseFunc: parsePathStyleURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+:[0-9]+/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "localhost:9000", true)
		},
	},

	ProviderDigitalOcean: {
		Name:                  ProviderDigitalOcean,
		DisplayName:           "DigitalOcean Spaces",
		DefaultRegion:         "nyc3",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleVirtualHosted,
		EndpointPattern:       "{region}.digitaloceanspaces.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     false,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.[^/]+\.digitaloceanspaces\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "{region}.digitaloceanspaces.com", false)
		},
	},

	ProviderLinode: {
		Name:                  ProviderLinode,
		DisplayName:           "Linode Object Storage",
		DefaultRegion:         "us-east-1",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleVirtualHosted,
		EndpointPattern:       "{region}.linodeobjects.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     false,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.[^/]+\.linodeobjects\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "{region}.linodeobjects.com", false)
		},
	},

	ProviderWasabi: {
		Name:                  ProviderWasabi,
		DisplayName:           "Wasabi Hot Cloud Storage",
		DefaultRegion:         "us-east-1",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleBoth,
		EndpointPattern:       "s3.{region}.wasabisys.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.[^/]+\.wasabisys\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://s3\.[^/]+\.wasabisys\.com/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.wasabisys.com", false)
		},
	},

	ProviderBackblaze: {
		Name:                  ProviderBackblaze,
		DisplayName:           "Backblaze B2",
		DefaultRegion:         "us-west-000",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleBoth,
		EndpointPattern:       "s3.{region}.backblazeb2.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.[^/]+\.backblazeb2\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://s3\.[^/]+\.backblazeb2\.com/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.backblazeb2.com", false)
		},
	},

	ProviderCloudflare: {
		Name:                  ProviderCloudflare,
		DisplayName:           "Cloudflare R2",
		DefaultRegion:         "auto",
		DefaultPort:           "443",
		ForcePathStyle:        true,
		URLStyle:              URLStylePath,
		EndpointPattern:       "{account_id}.r2.cloudflarestorage.com",
		SupportsVirtualHosted: false,
		SupportsPathStyle:     true,
		RequiresAccountID:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.r2\.cloudflarestorage\.com/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "{account_id}.r2.cloudflarestorage.com", true)
		},
	},

	ProviderScaleway: {
		Name:                  ProviderScaleway,
		DisplayName:           "Scaleway Object Storage",
		DefaultRegion:         "fr-par",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleBoth,
		EndpointPattern:       "s3.{region}.scw.cloud",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.[^/]+\.scw\.cloud/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://s3\.[^/]+\.scw\.cloud/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.scw.cloud", false)
		},
	},

	ProviderAlibaba: {
		Name:                  ProviderAlibaba,
		DisplayName:           "Alibaba Cloud OSS",
		DefaultRegion:         "oss-cn-hangzhou",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleBoth,
		EndpointPattern:       "oss-{region}.aliyuncs.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.oss-[^/]+\.aliyuncs\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
			{
				Pattern:   regexp.MustCompile(`^https?://oss-[^/]+\.aliyuncs\.com/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "oss-{region}.aliyuncs.com", false)
		},
	},

	ProviderGoogle: {
		Name:                  ProviderGoogle,
		DisplayName:           "Google Cloud Storage",
		DefaultRegion:         "us-central1",
		DefaultPort:           "443",
		ForcePathStyle:        true,
		URLStyle:              URLStylePath,
		EndpointPattern:       "storage.googleapis.com",
		SupportsVirtualHosted: false,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://storage\.googleapis\.com/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "storage.googleapis.com", true)
		},
	},

	ProviderOracle: {
		Name:                  ProviderOracle,
		DisplayName:           "Oracle Cloud Infrastructure",
		DefaultRegion:         "us-ashburn-1",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleVirtualHosted,
		EndpointPattern:       "{namespace}.compat.objectstorage.{region}.oraclecloud.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     false,
		RequiresNamespace:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.compat\.objectstorage\.[^/]+\.oraclecloud\.com/`),
				ParseFunc: parseVirtualHostedURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "{namespace}.compat.objectstorage.{region}.oraclecloud.com", false)
		},
	},

	ProviderIBM: {
		Name:                  ProviderIBM,
		DisplayName:           "IBM Cloud Object Storage",
		DefaultRegion:         "us-south",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStyleVirtualHosted,
		EndpointPattern:       "s3.{region}.cloud-object-storage.appdomain.cloud",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     false,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+\.s3\.[^/]+\.cloud-object-storage\.appdomain\.cloud/`),
				ParseFunc: parseVirtualHostedURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.cloud-object-storage.appdomain.cloud", false)
		},
	},

	ProviderUnknown: {
		Name:                  ProviderUnknown,
		DisplayName:           "Unknown S3 Service",
		DefaultRegion:         "us-east-1",
		DefaultPort:           "443",
		ForcePathStyle:        false,
		URLStyle:              URLStylePath,
		EndpointPattern:       "s3.amazonaws.com",
		SupportsVirtualHosted: true,
		SupportsPathStyle:     true,
		URLPatterns:           []URLPattern{},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			// Default to AWS S3 format for unknown providers
			if region == "" || region == "us-east-1" {
				return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.amazonaws.com", false)
			}
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "s3.{region}.amazonaws.com", false)
		},
	},

	ProviderCustom: {
		Name:                  ProviderCustom,
		DisplayName:           "Custom S3 Service",
		DefaultRegion:         "us-east-1",
		DefaultPort:           "9000",
		ForcePathStyle:        true,
		URLStyle:              URLStylePath,
		EndpointPattern:       "localhost:9000",
		SupportsVirtualHosted: false,
		SupportsPathStyle:     true,
		URLPatterns: []URLPattern{
			{
				Pattern:   regexp.MustCompile(`^https?://[^/]+/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "localhost:9000", true)
		},
	},
}
