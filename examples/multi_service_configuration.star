load("s3", "create_client", "get_supported_services", "get_client_info")

def test_service_connectivity(client, service_name):
    """Test basic connectivity and operations for a service"""
    
    print("Testing {} connectivity...".format(service_name))
    
    try:
        # Test 1: List buckets (basic connectivity test)
        buckets = client.list_buckets()
        print("  ✓ Connected successfully - {} buckets accessible".format(len(buckets)))
        
        # Test 2: Create a test bucket
        test_bucket = "test-{}-bucket".format(service_name.lower().replace(" ", "-"))
        
        if not client.bucket_exists(test_bucket):
            print("  ✓ Creating test bucket: {}".format(test_bucket))
            client.create_bucket(test_bucket)
        else:
            print("  ✓ Test bucket already exists: {}".format(test_bucket))
        
        # Test 3: Upload test object
        test_key = "test/connectivity-test.txt"
        test_content = "Hello from {} via Starlark S3!".format(service_name)
        
        print("  ✓ Uploading test object...")
        client.put_object(test_bucket, test_key, test_content)
        
        # Test 4: Download and verify
        print("  ✓ Downloading test object...")
        downloaded = client.get_object(test_bucket, test_key)
        
        if downloaded == test_content:
            print("  ✓ Content verification passed")
        else:
            print("  ✗ Content verification failed")
            return False
        
        # Test 5: Cleanup
        print("  ✓ Cleaning up test object...")
        client.delete_object(test_bucket, test_key)
        
        print("  ✓ {} test completed successfully!".format(service_name))
        return True
        
    except Exception as e:
        print("  ✗ {} test failed: {}".format(service_name, e))
        return False

def configure_aws_s3():
    """Configure AWS S3 client"""
    
    print("Configuring AWS S3...")
    
    # AWS S3 with default configuration
    s3 = create_client(
        service_type="aws_s3",
        aws_region="us-east-1",
        # Uses AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY from environment
        timeout=30,
        max_retries=3
    )
    
    return s3

def configure_cloudflare_r2():
    """Configure Cloudflare R2 client"""
    
    print("Configuring Cloudflare R2...")
    
    # Note: Replace with your actual Cloudflare R2 credentials
    s3 = create_client(
        service_type="cloudflare_r2",
        endpoint="https://YOUR_ACCOUNT_ID.r2.cloudflarestorage.com",
        aws_access_key="YOUR_R2_ACCESS_KEY_ID",
        aws_secret_key="YOUR_R2_SECRET_ACCESS_KEY",
        aws_region="auto",  # R2 uses "auto" region
        timeout=30
    )
    
    return s3

def configure_backblaze_b2():
    """Configure Backblaze B2 client"""
    
    print("Configuring Backblaze B2...")
    
    # Note: Replace with your actual Backblaze B2 credentials
    s3 = create_client(
        service_type="backblaze_b2",
        endpoint="https://s3.us-west-004.backblazeb2.com",
        aws_access_key="YOUR_B2_KEY_ID",
        aws_secret_key="YOUR_B2_APPLICATION_KEY",
        aws_region="us-west-004",
        timeout=60  # B2 can be slower
    )
    
    return s3

def configure_digitalocean_spaces():
    """Configure DigitalOcean Spaces client"""
    
    print("Configuring DigitalOcean Spaces...")
    
    # Note: Replace with your actual DigitalOcean Spaces credentials
    s3 = create_client(
        service_type="digitalocean",
        endpoint="https://nyc3.digitaloceanspaces.com",
        aws_access_key="YOUR_SPACES_ACCESS_KEY",
        aws_secret_key="YOUR_SPACES_SECRET_KEY",
        aws_region="nyc3",
        timeout=30
    )
    
    return s3

def configure_minio():
    """Configure MinIO client (local development)"""
    
    print("Configuring MinIO...")
    
    # MinIO local development setup
    s3 = create_client(
        service_type="minio",
        endpoint="http://localhost:9000",
        aws_access_key="minioadmin",
        aws_secret_key="minioadmin",
        aws_region="us-east-1",
        force_path_style=True,  # Required for MinIO
        use_ssl=False,  # For local development
        timeout=30
    )
    
    return s3

def demonstrate_service_features(client, service_name, bucket_name):
    """Demonstrate key features for each service"""
    
    print("\nDemonstrating features for {}...".format(service_name))
    print("-" * 50)
    
    try:
        # Ensure test bucket exists
        if not client.bucket_exists(bucket_name):
            client.create_bucket(bucket_name)
        
        # Feature 1: Object operations with metadata
        print("1. Testing object operations with metadata...")
        client.put_object(
            bucket_name,
            "demo/metadata-test.txt",
            "This is a test file with metadata",
            content_type="text/plain",
            metadata={
                "service": service_name,
                "test-type": "feature-demo",
                "created-by": "starlark-s3"
            }
        )
        
        # Get object info
        info = client.get_object_info(bucket_name, "demo/metadata-test.txt")
        print("   Object size: {} bytes".format(info["size"]))
        print("   Content type: {}".format(info.get("content_type", "unknown")))
        
        # Feature 2: Object listing with prefix
        print("2. Testing object listing...")
        objects = client.list_objects(bucket_name, prefix="demo/")
        print("   Found {} objects with 'demo/' prefix".format(len(objects["contents"])))
        
        # Feature 3: Object copying
        print("3. Testing object copying...")
        client.copy_object(
            bucket_name, "demo/metadata-test.txt",
            bucket_name, "demo/copied-test.txt"
        )
        print("   Successfully copied object")
        
        # Feature 4: Pre-signed URLs (if supported)
        try:
            print("4. Testing pre-signed URLs...")
            url = client.presign_url(bucket_name, "demo/metadata-test.txt", expires_in=3600)
            print("   Generated pre-signed URL: {}".format(url[:50] + "..."))
        except Exception as e:
            print("   Pre-signed URLs not supported or failed: {}".format(e))
        
        # Feature 5: Object tagging (if supported)
        try:
            print("5. Testing object tagging...")
            client.set_object_tags(bucket_name, "demo/metadata-test.txt", {
                "environment": "test",
                "service": service_name.lower(),
                "purpose": "demo"
            })
            
            tags = client.get_object_tags(bucket_name, "demo/metadata-test.txt")
            print("   Set {} tags successfully".format(len(tags)))
        except Exception as e:
            print("   Object tagging not supported: {}".format(e))
        
        # Feature 6: Batch operations
        print("6. Testing batch operations...")
        objects_to_delete = ["demo/metadata-test.txt", "demo/copied-test.txt"]
        result = client.delete_objects(bucket_name, objects_to_delete)
        print("   Deleted {} objects in batch".format(len(result["deleted"])))
        
        print("   ✓ All supported features tested successfully!")
        
    except Exception as e:
        print("   ✗ Feature testing failed: {}".format(e))

def compare_service_performance(services):
    """Basic performance comparison between services"""
    
    print("\nPerformance Comparison")
    print("=" * 60)
    
    test_data = "x" * (1024 * 100)  # 100KB test data
    
    for service_name, client in services.items():
        if client == None:
            continue
        
        try:
            bucket_name = "perf-test-{}".format(service_name.lower().replace(" ", "-"))
            
            if not client.bucket_exists(bucket_name):
                client.create_bucket(bucket_name)
            
            # Time upload operation
            import time
            start_time = time.time()
            
            client.put_object(bucket_name, "perf-test.txt", test_data)
            
            upload_time = time.time() - start_time
            
            # Time download operation  
            start_time = time.time()
            
            downloaded = client.get_object(bucket_name, "perf-test.txt")
            
            download_time = time.time() - start_time
            
            # Cleanup
            client.delete_object(bucket_name, "perf-test.txt")
            
            print("{}: Upload {:.2f}s, Download {:.2f}s".format(
                service_name, upload_time, download_time
            ))
            
        except Exception as e:
            print("{}: Performance test failed - {}".format(service_name, e))

def main():
    """Main multi-service configuration example"""
    
    print("S3 Multi-Service Configuration Example")
    print("=" * 60)
    
    # Show supported services
    print("Supported services:")
    supported = get_supported_services()
    for service in supported:
        print("  - {}".format(service))
    
    print("\n" + "=" * 60)
    
    # Configure all services
    services = {}
    
    try:
        services["AWS S3"] = configure_aws_s3()
    except Exception as e:
        print("Failed to configure AWS S3: {}".format(e))
        services["AWS S3"] = None
    
    try:
        services["Cloudflare R2"] = configure_cloudflare_r2()
    except Exception as e:
        print("Failed to configure Cloudflare R2: {}".format(e))
        services["Cloudflare R2"] = None
    
    try:
        services["Backblaze B2"] = configure_backblaze_b2()
    except Exception as e:
        print("Failed to configure Backblaze B2: {}".format(e))
        services["Backblaze B2"] = None
    
    try:
        services["DigitalOcean Spaces"] = configure_digitalocean_spaces()
    except Exception as e:
        print("Failed to configure DigitalOcean Spaces: {}".format(e))
        services["DigitalOcean Spaces"] = None
    
    try:
        services["MinIO"] = configure_minio()
    except Exception as e:
        print("Failed to configure MinIO: {}".format(e))
        services["MinIO"] = None
    
    # Test connectivity for each service
    print("\nTesting Service Connectivity")
    print("=" * 60)
    
    working_services = {}
    
    for service_name, client in services.items():
        if client != None:
            if test_service_connectivity(client, service_name):
                working_services[service_name] = client
            print()
    
    # Show client information for working services
    print("Working Service Configurations")
    print("=" * 60)
    
    for service_name, client in working_services.items():
        try:
            info = get_client_info(client)
            print("{}:".format(service_name))
            print("  Endpoint: {}".format(info.get("endpoint", "default")))
            print("  Region: {}".format(info.get("region", "default")))
            print("  SSL: {}".format(info.get("use_ssl", True)))
            print()
        except Exception as e:
            print("{}: Failed to get client info - {}".format(service_name, e))
    
    # Demonstrate features for each working service
    for service_name, client in working_services.items():
        bucket_name = "demo-{}".format(service_name.lower().replace(" ", "-"))
        demonstrate_service_features(client, service_name, bucket_name)
    
    # Performance comparison
    if len(working_services) > 1:
        compare_service_performance(working_services)
    
    print("\nMulti-service configuration example completed!")

if __name__ == "__main__":
    main() 