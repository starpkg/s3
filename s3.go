// Package s3 provides a Starlark module for S3-compatible storage operations.
package s3

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv"
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
	cm := base.NewConfigurableModule()
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
		"parse_s3_url":           starlark.NewBuiltin(ModuleName+".parse_s3_url", starParseS3URL),
		"generate_s3_url":        starlark.NewBuiltin(ModuleName+".generate_s3_url", starGenerateS3URL),
		"get_public_url":         starlark.NewBuiltin(ModuleName+".get_public_url", starGetPublicURL),
		"validate_bucket_name":   starlark.NewBuiltin(ModuleName+".validate_bucket_name", starValidateBucketName),
		"validate_object_key":    starlark.NewBuiltin(ModuleName+".validate_object_key", starValidateObjectKey),
		"get_supported_services": starlark.NewBuiltin(ModuleName+".get_supported_services", starGetSupportedServices),
		"get_client_info":        starlark.NewBuiltin(ModuleName+".get_client_info", starGetClientInfo),
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
		forcePathStyle = false
		useSSL         = true
		timeout        = 30
		maxRetries     = 3
		partSize       = int64(5 * 1024 * 1024)
		concurrency    = 3
		enableLogging  = false
		userAgent      = ""
	)

	// Parse arguments
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"service_type?", &serviceType,
		"access_key?", &accessKey,
		"secret_key?", &secretKey,
		"session_token?", &sessionToken,
		"region?", &region,
		"endpoint?", &endpoint,
		"force_path_style?", &forcePathStyle,
		"use_ssl?", &useSSL,
		"timeout?", &timeout,
		"max_retries?", &maxRetries,
		"part_size?", &partSize,
		"concurrency?", &concurrency,
		"enable_logging?", &enableLogging,
		"user_agent?", &userAgent,
	); err != nil {
		return none, err
	}

	// Use defaults if not provided
	if serviceType == "" {
		serviceType = "auto"
	}
	if region == "" {
		region = "us-east-1"
	}
	if userAgent == "" {
		userAgent = "starlark-s3/1.0"
	}

	// Create client configuration
	config := &ClientConfig{
		ServiceType:    serviceType,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		SessionToken:   sessionToken,
		Region:         region,
		Endpoint:       endpoint,
		ForcePathStyle: forcePathStyle,
		UseSSL:         useSSL,
		Timeout:        timeout,
		MaxRetries:     maxRetries,
		PartSize:       partSize,
		Concurrency:    concurrency,
		EnableLogging:  enableLogging,
		UserAgent:      userAgent,
	}

	// Create the client
	client, err := NewS3Client(context.Background(), config)
	if err != nil {
		return none, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create the wrapper and return it as a Starlark struct
	wrapper := &S3ClientStruct{client: client}
	return wrapper.toStarlarkStruct(), nil
}

// S3ClientStruct wraps the S3Client for Starlark
type S3ClientStruct struct {
	client *S3Client
}

// toStarlarkStruct converts the S3Client to a Starlark struct
func (s *S3ClientStruct) toStarlarkStruct() *starlarkstruct.Struct {
	return starlarkstruct.FromStringDict(starlark.String("S3Client"), starlark.StringDict{
		// Bucket operations
		"create_bucket":       starlark.NewBuiltin("s3.create_bucket", s.createBucket),
		"delete_bucket":       starlark.NewBuiltin("s3.delete_bucket", s.deleteBucket),
		"list_buckets":        starlark.NewBuiltin("s3.list_buckets", s.listBuckets),
		"bucket_exists":       starlark.NewBuiltin("s3.bucket_exists", s.bucketExists),
		"get_bucket_location": starlark.NewBuiltin("s3.get_bucket_location", s.getBucketLocation),

		// Object operations
		"put_object":      starlark.NewBuiltin("s3.put_object", s.putObject),
		"get_object":      starlark.NewBuiltin("s3.get_object", s.getObject),
		"delete_object":   starlark.NewBuiltin("s3.delete_object", s.deleteObject),
		"list_objects":    starlark.NewBuiltin("s3.list_objects", s.listObjects),
		"object_exists":   starlark.NewBuiltin("s3.object_exists", s.objectExists),
		"get_object_info": starlark.NewBuiltin("s3.get_object_info", s.getObjectInfo),

		// Utility methods
		"get_config": starlark.NewBuiltin("s3.get_config", s.getConfig),
	})
}

// createBucket creates a new S3 bucket
func (s *S3ClientStruct) createBucket(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	ctx := context.Background()

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
func (s *S3ClientStruct) deleteBucket(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	ctx := context.Background()
	err := s.client.DeleteBucket(ctx, bucket, force)
	if err != nil {
		return none, fmt.Errorf("failed to delete bucket: %w", err)
	}

	return none, nil
}

// listBuckets lists all S3 buckets
func (s *S3ClientStruct) listBuckets(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return none, err
	}

	ctx := context.Background()
	buckets, err := s.client.ListBuckets(ctx)
	if err != nil {
		return none, fmt.Errorf("failed to list buckets: %w", err)
	}

	return dataconv.Marshal(buckets)
}

// bucketExists checks if a bucket exists
func (s *S3ClientStruct) bucketExists(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var bucket string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "bucket", &bucket); err != nil {
		return none, err
	}

	ctx := context.Background()
	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return none, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	return starlark.Bool(exists), nil
}

// getBucketLocation gets the location/region of a bucket
func (s *S3ClientStruct) getBucketLocation(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var bucket string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "bucket", &bucket); err != nil {
		return none, err
	}

	ctx := context.Background()
	location, err := s.client.GetBucketLocation(ctx, bucket)
	if err != nil {
		return none, fmt.Errorf("failed to get bucket location: %w", err)
	}

	return starlark.String(location), nil
}

// putObject uploads an object to S3
func (s *S3ClientStruct) putObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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
	var options []PutObjectOption

	if contentType != "" {
		options = append(options, WithContentType(contentType))
	}

	if contentEncoding != "" {
		options = append(options, WithContentEncoding(contentEncoding))
	}

	if cacheControl != "" {
		options = append(options, WithCacheControl(cacheControl))
	}

	// Handle metadata
	if metadata.Len() > 0 {
		metadataMap := make(map[string]string)
		for _, item := range metadata.Items() {
			key := item[0].(starlark.String).GoString()
			value := item[1].(starlark.String).GoString()
			metadataMap[key] = value
		}
		options = append(options, WithMetadata(metadataMap))
	}

	ctx := context.Background()
	err := s.client.PutObject(ctx, bucket, key, contentReader, options...)
	if err != nil {
		return none, fmt.Errorf("failed to put object: %w", err)
	}

	return none, nil
}

// getObject downloads an object from S3
func (s *S3ClientStruct) getObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	ctx := context.Background()
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

// deleteObject deletes an object from S3
func (s *S3ClientStruct) deleteObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	ctx := context.Background()
	err := s.client.DeleteObject(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to delete object: %w", err)
	}

	return none, nil
}

// listObjects lists objects in an S3 bucket
func (s *S3ClientStruct) listObjects(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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
	var options []ListObjectsOption

	if prefix != "" {
		options = append(options, WithPrefix(prefix))
	}

	if delimiter != "" {
		options = append(options, WithDelimiter(delimiter))
	}

	if maxKeys > 0 {
		options = append(options, WithMaxKeys(maxKeys))
	}

	ctx := context.Background()
	result, err := s.client.ListObjects(ctx, bucket, options...)
	if err != nil {
		return none, fmt.Errorf("failed to list objects: %w", err)
	}

	return dataconv.Marshal(result)
}

// objectExists checks if an object exists
func (s *S3ClientStruct) objectExists(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	ctx := context.Background()
	exists, err := s.client.ObjectExists(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to check object existence: %w", err)
	}

	return starlark.Bool(exists), nil
}

// getObjectInfo gets metadata about an object
func (s *S3ClientStruct) getObjectInfo(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	ctx := context.Background()
	info, err := s.client.GetObjectInfo(ctx, bucket, key)
	if err != nil {
		return none, fmt.Errorf("failed to get object info: %w", err)
	}

	return dataconv.Marshal(info)
}

// getConfig returns the client configuration
func (s *S3ClientStruct) getConfig(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return none, err
	}

	config := s.client.GetConfig()

	// Convert to a simple map for Starlark
	configMap := map[string]interface{}{
		"service_type":     config.ServiceType,
		"endpoint":         config.Endpoint,
		"region":           config.Region,
		"force_path_style": config.ForcePathStyle,
		"use_ssl":          config.UseSSL,
		"timeout":          config.Timeout,
		"max_retries":      config.MaxRetries,
		"part_size":        config.PartSize,
		"concurrency":      config.Concurrency,
		"enable_logging":   config.EnableLogging,
		"user_agent":       config.UserAgent,
	}

	return dataconv.Marshal(configMap)
}

// Utility functions for Starlark

// starParseS3URL parses an S3 URL into bucket and key components
func starParseS3URL(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var urlStr string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "url", &urlStr); err != nil {
		return none, err
	}

	bucket, key, err := parseS3URL(urlStr)
	if err != nil {
		return none, err
	}

	result := map[string]string{
		"bucket": bucket,
		"key":    key,
	}
	return dataconv.Marshal(result)
}

// starGenerateS3URL generates a standard S3 URL
func starGenerateS3URL(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket = ""
		key    = ""
		region = "us-east-1"
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"region?", &region,
	); err != nil {
		return none, err
	}

	url := generateS3URL(bucket, key)
	return starlark.String(url), nil
}

// starGetPublicURL generates a public HTTP URL for an object
func starGetPublicURL(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		clientVal starlark.Value
		bucket    string
		key       string
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"client", &clientVal,
		"bucket", &bucket,
		"key", &key,
	); err != nil {
		return none, err
	}

	// Extract client from Starlark struct
	clientStruct, ok := clientVal.(*starlarkstruct.Struct)
	if !ok {
		return none, fmt.Errorf("invalid client type")
	}

	// Get the get_config method and call it to get config info
	getConfigMethod, err := clientStruct.Attr("get_config")
	if err != nil {
		return none, fmt.Errorf("client does not have get_config method")
	}

	configVal, err := starlark.Call(thread, getConfigMethod, starlark.Tuple{}, nil)
	if err != nil {
		return none, fmt.Errorf("failed to get client config: %w", err)
	}

	// Extract config to determine service type and region
	configData, err := dataconv.Unmarshal(configVal)
	if err != nil {
		return none, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config, ok := configData.(*ClientConfig)
	if !ok {
		return none, fmt.Errorf("invalid client config type")
	}

	// Generate public URL using the config
	url := getPublicURL(bucket, key, config.Region, config.Endpoint, config.UseSSL)
	return starlark.String(url), nil
}

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

	services := getSupportedServices()
	return dataconv.Marshal(services)
}

// starGetClientInfo returns information about an S3 client
func starGetClientInfo(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var clientVal starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "client", &clientVal); err != nil {
		return none, err
	}

	// Extract client from Starlark struct
	clientStruct, ok := clientVal.(*starlarkstruct.Struct)
	if !ok {
		return none, fmt.Errorf("invalid client type")
	}

	// Get the get_config method and call it to get config info
	getConfigMethod, err := clientStruct.Attr("get_config")
	if err != nil {
		return none, fmt.Errorf("client does not have get_config method")
	}

	configVal, err := starlark.Call(thread, getConfigMethod, starlark.Tuple{}, nil)
	if err != nil {
		return none, fmt.Errorf("failed to get client config: %w", err)
	}

	return configVal, nil
}
