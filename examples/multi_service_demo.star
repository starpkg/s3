#!/usr/bin/env starcli
"""
Multi-Service S3 Demo

This example demonstrates working with multiple S3-compatible services:
- Amazon S3
- Cloudflare R2
- Backblaze B2
- DigitalOcean Spaces
- MinIO

Shows configuration differences and unified API usage across services.

Usage: starcli multi_service_demo.star
"""

load("s3", "create_client", "get_supported_services")

def main():
    """Demonstrate multi-service S3 operations"""
    
    print("=== Multi-Service S3 Demo ===")
    print()
    
    # Show supported services
    services = get_supported_services()
    print("Supported services:")
    for service in services:
        print("   ✓ {}".format(service))
    print()
    
    # Test AWS S3
    test_aws_s3()
    
    # Test MinIO  
    test_minio()
    
    # Test other services (if configured)
    test_other_services()
    
    print("=== Multi-service demo completed! ===")

def test_aws_s3():
    """Test AWS S3 operations"""
    print("1. Testing AWS S3...")
    
    aws_access_key = runtime.getenv("AWS_ACCESS_KEY_ID")
    if not aws_access_key:
        print("   ⚠️  AWS credentials not found (set AWS_ACCESS_KEY_ID)")
        return
    
    try:
        s3 = create_client(
            service_type="aws_s3",
            region="us-east-1"
        )
        
        test_basic_operations(s3, "aws-test-bucket", "AWS S3")
        
    except Exception as e:
        print("   ❌ AWS S3 test failed: {}".format(e))

def test_minio():
    """Test MinIO operations"""
    print("2. Testing MinIO...")
    
    try:
        s3 = create_client(
            service_type="minio",
            endpoint="http://localhost:9000",
            access_key="minioadmin",
            secret_key="minioadmin",
            force_path_style=True,
            use_ssl=False
        )
        
        test_basic_operations(s3, "minio-test-bucket", "MinIO")
        
    except Exception as e:
        print("   ❌ MinIO test failed: {}".format(e))
        print("   💡 Make sure MinIO is running on localhost:9000")

def test_other_services():
    """Test other S3-compatible services if configured"""
    print("3. Testing other services...")
    
    # Cloudflare R2
    r2_endpoint = runtime.getenv("R2_ENDPOINT")
    if r2_endpoint:
        test_cloudflare_r2()
    else:
        print("   ⚠️  Cloudflare R2 not configured")
    
    # DigitalOcean Spaces
    spaces_endpoint = runtime.getenv("SPACES_ENDPOINT")
    if spaces_endpoint:
        test_digitalocean_spaces()
    else:
        print("   ⚠️  DigitalOcean Spaces not configured")

def test_cloudflare_r2():
    """Test Cloudflare R2"""
    try:
        s3 = create_client(
            service_type="cloudflare_r2",
            endpoint=runtime.getenv("R2_ENDPOINT"),
            access_key=runtime.getenv("R2_ACCESS_KEY"),
            secret_key=runtime.getenv("R2_SECRET_KEY"),
            region="auto"
        )
        
        test_basic_operations(s3, "r2-test-bucket", "Cloudflare R2")
        
    except Exception as e:
        print("   ❌ Cloudflare R2 test failed: {}".format(e))

def test_digitalocean_spaces():
    """Test DigitalOcean Spaces"""
    try:
        s3 = create_client(
            service_type="digitalocean_spaces",
            endpoint=runtime.getenv("SPACES_ENDPOINT"),
            access_key=runtime.getenv("SPACES_ACCESS_KEY"),
            secret_key=runtime.getenv("SPACES_SECRET_KEY"),
            region="nyc3"
        )
        
        test_basic_operations(s3, "spaces-test-bucket", "DigitalOcean Spaces")
        
    except Exception as e:
        print("   ❌ DigitalOcean Spaces test failed: {}".format(e))

def test_basic_operations(client, bucket_name, service_name):
    """Test basic operations on any S3-compatible service"""
    print("   Testing {} operations...".format(service_name))
    
    try:
        # Test connection
        buckets = client.list_buckets()
        print("     ✓ Connection successful ({} buckets)".format(len(buckets)))
        
        # Ensure bucket exists
        if not client.bucket_exists(bucket_name):
            client.create_bucket(bucket_name)
            print("     ✓ Created test bucket")
        
        # Test upload/download
        test_key = "test-file.txt"
        test_content = "Hello from {}!".format(service_name)
        
        client.put_object(bucket_name, test_key, test_content)
        print("     ✓ File uploaded")
        
        downloaded = client.get_object(bucket_name, test_key)
        if downloaded == test_content:
            print("     ✓ File downloaded and verified")
        
        # Test metadata
        info = client.get_object_info(bucket_name, test_key)
        print("     ✓ Object info retrieved ({} bytes)".format(info["size"]))
        
        # Test pre-signed URL (if supported)
        try:
            url = client.presign_url(bucket_name, test_key, expires_in=3600)
            print("     ✓ Pre-signed URL generated")
        except Exception:
            print("     ⚠️  Pre-signed URLs not supported")
        
        # Cleanup
        client.delete_object(bucket_name, test_key)
        print("     ✓ File deleted")
        
    except Exception as e:
        print("     ❌ Operation failed: {}".format(e))

# Run the demo
main() 