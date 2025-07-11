package s3

import (
	"testing"

	"github.com/1set/starlet"
	"github.com/starpkg/base"
)

// TestStarlarkScripts runs Starlark test scripts from the test directory.
// Scripts with "test-" prefix should succeed, "panic-" prefix should fail.
func TestStarlarkScripts(t *testing.T) {
	// Create a module factory function that returns a fresh module loader for each test
	moduleFactory := func() starlet.ModuleLoader {
		return NewModule().LoadModule()
	}
	extraModules := []string{"go_idiomatic", "http", "json", "file", "path"}

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
        access_key="",
        secret_key="",
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
load("s3", "parse_s3_url", "generate_s3_url", "validate_bucket_name", "validate_object_key", "get_supported_services")

def test_utility_functions():
    # Test S3 URL parsing
    result = parse_s3_url("s3://my-bucket/path/to/file.txt")
    if result["bucket"] != "my-bucket":
        fail("URL parsing failed for bucket")
    if result["key"] != "path/to/file.txt":
        fail("URL parsing failed for key")

    # Test S3 URL generation
    url = generate_s3_url("test-bucket", "test-key.txt")
    if url != "s3://test-bucket/test-key.txt":
        fail("URL generation failed")

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
        access_key="test-key",
        secret_key="test-secret",
        use_ssl=True,
        timeout=60,
        max_retries=5,
        part_size=8388608,  # 8MB
        concurrency=5,
        user_agent="test-agent",
    )

    # Get client configuration
    config = client.get_config()
    if config == None:
        fail("Failed to get client configuration")

    print("Client configuration retrieved successfully!")

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
        access_key="minioadmin",
        secret_key="minioadmin",
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

    if not hasattr(client, "get_bucket_location"):
        fail("get_bucket_location method not found")

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
        access_key="minioadmin",
        secret_key="minioadmin",
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

func TestS3URLParsing(t *testing.T) {
	// Test different URL formats
	script := `
load("s3", "parse_s3_url")

def test_url_parsing():
    # Test s3:// URL
    result = parse_s3_url("s3://my-bucket/path/to/file.txt")
    if result["bucket"] != "my-bucket" or result["key"] != "path/to/file.txt":
        fail("s3:// URL parsing failed")

    # Test s3:// URL without key
    result = parse_s3_url("s3://my-bucket")
    if result["bucket"] != "my-bucket" or result["key"] != "":
        fail("s3:// URL without key parsing failed")

    # Test HTTPS URL (virtual-hosted style)
    result = parse_s3_url("https://my-bucket.s3.amazonaws.com/path/to/file.txt")
    if result["bucket"] != "my-bucket" or result["key"] != "path/to/file.txt":
        fail("HTTPS virtual-hosted URL parsing failed")

    # Test HTTPS URL (path style)
    result = parse_s3_url("https://s3.amazonaws.com/my-bucket/path/to/file.txt")
    if result["bucket"] != "my-bucket" or result["key"] != "path/to/file.txt":
        fail("HTTPS path-style URL parsing failed")

    print("All URL parsing tests passed!")

test_url_parsing()
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
