// Package s3 provides a Starlark module for S3-compatible storage operations.
package s3

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv"
	"github.com/1set/starlet/dataconv/types"
	"github.com/starpkg/base"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ModuleName defines the expected name for this module when used in Starlark's load() function
const ModuleName = "s3"

var (
	none = starlark.None
)

// Module wraps the ConfigurableModule with specific functionality for S3 operations
type Module struct {
	cfgMod *base.ConfigurableModule
	ext    *base.ConfigurableModuleExt
}

// NewModule creates a new instance of Module with default configurations
func NewModule() *Module {
	return newModuleWithOptions(
		genConfigOption(configKeyServiceType, "Default S3 service type (aws, minio, digitalocean, etc.)", "auto"),
		genSecretConfigOption(configKeyAccessKey, "Default S3 access key ID", ""),
		genSecretConfigOption(configKeySecretKey, "Default S3 secret access key", ""),
		genConfigOption(configKeySessionToken, "Default S3 session token", ""),
		genConfigOption(configKeyRegion, "Default S3 region", "us-east-1"),
		genConfigOption(configKeyEndpoint, "Default S3 endpoint URL", ""),
		genConfigOption(configKeyForcePathStyle, "Default force path-style addressing", false),
		genConfigOption(configKeyUseSSL, "Default use SSL/TLS", true),
		genConfigOption(configKeyTimeout, "Default request timeout in seconds", 30),
		genConfigOption(configKeyMaxRetries, "Default maximum retry attempts", 3),
		genConfigOption(configKeyPartSize, "Default multipart upload part size in bytes", int64(5*1024*1024)),
		genConfigOption(configKeyConcurrency, "Default number of concurrent operations", 3),
		genConfigOption(configKeyEnableLogging, "Default enable debug logging", false),
		genConfigOption(configKeyUserAgent, "Default user agent string", "starlark-s3/1.0"),
	)
}

// Helper functions

// genConfigOption creates a configuration option with common settings
func genConfigOption[T any](name, description string, defaultValue T) *base.ConfigOption[T] {
	envVar := fmt.Sprintf("S3_%s", strings.ToUpper(strings.ReplaceAll(name, "_", "_")))
	// Also support AWS standard environment variables
	switch name {
	case configKeySessionToken:
		envVar = "AWS_SESSION_TOKEN"
	case configKeyRegion:
		envVar = "AWS_DEFAULT_REGION"
	}

	return base.NewConfigOption(defaultValue).
		WithName(name).
		WithDescription(description).
		WithEnvVar(envVar)
}

// genSecretConfigOption creates a secret configuration option
func genSecretConfigOption(name, description, defaultValue string) *base.ConfigOption[string] {
	envVar := fmt.Sprintf("S3_%s", strings.ToUpper(strings.ReplaceAll(name, "_", "_")))
	// Also support AWS standard environment variables
	switch name {
	case configKeyAccessKey:
		envVar = "AWS_ACCESS_KEY_ID"
	case configKeySecretKey:
		envVar = "AWS_SECRET_ACCESS_KEY"
	}

	return base.NewConfigOption(defaultValue).
		WithName(name).
		WithDescription(description).
		WithEnvVar(envVar).
		SetSecret(true)
}

// newModuleWithOptions creates a Module with the given configuration options
func newModuleWithOptions(
	serviceTypeOpt *base.ConfigOption[string],
	accessKeyOpt *base.ConfigOption[string],
	secretKeyOpt *base.ConfigOption[string],
	sessionTokenOpt *base.ConfigOption[string],
	regionOpt *base.ConfigOption[string],
	endpointOpt *base.ConfigOption[string],
	forcePathStyleOpt *base.ConfigOption[bool],
	useSSLOpt *base.ConfigOption[bool],
	timeoutOpt *base.ConfigOption[int],
	maxRetriesOpt *base.ConfigOption[int],
	partSizeOpt *base.ConfigOption[int64],
	concurrencyOpt *base.ConfigOption[int],
	enableLoggingOpt *base.ConfigOption[bool],
	userAgentOpt *base.ConfigOption[string],
) *Module {
	cm, _ := base.NewConfigurableModuleWithConfigOptions(
		serviceTypeOpt,
		accessKeyOpt,
		secretKeyOpt,
		sessionTokenOpt,
		regionOpt,
		endpointOpt,
		forcePathStyleOpt,
		useSSLOpt,
		timeoutOpt,
		maxRetriesOpt,
		partSizeOpt,
		concurrencyOpt,
		enableLoggingOpt,
		userAgentOpt,
	)
	return &Module{
		cfgMod: cm,
		ext:    cm.Extend(),
	}
}

// LoadModule returns the Starlark module loader with S3-specific functions
func (m *Module) LoadModule() starlet.ModuleLoader {
	// Module functions
	additionalFuncs := starlark.StringDict{
		"create_client":          starlark.NewBuiltin(ModuleName+".create_client", m.starCreateClient),
		"validate_bucket_name":   starlark.NewBuiltin(ModuleName+".validate_bucket_name", starValidateBucketName),
		"validate_object_key":    starlark.NewBuiltin(ModuleName+".validate_object_key", starValidateObjectKey),
		"get_supported_services": starlark.NewBuiltin(ModuleName+".get_supported_services", starGetSupportedServices),
	}
	return m.cfgMod.LoadModule(ModuleName, additionalFuncs)
}

// starCreateClient creates and returns an S3 client
func (m *Module) starCreateClient(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		serviceType    = ""
		accessKey      = ""
		secretKey      = ""
		sessionToken   = ""
		region         = ""
		endpoint       = ""
		forcePathStyle = types.NewNullableBool(starlark.False)
		useSSL         = types.NewNullableBool(starlark.True)
		timeout        = 0
		maxRetries     = 0
		partSize       = int64(0)
		concurrency    = 0
		enableLogging  = types.NewNullableBool(starlark.False)
		userAgent      = ""
	)

	// Parse arguments - all optional
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"service_type?", &serviceType,
		"access_key?", &accessKey,
		"secret_key?", &secretKey,
		"session_token?", &sessionToken,
		"region?", &region,
		"endpoint?", &endpoint,
		"force_path_style?", forcePathStyle,
		"use_ssl?", useSSL,
		"timeout?", &timeout,
		"max_retries?", &maxRetries,
		"part_size?", &partSize,
		"concurrency?", &concurrency,
		"enable_logging?", enableLogging,
		"user_agent?", &userAgent,
	); err != nil {
		return none, err
	}

	// Helper function to get boolean config value
	getBoolConfigValue := func(moduleDefault bool, nullableOverride *types.NullableBool) bool {
		if !nullableOverride.IsNull() {
			return bool(nullableOverride.Value())
		}
		return moduleDefault
	}

	// Get configuration values from module, using provided values as overrides
	config := &ClientConfig{
		ServiceType:    getConfigValue(m.ext.GetString(configKeyServiceType), serviceType),
		AccessKey:      getConfigValue(m.ext.GetString(configKeyAccessKey), accessKey),
		SecretKey:      getConfigValue(m.ext.GetString(configKeySecretKey), secretKey),
		SessionToken:   getConfigValue(m.ext.GetString(configKeySessionToken), sessionToken),
		Region:         getConfigValue(m.ext.GetString(configKeyRegion), region),
		Endpoint:       getConfigValue(m.ext.GetString(configKeyEndpoint), endpoint),
		ForcePathStyle: getBoolConfigValue(m.ext.GetBool(configKeyForcePathStyle), forcePathStyle),
		UseSSL:         getBoolConfigValue(m.ext.GetBool(configKeyUseSSL), useSSL),
		Timeout:        getIntConfigValue(m.ext.GetInt(configKeyTimeout), timeout),
		MaxRetries:     getIntConfigValue(m.ext.GetInt(configKeyMaxRetries), maxRetries),
		PartSize:       getInt64ConfigValue(int64(m.ext.GetInt(configKeyPartSize)), partSize),
		Concurrency:    getIntConfigValue(m.ext.GetInt(configKeyConcurrency), concurrency),
		EnableLogging:  getBoolConfigValue(m.ext.GetBool(configKeyEnableLogging), enableLogging),
		UserAgent:      getConfigValue(m.ext.GetString(configKeyUserAgent), userAgent),
	}

	// Create the client
	ctx := dataconv.GetThreadContext(thread)
	client, err := NewClient(ctx, config)
	if err != nil {
		return none, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create the wrapper and return it as a Starlark struct
	wrapper := &ClientWrapper{client: client}
	return wrapper.Struct(), nil
}

// Helper functions for config value resolution
func getConfigValue(moduleDefault, override string) string {
	if override != "" {
		return override
	}
	return moduleDefault
}

func getBoolConfigValue(moduleDefault bool, override *bool) bool {
	if override != nil {
		return *override
	}
	return moduleDefault
}

func getBoolConfigValueDirect(moduleDefault bool, override bool, hasOverride bool) bool {
	if hasOverride {
		return override
	}
	return moduleDefault
}

func getIntConfigValue(moduleDefault, override int) int {
	if override != 0 {
		return override
	}
	return moduleDefault
}

func getInt64ConfigValue(moduleDefault, override int64) int64 {
	if override != 0 {
		return override
	}
	return moduleDefault
}

// ClientWrapper wraps the S3Client for Starlark
type ClientWrapper struct {
	client *Client
}

// Struct converts the S3Client to a Starlark struct
func (s *ClientWrapper) Struct() *starlarkstruct.Struct {
	return starlarkstruct.FromStringDict(starlark.String("S3Client"), starlark.StringDict{
		// Bucket operations
		"create_bucket":   starlark.NewBuiltin("s3.create_bucket", s.createBucket),
		"delete_bucket":   starlark.NewBuiltin("s3.delete_bucket", s.deleteBucket),
		"list_buckets":    starlark.NewBuiltin("s3.list_buckets", s.listBuckets),
		"bucket_exists":   starlark.NewBuiltin("s3.bucket_exists", s.bucketExists),
		"get_bucket_info": starlark.NewBuiltin("s3.get_bucket_info", s.getBucketInfo),

		// Object operations
		"put_object":      starlark.NewBuiltin("s3.put_object", s.putObject),
		"put_object_file": starlark.NewBuiltin("s3.put_object_file", s.putObjectFile),
		"get_object":      starlark.NewBuiltin("s3.get_object", s.getObject),
		"get_object_file": starlark.NewBuiltin("s3.get_object_file", s.getObjectFile),
		"delete_object":   starlark.NewBuiltin("s3.delete_object", s.deleteObject),
		"list_objects":    starlark.NewBuiltin("s3.list_objects", s.listObjects),
		"object_exists":   starlark.NewBuiltin("s3.object_exists", s.objectExists),
		"get_object_info": starlark.NewBuiltin("s3.get_object_info", s.getObjectInfo),
		"set_object_info": starlark.NewBuiltin("s3.set_object_info", s.setObjectInfo),
		"copy_object":     starlark.NewBuiltin("s3.copy_object", s.copyObject),
		"presign_url":     starlark.NewBuiltin("s3.presign_url", s.presignURL),
		"get_public_url":  starlark.NewBuiltin("s3.get_public_url", s.getPublicURL),
	})
}

// createBucket creates a new S3 bucket
func (s *ClientWrapper) createBucket(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		region = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"region?", &region,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)

	if region == "" {
		err := s.client.CreateBucket(ctx, bucket)
		if err != nil {
			return none, fmt.Errorf("failed to create bucket: %w", err)
		}
	} else {
		err := s.client.CreateBucket(ctx, bucket, region)
		if err != nil {
			return none, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return none, nil
}

// deleteBucket deletes an S3 bucket
func (s *ClientWrapper) deleteBucket(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		force  = false
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"force?", &force,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.DeleteBucket(ctx, bucket, force)
	if err != nil {
		return none, fmt.Errorf("failed to delete bucket: %w", err)
	}

	return none, nil
}

// listBuckets lists all S3 buckets
func (s *ClientWrapper) listBuckets(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	buckets, err := s.client.ListBuckets(ctx)
	if err != nil {
		return none, fmt.Errorf("failed to list buckets: %w", err)
	}

	return dataconv.Marshal(buckets)
}

// bucketExists checks if a bucket exists
func (s *ClientWrapper) bucketExists(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var bucket string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "bucket", &bucket); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return none, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return starlark.Bool(exists), nil
}

// getBucketInfo gets comprehensive information about a bucket
func (s *ClientWrapper) getBucketInfo(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var bucket string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "bucket", &bucket); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	info, err := s.client.GetBucketInfo(ctx, bucket)
	if err != nil {
		return none, fmt.Errorf("failed to get bucket info: %w", err)
	}

	return dataconv.Marshal(info)
}

// putObject uploads an object to S3
func (s *ClientWrapper) putObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket          = ""
		key             = ""
		content         = ""
		contentType     = ""
		metadata        = starlark.NewDict(0)
		tags            = starlark.NewDict(0)
		contentEncoding = ""
		cacheControl    = ""
		expires         = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"content", &content,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"tags?", &tags,
		"content_encoding?", &contentEncoding,
		"cache_control?", &cacheControl,
		"expires?", &expires,
	); err != nil {
		return none, err
	}

	// Convert content to reader
	contentReader := strings.NewReader(content)

	// Build options
	option := &ObjectOptions{}

	if contentType != "" {
		option.ContentType = &contentType
	}
	if contentEncoding != "" {
		option.ContentEncoding = &contentEncoding
	}
	if cacheControl != "" {
		option.CacheControl = &cacheControl
	}

	// Handle metadata
	if metadata.Len() > 0 {
		metadataMap, err := convertMetadataDict(metadata)
		if err != nil {
			return none, fmt.Errorf("failed to convert metadata: %w", err)
		}
		option.Metadata = &metadataMap
	}

	// Handle tags
	if tags.Len() > 0 {
		tagsMap, err := convertMetadataDict(tags)
		if err != nil {
			return none, fmt.Errorf("failed to convert tags: %w", err)
		}
		option.Tags = &tagsMap
	}

	// Handle expires
	if expires != "" {
		convertedTime, err := convertStarlarkStringToTime(expires)
		if err != nil {
			return none, fmt.Errorf("failed to convert expires time: %w", err)
		}
		option.Expires = &convertedTime
	}

	var opts *ObjectOptions
	if option.Validate() {
		opts = option
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.PutObject(ctx, bucket, key, contentReader, opts)
	if err != nil {
		return none, fmt.Errorf("failed to put object: %w", err)
	}

	return none, nil
}

// putObjectFile uploads a file directly to S3
func (s *ClientWrapper) putObjectFile(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket          = ""
		key             = ""
		filePath        = ""
		contentType     = ""
		metadata        = starlark.NewDict(0)
		contentEncoding = ""
		cacheControl    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"file_path", &filePath,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"content_encoding?", &contentEncoding,
		"cache_control?", &cacheControl,
	); err != nil {
		return none, err
	}

	// Build options
	option := &ObjectOptions{}

	if contentType != "" {
		option.ContentType = &contentType
	}
	if contentEncoding != "" {
		option.ContentEncoding = &contentEncoding
	}
	if cacheControl != "" {
		option.CacheControl = &cacheControl
	}

	// Handle metadata
	if metadata.Len() > 0 {
		metadataMap, err := convertMetadataDict(metadata)
		if err != nil {
			return none, fmt.Errorf("failed to convert metadata: %w", err)
		}
		option.Metadata = &metadataMap
	}

	var opts *ObjectOptions
	if option.Validate() {
		opts = option
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.PutObjectFromFile(ctx, bucket, key, filePath, opts)
	if err != nil {
		return none, fmt.Errorf("failed to put object from file: %w", err)
	}

	return none, nil
}

// getObject downloads an object from S3
func (s *ClientWrapper) getObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		key    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	reader, err := s.client.GetObject(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to get object: %w", err)
	}
	defer reader.Close()

	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return none, fmt.Errorf("failed to read object content: %w", err)
	}

	return starlark.String(string(content)), nil
}

// getObjectFile downloads an object from S3 to a local file
func (s *ClientWrapper) getObjectFile(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket   = ""
		key      = ""
		filePath = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"file_path", &filePath,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.GetObjectToFile(ctx, bucket, key, filePath)
	if err != nil {
		return none, fmt.Errorf("failed to get object to file: %w", err)
	}

	return none, nil
}

// deleteObject deletes an object from S3
func (s *ClientWrapper) deleteObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		key    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.DeleteObject(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to delete object: %w", err)
	}

	return none, nil
}

// listObjects lists objects in an S3 bucket
func (s *ClientWrapper) listObjects(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket    = ""
		prefix    = ""
		delimiter = ""
		maxKeys   = 1000
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"prefix?", &prefix,
		"delimiter?", &delimiter,
		"max_keys?", &maxKeys,
	); err != nil {
		return none, err
	}

	// Build options
	option := &ListObjectsOptions{}

	if prefix != "" {
		option.Prefix = &prefix
	}
	if delimiter != "" {
		option.Delimiter = &delimiter
	}
	if maxKeys > 0 {
		option.MaxKeys = &maxKeys
	}

	var opts *ListObjectsOptions
	if option.Validate() {
		opts = option
	}

	ctx := dataconv.GetThreadContext(thread)
	result, err := s.client.ListObjects(ctx, bucket, opts)
	if err != nil {
		return none, fmt.Errorf("failed to list objects: %w", err)
	}

	return dataconv.Marshal(result)
}

// objectExists checks if an object exists
func (s *ClientWrapper) objectExists(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		key    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	exists, err := s.client.ObjectExists(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to check object existence: %w", err)
	}

	return starlark.Bool(exists), nil
}

// getObjectInfo gets metadata about an object
func (s *ClientWrapper) getObjectInfo(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		key    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	info, err := s.client.GetObjectInfo(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to get object info: %w", err)
	}

	return dataconv.Marshal(info)
}

// setObjectInfo sets metadata and other object properties for an object
func (s *ClientWrapper) setObjectInfo(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket             = ""
		key                = ""
		metadata           = starlark.NewDict(0)
		tags               = starlark.NewDict(0)
		contentType        = ""
		cacheControl       = ""
		contentEncoding    = ""
		contentDisposition = ""
		contentLanguage    = ""
		expires            = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"metadata?", &metadata,
		"tags?", &tags,
		"content_type?", &contentType,
		"cache_control?", &cacheControl,
		"content_encoding?", &contentEncoding,
		"content_disposition?", &contentDisposition,
		"content_language?", &contentLanguage,
		"expires?", &expires,
	); err != nil {
		return none, err
	}

	// Convert metadata
	var metadataMap map[string]string
	if metadata.Len() > 0 {
		metadataMap = make(map[string]string)
		for _, item := range metadata.Items() {
			key := item[0].(starlark.String).GoString()
			value := item[1].(starlark.String).GoString()
			metadataMap[key] = value
		}
	}

	// Convert tags
	var tagsMap map[string]string
	if tags.Len() > 0 {
		tagsMap = make(map[string]string)
		for _, item := range tags.Items() {
			key := item[0].(starlark.String).GoString()
			value := item[1].(starlark.String).GoString()
			tagsMap[key] = value
		}
	}

	// Convert expires
	var expiresTime *time.Time
	if expires != "" {
		convertedTime, err := convertStarlarkStringToTime(expires)
		if err != nil {
			return none, fmt.Errorf("failed to convert expires time: %w", err)
		}
		expiresTime = &convertedTime
	}

	// Build options using the new ObjectOptions struct with pointer types
	option := &ObjectOptions{}

	if contentType != "" {
		option.ContentType = &contentType
	}
	if len(metadataMap) > 0 {
		option.Metadata = &metadataMap
	}
	if len(tagsMap) > 0 {
		option.Tags = &tagsMap
	}
	if cacheControl != "" {
		option.CacheControl = &cacheControl
	}
	if contentEncoding != "" {
		option.ContentEncoding = &contentEncoding
	}
	if contentDisposition != "" {
		option.ContentDisposition = &contentDisposition
	}
	if contentLanguage != "" {
		option.ContentLanguage = &contentLanguage
	}
	if expiresTime != nil {
		option.Expires = expiresTime
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.SetObjectInfo(ctx, bucket, key, option)
	if err != nil {
		return none, fmt.Errorf("failed to set object info: %w", err)
	}

	return none, nil
}

// copyObject copies an object from one location to another
func (s *ClientWrapper) copyObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		srcBucket       = ""
		srcKey          = ""
		dstBucket       = ""
		dstKey          = ""
		contentType     = ""
		metadata        = starlark.NewDict(0)
		contentEncoding = ""
		cacheControl    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"src_bucket", &srcBucket,
		"src_key", &srcKey,
		"dst_bucket", &dstBucket,
		"dst_key", &dstKey,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"content_encoding?", &contentEncoding,
		"cache_control?", &cacheControl,
	); err != nil {
		return none, err
	}

	// Build options
	option := &ObjectOptions{}

	if contentType != "" {
		option.ContentType = &contentType
	}
	if contentEncoding != "" {
		option.ContentEncoding = &contentEncoding
	}
	if cacheControl != "" {
		option.CacheControl = &cacheControl
	}

	// Handle metadata
	if metadata.Len() > 0 {
		metadataMap, err := convertMetadataDict(metadata)
		if err != nil {
			return none, fmt.Errorf("failed to convert metadata: %w", err)
		}
		option.Metadata = &metadataMap
	}

	var opts *ObjectOptions
	if option.Validate() {
		opts = option
	}

	ctx := dataconv.GetThreadContext(thread)
	err := s.client.CopyObject(ctx, srcBucket, srcKey, dstBucket, dstKey, opts)
	if err != nil {
		return none, fmt.Errorf("failed to copy object: %w", err)
	}

	return none, nil
}

// presignURL generates a pre-signed URL for temporary access to an object
func (s *ClientWrapper) presignURL(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket    = ""
		key       = ""
		expiresIn = 3600  // Default: 1 hour
		method    = "GET" // Default: GET
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"expires_in?", &expiresIn,
		"method?", &method,
	); err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	url, err := s.client.PresignURL(ctx, bucket, key, expiresIn, method)
	if err != nil {
		return none, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return starlark.String(url), nil
}

// getPublicURL generates a public HTTP URL for an object using client configuration
func (s *ClientWrapper) getPublicURL(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		key    = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
	); err != nil {
		return none, err
	}

	// Get client configuration
	config := s.client.GetConfig()

	// Use the client's configuration to generate the public URL
	url := GenerateURLWithProvider(bucket, key, config.Region, config.Endpoint, config.UseSSL, config.ServiceType)
	return starlark.String(url), nil
}

// Utility functions for Starlark

// starValidateBucketName validates a bucket name
func starValidateBucketName(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name); err != nil {
		return none, err
	}

	err := validateBucketName(name)
	return starlark.Bool(err == nil), nil
}

// starValidateObjectKey validates an object key
func starValidateObjectKey(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var key string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "key", &key); err != nil {
		return none, err
	}

	err := validateObjectKey(key)
	return starlark.Bool(err == nil), nil
}

// starGetSupportedServices returns the list of supported S3 services
func starGetSupportedServices(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return none, err
	}

	services := GetAllProviders()
	return dataconv.Marshal(services)
}
