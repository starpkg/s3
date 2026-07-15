// Package s3 provides a Starlark module for S3-compatible storage operations.
package s3

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/1set/starlet"
	"github.com/1set/starlet/dataconv"
	"github.com/1set/starlet/dataconv/types"
	"github.com/starpkg/base"
	"github.com/starpkg/base/util"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ModuleName defines the expected name for this module when used in Starlark's load() function
const ModuleName = "s3"

// defaultMaxObjectSize caps how many bytes get_object reads into memory (256 MiB).
const defaultMaxObjectSize = 256 * 1024 * 1024

// resolveMaxObjectSize maps a configured max_object_size to the effective cap: 0
// means unlimited (explicit), but a NEGATIVE value is a misconfiguration that must
// fail SAFE — it falls back to the default rather than silently disabling the
// guard (fail-open).
func resolveMaxObjectSize(configured int) int {
	if configured < 0 {
		return defaultMaxObjectSize
	}
	return configured
}

var (
	none = starlark.None
)

// Ensure ClientWrapper implements the required Starlark interfaces
var (
	_ starlark.Value    = (*ClientWrapper)(nil)
	_ starlark.HasAttrs = (*ClientWrapper)(nil)
)

// Module wraps the ConfigurableModule with specific functionality for S3 operations
type Module struct {
	cfgMod *base.ConfigurableModule
	ext    *base.ConfigurableModuleExt
	// fileRoot is the absolute local-file jail root, resolved ONCE at module
	// construction (in Go, before any script runs) so a script cannot move it by
	// changing the working directory before calling create_client. Empty means
	// the root could not be resolved — file operations then fail closed.
	fileRoot string
}

// NewModule creates a new instance of Module with default configurations
func NewModule() *Module {
	return newModuleWithOptions(
		genConfigOption(configKeyServiceType, "Default S3 service type (aws, minio, digitalocean, etc.)", "auto"),
		genSecretConfigOption(configKeyAccessKey, "Default S3 access key ID", ""),
		genSecretConfigOption(configKeySecretKey, "Default S3 secret access key", ""),
		genSecretConfigOption(configKeySessionToken, "Default S3 session token", ""),
		genConfigOption(configKeyRegion, "Default S3 region", "us-east-1"),
		genConfigOption(configKeyEndpoint, "Default S3 endpoint URL", ""),
		genConfigOption(configKeyForcePathStyle, "Default force path-style addressing", false),
		genConfigOption(configKeyUseSSL, "Default use SSL/TLS", true),
		genConfigOption(configKeyTimeout, "Default request timeout in seconds", 30),
		genConfigOption(configKeyMaxRetries, "Default maximum retry attempts", 3),
		genConfigOption(configKeyPartSize, "Default multipart upload part size in bytes", int64(5*1024*1024)),
		genConfigOption(configKeyConcurrency, "Default number of concurrent operations", 3),
		genConfigOption(configKeyEnableLogging, "Default enable debug logging", false),
		genConfigOption(configKeyUserAgent, "Default user agent string", "Starlark-S3/1.0"),
		// File-access policy for put_object_file / get_object_file. HOST-ONLY: a
		// script must not be able to widen its own filesystem reach. file_root
		// confines those paths ("" = the process working directory);
		// allow_unsafe_file_paths (default false) is the explicit opt-out that
		// disables the confinement — the host's decision, never the script's.
		genConfigOption(configKeyFileRoot, "Root directory that put_object_file/get_object_file paths are confined under ('' = working directory)", "").
			SetHostOnly(true),
		genConfigOption(configKeyAllowUnsafeFilePaths, "Allow put_object_file/get_object_file to use paths outside file_root", false).
			SetHostOnly(true),
		// max_object_size bounds how much get_object reads into memory, so a
		// hostile/huge object can't exhaust host memory. HOST-ONLY (a DoS guard a
		// script must not be able to raise); 0 = unlimited. Default 256 MiB.
		genConfigOption(configKeyMaxObjectSize, "Maximum bytes get_object reads into memory (0 = unlimited)", defaultMaxObjectSize).
			SetHostOnly(true),
	)
}

// Helper functions

// genConfigOption creates a configuration option with common settings
func genConfigOption[T any](name, description string, defaultValue T) *base.ConfigOption[T] {
	envVar := fmt.Sprintf("S3_%s", strings.ToUpper(name))

	return base.NewConfigOption(defaultValue).
		WithName(name).
		WithDescription(description).
		WithEnvVar(envVar)
}

// genSecretConfigOption creates a secret configuration option
func genSecretConfigOption(name, description, defaultValue string) *base.ConfigOption[string] {
	envVar := fmt.Sprintf("S3_%s", strings.ToUpper(name))

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
	fileRootOpt *base.ConfigOption[string],
	allowUnsafeFilePathsOpt *base.ConfigOption[bool],
	maxObjectSizeOpt *base.ConfigOption[int],
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
		fileRootOpt,
		allowUnsafeFilePathsOpt,
		maxObjectSizeOpt,
	)
	m := &Module{
		cfgMod: cm,
		ext:    cm.Extend(),
	}
	// Snapshot the file-access jail root now, in Go, before any script runs — so a
	// script cannot chdir before create_client to widen the default (working-dir)
	// jail. absFileRoot returns "" if the root can't be resolved (fail closed).
	m.fileRoot = absFileRoot(m.ext.GetString(configKeyFileRoot))
	return m
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
	// Credentials are intentionally NOT accepted here: they must be injected by
	// the host (via the access_key/secret_key/session_token config options, the
	// S3_* environment variables, or the AWS default credential chain), never
	// passed as literals from an untrusted script. Passing access_key= etc. is
	// rejected by UnpackArgs as an unexpected keyword argument.
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"service_type?", &serviceType,
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
		ServiceType: getConfigValue(m.ext.GetString(configKeyServiceType), serviceType),
		// Host-injected only (no script override).
		AccessKey:      m.ext.GetString(configKeyAccessKey),
		SecretKey:      m.ext.GetString(configKeySecretKey),
		SessionToken:   m.ext.GetString(configKeySessionToken),
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

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client configuration: %w", err)
	}

	// Apply smart detection if service type is "auto" or empty
	if config.ServiceType == "auto" || config.ServiceType == "" {
		config.ServiceType = config.detectServiceType()
	}

	// Create the client
	ctx := dataconv.GetThreadContext(thread)
	client, err := NewClient(ctx, config)
	if err != nil {
		return none, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Create the wrapper and apply the host's file-access policy for
	// put_object_file / get_object_file. m.fileRoot was resolved absolute at module
	// construction (before any script ran), so it is immune to an in-script chdir.
	wrapper := NewClientWrapper(client)
	wrapper.setFileAccess(m.fileRoot, m.ext.GetBool(configKeyAllowUnsafeFilePaths))
	wrapper.maxObjectSize = resolveMaxObjectSize(m.ext.GetInt(configKeyMaxObjectSize))
	return wrapper, nil
}

// Helper functions for config value resolution
func getConfigValue(moduleDefault, override string) string {
	if override != "" {
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
	client    *Client
	methodMap map[string]func() starlark.Value
	allNames  []string

	// File-access policy for put_object_file / get_object_file (host-only).
	fileRoot             string
	allowUnsafeFilePaths bool
	// maxObjectSize bounds get_object's in-memory read (host-only; 0 = unlimited).
	maxObjectSize int
}

// setFileAccess records the host's file-access policy for put_object_file /
// get_object_file. fileRoot must already be absolute (it is snapshotted at module
// construction, before any script runs); it is stored as-is.
func (s *ClientWrapper) setFileAccess(fileRoot string, allowUnsafeFilePaths bool) {
	s.fileRoot = fileRoot
	s.allowUnsafeFilePaths = allowUnsafeFilePaths
}

// absFileRoot resolves a configured file_root to an absolute path ONCE. An empty
// root means the working directory. It returns "" if the path cannot be made
// absolute (e.g. os.Getwd fails) — the caller then fails closed rather than
// retaining a relative root that a later chdir could re-anchor.
func absFileRoot(fileRoot string) string {
	if fileRoot == "" {
		fileRoot = "."
	}
	abs, err := filepath.Abs(fileRoot)
	if err != nil {
		return "" // fail closed
	}
	return abs
}

// resolveFilePath confines a script-supplied local file path to the host's
// file_root (an absolute root snapshotted at module construction), rejecting any
// path that escapes it (via `..` or a symlink) — so a script cannot read or write
// arbitrary host files through put_object_file / get_object_file. It is bypassed
// only when the host set allow_unsafe_file_paths; and it fails closed if the root
// could not be resolved.
func (s *ClientWrapper) resolveFilePath(p string) (string, error) {
	if s.allowUnsafeFilePaths {
		return p, nil
	}
	if s.fileRoot == "" || !filepath.IsAbs(s.fileRoot) {
		return "", fmt.Errorf("file operations are disabled: no valid file_root")
	}
	resolved, err := util.ResolveUnder(s.fileRoot, p)
	if err != nil {
		return "", fmt.Errorf("file_path %q is outside the allowed root: %w", p, err)
	}
	return resolved, nil
}

// NewClientWrapper creates a new ClientWrapper with initialized method maps
func NewClientWrapper(client *Client) *ClientWrapper {
	cw := &ClientWrapper{
		client: client,
		// Secure default: confine local file paths to the working directory
		// (snapshotted absolute) until the host applies its own policy.
		fileRoot: absFileRoot(""),
	}

	// Initialize method map
	cw.methodMap = map[string]func() starlark.Value{
		// Client information
		"get_client_info": func() starlark.Value { return starlark.NewBuiltin("s3.get_client_info", cw.getClientInfo) },

		// Bucket operations
		"create_bucket":   func() starlark.Value { return starlark.NewBuiltin("s3.create_bucket", cw.createBucket) },
		"delete_bucket":   func() starlark.Value { return starlark.NewBuiltin("s3.delete_bucket", cw.deleteBucket) },
		"list_buckets":    func() starlark.Value { return starlark.NewBuiltin("s3.list_buckets", cw.listBuckets) },
		"bucket_exists":   func() starlark.Value { return starlark.NewBuiltin("s3.bucket_exists", cw.bucketExists) },
		"get_bucket_info": func() starlark.Value { return starlark.NewBuiltin("s3.get_bucket_info", cw.getBucketInfo) },

		// Object operations
		"put_object":      func() starlark.Value { return starlark.NewBuiltin("s3.put_object", cw.putObject) },
		"put_object_file": func() starlark.Value { return starlark.NewBuiltin("s3.put_object_file", cw.putObjectFile) },
		"get_object":      func() starlark.Value { return starlark.NewBuiltin("s3.get_object", cw.getObject) },
		"get_object_file": func() starlark.Value { return starlark.NewBuiltin("s3.get_object_file", cw.getObjectFile) },
		"delete_object":   func() starlark.Value { return starlark.NewBuiltin("s3.delete_object", cw.deleteObject) },
		"list_objects":    func() starlark.Value { return starlark.NewBuiltin("s3.list_objects", cw.listObjects) },
		"object_exists":   func() starlark.Value { return starlark.NewBuiltin("s3.object_exists", cw.objectExists) },
		"get_object_info": func() starlark.Value { return starlark.NewBuiltin("s3.get_object_info", cw.getObjectInfo) },
		"set_object_info": func() starlark.Value { return starlark.NewBuiltin("s3.set_object_info", cw.setObjectInfo) },
		"copy_object":     func() starlark.Value { return starlark.NewBuiltin("s3.copy_object", cw.copyObject) },
		"presign_url":     func() starlark.Value { return starlark.NewBuiltin("s3.presign_url", cw.presignURL) },
		"get_public_url":  func() starlark.Value { return starlark.NewBuiltin("s3.get_public_url", cw.getPublicURL) },
	}

	// Collect all attribute names
	cw.allNames = make([]string, 0, len(cw.methodMap))
	for name := range cw.methodMap {
		cw.allNames = append(cw.allNames, name)
	}

	return cw
}

// String implements starlark.Value; it renders the client as
// "<s3.Client service_type=… region=…>" for printing and error messages.
func (cw *ClientWrapper) String() string {
	config := cw.client.GetConfig()
	return fmt.Sprintf("<s3.Client service_type=%s region=%s>", config.ServiceType, config.Region)
}

// Type implements starlark.Value and returns the Starlark type name "s3.Client".
func (cw *ClientWrapper) Type() string {
	return "s3.Client"
}

// Freeze implements starlark.Value. The client is immutable after creation, so
// freezing is a no-op.
func (cw *ClientWrapper) Freeze() {
	// Client is immutable after creation
}

// Truth implements starlark.Value; a client is always truthy.
func (cw *ClientWrapper) Truth() starlark.Bool {
	return starlark.True
}

// Hash implements starlark.Value. The client is unhashable (it cannot be used as
// a dict key), so this always returns an error.
func (cw *ClientWrapper) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", cw.Type())
}

// Attr implements starlark.HasAttrs, resolving a method name (e.g. "put_object")
// to its bound builtin, or returning a no-such-attribute error.
func (cw *ClientWrapper) Attr(name string) (starlark.Value, error) {
	// Check for methods using map lookup
	if methodFunc, exists := cw.methodMap[name]; exists {
		return methodFunc(), nil
	}

	return nil, starlark.NoSuchAttrError(fmt.Sprintf("%s has no .%s attribute", cw.Type(), name))
}

// AttrNames implements starlark.HasAttrs and lists every callable method name on
// the client (used for dir() and attribute completion).
func (cw *ClientWrapper) AttrNames() []string {
	return cw.allNames
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

	// Convert slice of BucketInfo to Starlark list manually
	bucketList := starlark.NewList(make([]starlark.Value, len(buckets)))
	for i, bucket := range buckets {
		bucketValue, err := bucket.MarshalStarlark()
		if err != nil {
			return none, fmt.Errorf("failed to marshal bucket info: %w", err)
		}
		bucketList.SetIndex(i, bucketValue)
	}

	return bucketList, nil
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
		bucket             = ""
		key                = ""
		content            = ""
		contentType        = ""
		metadata           = starlark.NewDict(0)
		tags               = starlark.NewDict(0)
		cacheControl       = ""
		contentDisposition = ""
		contentEncoding    = ""
		contentLanguage    = ""
		expires            = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"content", &content,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"tags?", &tags,
		"cache_control?", &cacheControl,
		"content_disposition?", &contentDisposition,
		"content_encoding?", &contentEncoding,
		"content_language?", &contentLanguage,
		"expires?", &expires,
	); err != nil {
		return none, err
	}

	// Convert content to reader
	contentReader := strings.NewReader(content)

	// Parse object options using helper function
	option, err := parseObjectOptions(contentType, cacheControl, contentDisposition, contentEncoding, contentLanguage, expires, metadata, tags)
	if err != nil {
		return none, err
	}

	var opts *ObjectOptions
	if option.Validate() {
		opts = option
	}

	ctx := dataconv.GetThreadContext(thread)
	err = s.client.PutObject(ctx, bucket, key, contentReader, opts)
	if err != nil {
		return none, fmt.Errorf("failed to put object: %w", err)
	}

	return none, nil
}

// putObjectFile uploads a file directly to S3
func (s *ClientWrapper) putObjectFile(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		bucket             = ""
		key                = ""
		filePath           = ""
		contentType        = ""
		metadata           = starlark.NewDict(0)
		tags               = starlark.NewDict(0)
		contentEncoding    = ""
		cacheControl       = ""
		contentDisposition = ""
		contentLanguage    = ""
		expires            = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"file_path", &filePath,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"tags?", &tags,
		"content_encoding?", &contentEncoding,
		"cache_control?", &cacheControl,
		"content_disposition?", &contentDisposition,
		"content_language?", &contentLanguage,
		"expires?", &expires,
	); err != nil {
		return none, err
	}

	// Parse object options using helper function
	option, err := parseObjectOptions(contentType, cacheControl, contentDisposition, contentEncoding, contentLanguage, expires, metadata, tags)
	if err != nil {
		return none, err
	}

	var opts *ObjectOptions
	if option.Validate() {
		opts = option
	}

	safePath, err := s.resolveFilePath(filePath)
	if err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	err = s.client.PutObjectFromFile(ctx, bucket, key, safePath, opts)
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
	// Bound the in-memory read so a huge object can't exhaust host memory.
	content, err := util.ReadAllLimited(reader, int64(s.maxObjectSize))
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

	safePath, err := s.resolveFilePath(filePath)
	if err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	err = s.client.GetObjectToFile(ctx, bucket, key, safePath)
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

	return result.MarshalStarlark()
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
		contentDisposition = ""
		contentEncoding    = ""
		contentLanguage    = ""
		expires            = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"bucket", &bucket,
		"key", &key,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"tags?", &tags,
		"cache_control?", &cacheControl,
		"content_disposition?", &contentDisposition,
		"content_encoding?", &contentEncoding,
		"content_language?", &contentLanguage,
		"expires?", &expires,
	); err != nil {
		return none, err
	}

	// Parse object options using helper function
	option, err := parseObjectOptions(contentType, cacheControl, contentDisposition, contentEncoding, contentLanguage, expires, metadata, tags)
	if err != nil {
		return none, err
	}

	ctx := dataconv.GetThreadContext(thread)
	err = s.client.SetObjectInfo(ctx, bucket, key, option)
	if err != nil {
		return none, fmt.Errorf("failed to set object info: %w", err)
	}

	return none, nil
}

// copyObject copies an object from one location to another
func (s *ClientWrapper) copyObject(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var (
		srcBucket          = ""
		srcKey             = ""
		dstBucket          = ""
		dstKey             = ""
		contentType        = ""
		metadata           = starlark.NewDict(0)
		tags               = starlark.NewDict(0)
		cacheControl       = ""
		contentDisposition = ""
		contentEncoding    = ""
		contentLanguage    = ""
		expires            = ""
	)

	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"src_bucket", &srcBucket,
		"src_key", &srcKey,
		"dst_bucket", &dstBucket,
		"dst_key", &dstKey,
		"content_type?", &contentType,
		"metadata?", &metadata,
		"tags?", &tags,
		"content_encoding?", &contentEncoding,
		"cache_control?", &cacheControl,
		"content_disposition?", &contentDisposition,
		"content_language?", &contentLanguage,
		"expires?", &expires,
	); err != nil {
		return none, err
	}

	// Parse object options using helper function
	option, err := parseObjectOptions(contentType, cacheControl, contentDisposition, contentEncoding, contentLanguage, expires, metadata, tags)
	if err != nil {
		return none, err
	}

	var opts *ObjectOptions
	if option.Validate() {
		opts = option
	}

	ctx := dataconv.GetThreadContext(thread)
	err = s.client.CopyObject(ctx, srcBucket, srcKey, dstBucket, dstKey, opts)
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

// getClientInfo returns information about the S3 client
func (s *ClientWrapper) getClientInfo(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(b.Name(), args, kwargs); err != nil {
		return none, err
	}

	config := s.client.GetConfig()

	// For security, only show if secrets are set, not their actual values
	accessKeySet := config.AccessKey != ""
	secretKeySet := config.SecretKey != ""
	sessionTokenSet := config.SessionToken != ""

	infoDict := starlark.StringDict{
		"service_type":      starlark.String(config.ServiceType),
		"access_key_set":    starlark.Bool(accessKeySet),
		"secret_key_set":    starlark.Bool(secretKeySet),
		"session_token_set": starlark.Bool(sessionTokenSet),
		"region":            starlark.String(config.Region),
		"endpoint":          starlark.String(config.Endpoint),
		"force_path_style":  starlark.Bool(config.ForcePathStyle),
		"use_ssl":           starlark.Bool(config.UseSSL),
		"timeout":           starlark.MakeInt(config.Timeout),
		"max_retries":       starlark.MakeInt(config.MaxRetries),
		"part_size":         starlark.MakeInt64(config.PartSize),
		"concurrency":       starlark.MakeInt(config.Concurrency),
		"enable_logging":    starlark.Bool(config.EnableLogging),
		"user_agent":        starlark.String(config.UserAgent),
	}

	return starlarkstruct.FromStringDict(starlark.String("ClientInfo"), infoDict), nil
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
