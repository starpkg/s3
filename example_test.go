package s3

import (
	"os"
	"strings"
	"testing"

	"github.com/1set/starlet"
	"github.com/starpkg/base"
	"go.starlark.net/starlark"
)

// runS3Script loads the s3 module and runs the given script content, returning
// any execution error. It is the shared harness for the offline error-branch
// sections below — none of these scripts touch the network: they exercise the
// argument-parsing/validation/conversion code that runs before any AWS call.
func runS3Script(t *testing.T, script string) error {
	t.Helper()
	runner := starlet.NewDefault()
	loaders := map[string]starlet.ModuleLoader{ModuleName: NewModule().LoadModule()}
	runner.SetLazyloadModules(loaders)
	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	return err
}

// TestStarlarkScripts runs the Starlark integration scripts from the private
// fixtures directory (../test/s3). They exercise live S3-compatible endpoints
// and therefore need real, host-injected credentials (PKG-15: credentials are
// never passed from a script). They are opt-in: set S3_RUN_INTEGRATION=1 (with
// real S3_ACCESS_KEY / S3_SECRET_KEY) to run them. Unit coverage for detection,
// validation, and the client API lives in detection_test.go and the offline
// Test functions in this file, so default `go test` stays hermetic.
func TestStarlarkScripts(t *testing.T) {
	if os.Getenv("S3_RUN_INTEGRATION") == "" {
		t.Skip("set S3_RUN_INTEGRATION=1 (with real credentials) to run live S3 integration scripts")
	}
	// Create a module factory function that returns a fresh module loader for each test
	moduleFactory := func() starlet.ModuleLoader {
		return NewModule().LoadModule()
	}
	extraModules := []string{"go_idiomatic", "http", "json", "file", "path", "random", "time"}

	// Use the helper function from the base package
	base.RunStarlarkTests(t, ModuleName, moduleFactory, extraModules, "")
}

func TestS3Module(t *testing.T) {
	// Test that the module can be created and loaded
	module := NewModule()
	if module == nil {
		t.Fatal("Failed to create S3 module")
	}

	loader := module.LoadModule()
	if loader == nil {
		t.Fatal("Failed to load S3 module")
	}
}

func TestS3ClientCreation(t *testing.T) {
	// Test creating an S3 client without credentials (should work for basic functions)
	script := `
load("s3", "create_client")

def test_client_creation():
    # Create a client without credentials
    client = create_client(
        service_type="minio",
        endpoint="localhost:9000",
        use_ssl=False,
    )

    # Check if client was created
    if client == None:
        fail("Client creation failed")

    print("S3 client created successfully!")

test_client_creation()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

// TestS3ClientInfoAndPublicURL covers the two client methods the README
// documents as the client-config surface: get_client_info() (which exposes the
// resolved non-secret config plus *_set booleans, never the secret values) and
// get_public_url(bucket, key) (which reads region/endpoint/service_type from the
// client's own config). It also pins the env-var idiom: S3_* overrides resolve
// through genConfigOption, where the env var is S3_ + the uppercased option
// name.
func TestS3ClientInfoAndPublicURL(t *testing.T) {
	// Host-injected, non-secret config via S3_* env vars. genConfigOption maps
	// "region" -> S3_REGION and "user_agent" -> S3_USER_AGENT.
	t.Setenv("S3_REGION", "eu-central-1")
	t.Setenv("S3_USER_AGENT", "regression-agent/9.9")

	script := `
load("s3", "create_client")

def test_client_info_and_public_url():
    client = create_client(service_type="aws")

    info = client.get_client_info()
    if info.service_type != "aws":
        fail("service_type not reported: " + info.service_type)
    if info.region != "eu-central-1":
        fail("S3_REGION env var did not resolve through genConfigOption: " + info.region)
    if info.user_agent != "regression-agent/9.9":
        fail("S3_USER_AGENT env var did not resolve: " + info.user_agent)
    # Secrets must be reported only as presence booleans, never as values.
    if info.access_key_set != False:
        fail("access_key_set should be False with no credentials injected")

    url = client.get_public_url("my-bucket", "path/to/file.txt")
    if "my-bucket" not in url or "path/to/file.txt" not in url:
        fail("public URL missing bucket/key: " + url)

    print("client info + public url ok")

test_client_info_and_public_url()
`

	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	if _, err := runner.Run(); err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3UtilityFunctions(t *testing.T) {
	// Test utility functions
	script := `
load("s3", "validate_bucket_name", "validate_object_key", "get_supported_services")

def test_utility_functions():
    # Test bucket name validation
    if not validate_bucket_name("valid-bucket-name"):
        fail("Valid bucket name rejected")

    if validate_bucket_name("Invalid-Bucket-Name"):
        fail("Invalid bucket name accepted")

    # Test object key validation
    if not validate_object_key("valid/object/key.txt"):
        fail("Valid object key rejected")

    # Test getting supported services
    services = get_supported_services()
    if len(services) == 0:
        fail("No supported services returned")

    if "aws" not in services:
        fail("AWS not in supported services")

    print("All utility functions work correctly!")

test_utility_functions()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3ClientConfiguration(t *testing.T) {
	// Test client configuration
	script := `
load("s3", "create_client")

def test_client_configuration():
    # Create a client with various configuration options
    client = create_client(
        service_type="aws",
        region="us-west-2",
        endpoint="",
        use_ssl=True,
        timeout=60,
        max_retries=5,
        part_size=8388608,  # 8MB
        concurrency=5,
        user_agent="test-agent",
    )

    # Check if client was created successfully (client configuration access removed)
    if client == None:
        fail("Failed to create client")

    print("Client configuration test completed successfully!")

test_client_configuration()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3BucketOperations(t *testing.T) {
	// Test bucket operations (these will fail without valid credentials, but should test the API)
	script := `
load("s3", "create_client")

def test_bucket_operations():
    # Create a client (will fail actual operations without credentials)
    client = create_client(
        service_type="minio",
        endpoint="localhost:9000",
        use_ssl=False,
    )

    # Test bucket methods exist
    if not hasattr(client, "create_bucket"):
        fail("create_bucket method not found")

    if not hasattr(client, "delete_bucket"):
        fail("delete_bucket method not found")

    if not hasattr(client, "list_buckets"):
        fail("list_buckets method not found")

    if not hasattr(client, "bucket_exists"):
        fail("bucket_exists method not found")

    if not hasattr(client, "get_bucket_info"):
        fail("get_bucket_info method not found")

    print("All bucket operation methods are available!")

test_bucket_operations()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3ObjectOperations(t *testing.T) {
	// Test object operations (these will fail without valid credentials, but should test the API)
	script := `
load("s3", "create_client")

def test_object_operations():
    # Create a client (will fail actual operations without credentials)
    client = create_client(
        service_type="minio",
        endpoint="localhost:9000",
        use_ssl=False,
    )

    # Test object methods exist
    if not hasattr(client, "put_object"):
        fail("put_object method not found")

    if not hasattr(client, "get_object"):
        fail("get_object method not found")

    if not hasattr(client, "delete_object"):
        fail("delete_object method not found")

    if not hasattr(client, "list_objects"):
        fail("list_objects method not found")

    if not hasattr(client, "object_exists"):
        fail("object_exists method not found")

    if not hasattr(client, "get_object_info"):
        fail("get_object_info method not found")

    print("All object operation methods are available!")

test_object_operations()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3EnhancedObjectOperations(t *testing.T) {
	// Test enhanced object operations including file operations, set_object_info, and copy_object
	script := `
load("s3", "create_client")

def test_enhanced_object_operations():
    # Create a client (will fail actual operations without credentials)
    client = create_client(
        service_type="minio",
        endpoint="localhost:9000",
        use_ssl=False,
    )

    # Test enhanced object methods exist
    if not hasattr(client, "put_object_file"):
        fail("put_object_file method not found")

    if not hasattr(client, "get_object_file"):
        fail("get_object_file method not found")

    if not hasattr(client, "set_object_info"):
        fail("set_object_info method not found")

    if not hasattr(client, "copy_object"):
        fail("copy_object method not found")

    print("All enhanced object operation methods are available!")

test_enhanced_object_operations()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3ModuleLevelConfiguration(t *testing.T) {
	// Test module-level configuration features
	script := `
load("s3", "create_client")

def test_module_configuration():
    # Test creating client with minimal parameters (relies on module defaults)
    client1 = create_client()
    if client1 == None:
        fail("Failed to create client with module defaults")

    # Test creating client with some overrides
    client2 = create_client(service_type="minio", region="us-west-1")
    if client2 == None:
        fail("Failed to create client with partial overrides")

    # Test boolean parameter handling with nullable types
    client3 = create_client(use_ssl=False, force_path_style=True)
    if client3 == None:
        fail("Failed to create client with boolean overrides")

    print("Module-level configuration test passed!")

test_module_configuration()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

func TestS3ValidationFunctions(t *testing.T) {
	// Test validation functions with edge cases
	script := `
load("s3", "validate_bucket_name", "validate_object_key")

def test_validation_functions():
    # Test valid bucket names
    valid_names = ["my-bucket", "test123", "bucket-with-dots.example", "a" * 3, "a" * 63]
    for name in valid_names:
        if not validate_bucket_name(name):
            fail("Valid bucket name rejected: " + name)

    # Test invalid bucket names
    invalid_names = ["My-Bucket", "bucket_with_underscores", "ab", "a" * 64, "192.168.1.1", "xn--example", "bucket-s3alias"]
    for name in invalid_names:
        if validate_bucket_name(name):
            fail("Invalid bucket name accepted: " + name)

    # Test valid object keys
    valid_keys = ["file.txt", "path/to/file.txt", "a" * 1024, "file with spaces.txt"]
    for key in valid_keys:
        if not validate_object_key(key):
            fail("Valid object key rejected: " + key)

    # Test invalid object keys
    invalid_keys = ["", "a" * 1025]  # Empty and too long
    for key in invalid_keys:
        if validate_object_key(key):
            fail("Invalid object key accepted: " + key)

    print("All validation function tests passed!")

test_validation_functions()
`

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}

// TestS3ScriptErrorBranches drives the error paths of the module's builtins
// that execute *before* any network call: missing/extra arguments, the
// host-injected-credentials rule (PKG-15), and the offline conversion/parse
// failures (RFC3339 expires, non-string metadata/tags, unsupported presign
// method). Each must surface as a clean Starlark-level error — never a host
// panic (invariant 3). Building a MinIO/AWS client locally needs no network,
// and the failing call here is the offline arg/option parsing inside the
// builtin, so these are hermetic.
func TestS3ScriptErrorBranches(t *testing.T) {
	const mkClient = `
load("s3", "create_client")
c = create_client(service_type="minio", endpoint="localhost:9000", use_ssl=False)
`
	cases := []struct {
		name   string
		script string
		want   string // substring expected in the error
	}{
		{
			name:   "credentials are never script kwargs (PKG-15)",
			script: `load("s3", "create_client")` + "\n" + `create_client(access_key="AKIA...")`,
			want:   `unexpected keyword argument "access_key"`,
		},
		{
			name:   "secret_key kwarg rejected (PKG-15)",
			script: `load("s3", "create_client")` + "\n" + `create_client(secret_key="x")`,
			want:   `unexpected keyword argument "secret_key"`,
		},
		{
			name:   "session_token kwarg rejected (PKG-15)",
			script: `load("s3", "create_client")` + "\n" + `create_client(session_token="x")`,
			want:   `unexpected keyword argument "session_token"`,
		},
		{
			name:   "put_object missing required args",
			script: mkClient + `c.put_object()`,
			want:   "missing argument for bucket",
		},
		{
			name:   "create_bucket missing bucket",
			script: mkClient + `c.create_bucket()`,
			want:   "missing argument for bucket",
		},
		{
			name:   "copy_object missing args",
			script: mkClient + `c.copy_object("sb", "sk")`,
			want:   "missing argument for dst_bucket",
		},
		{
			name:   "put_object invalid expires time",
			script: mkClient + `c.put_object("b", "k", "data", expires="not-a-time")`,
			want:   "failed to convert expires time",
		},
		{
			name:   "put_object non-string metadata value",
			script: mkClient + `c.put_object("b", "k", "data", metadata={"n": 5})`,
			want:   "value must be a string",
		},
		{
			name:   "put_object non-string metadata key",
			script: mkClient + `c.put_object("b", "k", "data", metadata={5: "v"})`,
			want:   "key must be a string",
		},
		{
			name:   "set_object_info non-string tag value",
			script: mkClient + `c.set_object_info("b", "k", tags={"x": True})`,
			want:   "value must be a string",
		},
		{
			name:   "presign_url unsupported method",
			script: mkClient + `c.presign_url("b", "k", method="DELETE")`,
			want:   "unsupported method: DELETE",
		},
		{
			name:   "presign_url expires_in out of 64-bit range",
			script: mkClient + `c.presign_url("b", "k", expires_in=100000000000000000000000)`,
			want:   "out of range",
		},
		{
			// In-range int64 that overflows when multiplied by time.Second
			// (1<<40 s * 1e9 ns) wraps to a negative time.Duration. This must
			// surface as a clean error from the AWS SDK presign guard, not a
			// host panic or a silently-corrupt presigned URL (invariant 3).
			name:   "presign_url expires_in overflows time.Duration",
			script: mkClient + `c.presign_url("b", "k", expires_in=1099511627776)`,
			want:   "duration must be 0 or greater",
		},
		{
			name:   "client is unhashable",
			script: mkClient + `d = {c: 1}`,
			want:   "unhashable type: s3.Client",
		},
		{
			name:   "client has no such attribute",
			script: mkClient + `c.no_such_method`,
			want:   "no .no_such_method attribute",
		},
		{
			name:   "validate_bucket_name missing arg",
			script: `load("s3", "validate_bucket_name")` + "\n" + `validate_bucket_name()`,
			want:   "missing argument for name",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := runS3Script(t, c.script)
			if err == nil {
				t.Fatalf("expected an error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), c.want)
			}
		})
	}
}

// TestS3GetClientInfoNeverEchoesSecrets pins invariant 2: get_client_info
// reports only *_set booleans for the host-injected credentials, never the
// secret values themselves. Credentials are injected via S3_* env vars (signing
// is offline; the values are never sent anywhere here).
func TestS3GetClientInfoNeverEchoesSecrets(t *testing.T) {
	const secret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	t.Setenv("S3_ACCESS_KEY", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("S3_SECRET_KEY", secret)
	t.Setenv("S3_SESSION_TOKEN", "tok-12345")

	script := `
load("s3", "create_client")

def check():
    info = create_client(service_type="aws").get_client_info()
    # Presence is reported...
    if not info.access_key_set:
        fail("access_key_set should be True")
    if not info.secret_key_set:
        fail("secret_key_set should be True")
    if not info.session_token_set:
        fail("session_token_set should be True")
    # ...but the struct must not carry the secret values at all.
    if hasattr(info, "access_key"):
        fail("info must not expose access_key")
    if hasattr(info, "secret_key"):
        fail("info must not expose secret_key")
    if hasattr(info, "session_token"):
        fail("info must not expose session_token")

check()
`
	if err := runS3Script(t, script); err != nil {
		t.Fatalf("script failed: %v", err)
	}

	// Belt and suspenders at the Go level: render the whole struct and confirm
	// the secret value never appears in any field string.
	cfg := &ClientConfig{ServiceType: ProviderAWS, Region: "us-east-1", SecretKey: secret, AccessKey: "AKIA", SessionToken: "tok"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validate: %v", err)
	}
	client, err := NewClient(nil, cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cw := NewClientWrapper(client)
	infoBuiltin, err := cw.Attr("get_client_info")
	if err != nil {
		t.Fatalf("Attr: %v", err)
	}
	builtin, ok := infoBuiltin.(*starlark.Builtin)
	if !ok {
		t.Fatalf("get_client_info is not a builtin: %T", infoBuiltin)
	}
	v, err := starlark.Call(&starlark.Thread{}, builtin, nil, nil)
	if err != nil {
		t.Fatalf("get_client_info call: %v", err)
	}
	if strings.Contains(v.String(), secret) {
		t.Fatalf("get_client_info struct leaked the secret value: %s", v.String())
	}
}

func TestS3PresignURL(t *testing.T) {
	// Test presign URL functionality
	script := `
load("s3", "create_client")

def test_presign_url():
    # Create a client (presigning doesn't require actual credentials to work)
    client = create_client(
        service_type="aws",
        region="us-west-2",
    )

    # Test presign_url method exists
    if not hasattr(client, "presign_url"):
        fail("presign_url method not found")

    # Test that presign_url method can be called (may fail due to credentials but method should exist)
    get_url = ""
    try_presign = True
    
    if try_presign:
        # The method will likely fail due to invalid credentials, but that's expected
        # We just want to verify the method exists and has the right signature
        get_url = client.presign_url("test-bucket", "test-file.txt")
        print("✓ GET presigned URL generated successfully: " + get_url[:50] + "...")
    
    # Test HEAD method presigning 
    head_url = client.presign_url("test-bucket", "test-file.txt", expires_in=7200, method="HEAD")
    print("✓ HEAD presigned URL generated successfully: " + head_url[:50] + "...")

    print("Presign URL functionality test completed!")

test_presign_url()
`

	// Presigning needs credentials to sign the request. They are injected by the
	// HOST via the S3_* environment variables (or the AWS default chain), never
	// passed from the script. Signing is local, so dummy values are fine here.
	t.Setenv("S3_ACCESS_KEY", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("S3_SECRET_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	// Run the script
	runner := starlet.NewDefault()
	loaders := make(map[string]starlet.ModuleLoader)
	loaders[ModuleName] = NewModule().LoadModule()
	runner.SetLazyloadModules(loaders)

	runner.SetScriptContent([]byte(script))
	_, err := runner.Run()
	if err != nil {
		t.Fatalf("Script execution failed: %v", err)
	}
}
