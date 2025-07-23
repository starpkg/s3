// Package s3 provides constants and configuration for S3-compatible service providers
package s3

import (
	"fmt"
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

	// Detection rules for smart provider detection
	DetectionRules []DetectionRule
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
	providers := make([]string, 0, len(providerOrder))
	for _, name := range providerOrder {
		if name != ProviderCustom {
			providers = append(providers, name)
		}
	}
	return providers
}

// DetectProviderFromConfig uses pluggable detection rules to identify the best provider
func DetectProviderFromConfig(config *ClientConfig) string {
	// Collect all detection rules from all providers
	type providerRule struct {
		provider string
		rule     DetectionRule
	}

	var allRules []providerRule
	for _, providerName := range providerOrder {
		if providerConfig, exists := providerConfigs[providerName]; exists {
			for _, rule := range providerConfig.DetectionRules {
				allRules = append(allRules, providerRule{
					provider: providerName,
					rule:     rule,
				})
			}
		}
	}

	// Sort by priority (lower priority number = higher priority)
	for i := 0; i < len(allRules)-1; i++ {
		for j := i + 1; j < len(allRules); j++ {
			if allRules[i].rule.Priority > allRules[j].rule.Priority {
				allRules[i], allRules[j] = allRules[j], allRules[i]
			}
		}
	}

	// Test rules in priority order
	for _, pr := range allRules {
		if pr.rule.DetectFunc(config) {
			return pr.provider
		}
	}

	return ProviderCustom
}

// GenerateURLWithProvider generates a public URL using provider-specific logic
func GenerateURLWithProvider(bucket, key, region, endpoint string, useSSL bool, provider string) string {
	config := GetProviderConfig(provider)

	// If a custom endpoint is provided, use it
	if endpoint != "" {
		// Check if endpoint already includes scheme
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			// Endpoint already has scheme, use it directly
			return fmt.Sprintf("%s/%s/%s", endpoint, bucket, key)
		} else {
			// Endpoint doesn't have scheme, add it
			scheme := "https"
			if !useSSL {
				scheme = "http"
			}
			return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, bucket, key)
		}
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

// providerOrder defines the order in which providers should be checked
// Specific providers first, fallback providers last
var providerOrder = []string{
	ProviderAWS,
	ProviderDigitalOcean,
	ProviderLinode,
	ProviderWasabi,
	ProviderBackblaze,
	ProviderCloudflare,
	ProviderScaleway,
	ProviderAlibaba,
	ProviderGoogle,
	ProviderOracle,
	ProviderIBM,
	// Fallback providers - these should be checked last
	ProviderMinIO,
	ProviderCustom,
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
		DetectionRules: []DetectionRule{
			{
				Priority:    10,
				DetectFunc:  hasEndpointContaining("amazonaws.com"),
				Description: "AWS S3 endpoint detected",
			},
			{
				Priority:    20,
				DetectFunc:  hasAccessKeyPattern(`^(?i)AKIA[A-Z0-9]{16}$`),
				Description: "AWS access key pattern (AKIA) detected",
			},
			{
				Priority:    21,
				DetectFunc:  hasAccessKeyPattern(`^(?i)ASIA[A-Z0-9]{16}$`),
				Description: "AWS temporary access key pattern (ASIA) detected",
			},
			{
				Priority:    30,
				DetectFunc:  hasRegionPattern(`^[a-z]{2,3}-[a-z]+-\d+$`),
				Description: "AWS region format detected",
			},
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
				// IP addresses with port
				Pattern:   regexp.MustCompile(`^https?://[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}:[0-9]+/`),
				ParseFunc: parsePathStyleURL,
			},
			{
				// Domain names with port
				Pattern:   regexp.MustCompile(`^https?://[^/]+:[0-9]+/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "localhost:9000", true)
		},
		DetectionRules: []DetectionRule{
			{
				Priority:    40,
				DetectFunc:  hasAccessKeyInList([]string{"minioadmin", "minio"}),
				Description: "MinIO default access key detected",
			},
			{
				Priority:    50,
				DetectFunc:  hasEndpointPattern(`^https?://[^/]+:[0-9]+/`),
				Description: "MinIO endpoint with port detected",
			},
			{
				Priority:    51,
				DetectFunc:  hasEndpointContaining("localhost"),
				Description: "MinIO localhost endpoint detected",
			},
			{
				Priority:    52,
				DetectFunc:  hasEndpointContaining("min.io"),
				Description: "MinIO domain detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("digitaloceanspaces.com"),
				Description: "DigitalOcean Spaces endpoint detected",
			},
			{
				Priority:    35,
				DetectFunc:  hasRegionInList([]string{"nyc1", "nyc2", "nyc3", "ams2", "ams3", "sfo1", "sfo2", "sfo3", "sgp1", "lon1", "fra1", "tor1", "blr1"}),
				Description: "DigitalOcean Spaces region detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("linodeobjects.com"),
				Description: "Linode Object Storage endpoint detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("wasabisys.com"),
				Description: "Wasabi Hot Cloud Storage endpoint detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("backblazeb2.com"),
				Description: "Backblaze B2 endpoint detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("r2.cloudflarestorage.com"),
				Description: "Cloudflare R2 endpoint detected",
			},
			{
				Priority:    15,
				DetectFunc:  hasExactRegion("auto"),
				Description: "Cloudflare R2 auto region detected",
			},
			{
				Priority:    25,
				DetectFunc:  hasAccessKeyPattern(`^[0-9a-fA-F]{32}$`),
				Description: "Cloudflare R2 access key pattern (32-char hex) detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("scw.cloud"),
				Description: "Scaleway Object Storage endpoint detected",
			},
		},
	},

	ProviderAlibaba: {
		Name:                  ProviderAlibaba,
		DisplayName:           "Alibaba Cloud OSS",
		DefaultRegion:         "cn-hangzhou",
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("aliyuncs.com"),
				Description: "Alibaba Cloud OSS endpoint detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("googleapis.com"),
				Description: "Google Cloud Storage endpoint detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("oraclecloud.com"),
				Description: "Oracle Cloud Infrastructure endpoint detected",
			},
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
		DetectionRules: []DetectionRule{
			{
				Priority:    5,
				DetectFunc:  hasEndpointContaining("cloud-object-storage.appdomain.cloud"),
				Description: "IBM Cloud Object Storage endpoint detected",
			},
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
				// Catch-all pattern for any HTTP/HTTPS URL
				Pattern:   regexp.MustCompile(`^https?://[^/]+/`),
				ParseFunc: parsePathStyleURL,
			},
		},
		GenerateURL: func(bucket, key, region, endpoint string, useSSL bool) string {
			return generateStandardURL(bucket, key, region, endpoint, useSSL, "localhost:9000", true)
		},
	},
}
