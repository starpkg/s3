package s3

import (
	"os"
	"testing"

	"github.com/1set/starlet"
	"github.com/starpkg/base"
)

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
