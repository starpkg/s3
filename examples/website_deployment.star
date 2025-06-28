load("s3", "create_client")
load("file", "exists", "read")
load("path", "join", "ext")

def get_content_type(file_path):
    """Get content type from file extension"""
    content_types = {
        ".html": "text/html",
        ".css": "text/css",
        ".js": "application/javascript",
        ".json": "application/json",
        ".png": "image/png",
        ".jpg": "image/jpeg",
        ".jpeg": "image/jpeg",
        ".gif": "image/gif",
        ".svg": "image/svg+xml",
        ".ico": "image/x-icon",
        ".woff": "font/woff",
        ".woff2": "font/woff2",
        ".txt": "text/plain",
        ".pdf": "application/pdf"
    }
    
    file_ext = ext(file_path)
    return content_types.get(file_ext, "application/octet-stream")

def get_cache_control(file_ext):
    """Get appropriate cache control header based on file type"""
    # Static assets get longer cache times
    if file_ext in [".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2"]:
        return "public, max-age=86400"  # 24 hours
    elif file_ext in [".html"]:
        return "public, max-age=3600"   # 1 hour
    else:
        return "public, max-age=300"    # 5 minutes

def deploy_website(bucket_name, local_dir):
    """Deploy a static website to S3 with proper content types and caching"""
    
    # Create S3 client
    s3 = create_client()
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating bucket: {}".format(bucket_name))
        s3.create_bucket(bucket_name)
    else:
        print("Using existing bucket: {}".format(bucket_name))
    
    # Example files to upload (in a real scenario, you'd scan the directory)
    files_to_upload = [
        "index.html",
        "about.html",
        "contact.html",
        "css/style.css",
        "css/responsive.css",
        "js/app.js",
        "js/utils.js",
        "images/logo.png",
        "images/hero-bg.jpg",
        "images/favicon.ico",
        "fonts/roboto.woff2",
        "manifest.json"
    ]
    
    uploaded_count = 0
    total_size = 0
    
    print("Starting website deployment to s3://{}".format(bucket_name))
    print("-" * 50)
    
    for file_path in files_to_upload:
        local_path = join(local_dir, file_path)
        
        # Check if file exists (in real usage, you'd scan the directory)
        if not exists(local_path):
            print("File not found: {} (skipping)".format(local_path))
            continue
        
        # Read file content
        content = read(local_path)
        file_size = len(content)
        total_size = total_size + file_size
        
        # Determine content type and cache settings
        file_ext = ext(file_path)
        content_type = get_content_type(file_path)
        cache_control = get_cache_control(file_ext)
        
        print("Uploading: {} ({} bytes, {})".format(
            file_path, 
            file_size, 
            content_type
        ))
        
        # Upload with appropriate headers
        s3.put_object(
            bucket_name,
            file_path,
            content,
            content_type=content_type,
            metadata={
                "cache-control": cache_control,
                "deployed-by": "starlark-s3",
                "deploy-time": "2024-01-15T10:30:00Z"
            }
        )
        
        uploaded_count = uploaded_count + 1
    
    print("-" * 50)
    print("Deployment completed!")
    print("Files uploaded: {}".format(uploaded_count))
    print("Total size: {:.2f} KB".format(total_size / 1024))
    
    # Generate website URL for AWS S3 static hosting
    region = s3.get_bucket_location(bucket_name) or "us-east-1"
    website_url = "http://{}.s3-website-{}.amazonaws.com".format(bucket_name, region)
    
    print("\nWebsite URLs:")
    print("S3 Website URL: {}".format(website_url))
    print("Direct S3 URL: https://{}.s3.{}.amazonaws.com/index.html".format(bucket_name, region))
    
    # List deployed files
    print("\nDeployed files:")
    objects = s3.list_objects(bucket_name)
    for obj in objects["contents"]:
        print("  {} ({} bytes)".format(obj["key"], obj["size"]))

def setup_static_website_hosting(bucket_name):
    """Configure bucket for static website hosting (example - actual implementation would depend on service)"""
    s3 = create_client()
    
    print("Note: Static website hosting configuration varies by S3-compatible service.")
    print("For AWS S3, you would typically:")
    print("1. Configure bucket policy for public read access")
    print("2. Enable static website hosting with index.html as index document")
    print("3. Set up custom domain and SSL certificate if needed")
    
    # Example of setting public read policy (pseudo-code)
    print("\nExample bucket policy for public read access:")
    policy = {
        "Version": "2012-10-17",
        "Statement": [
            {
                "Sid": "PublicReadGetObject",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:GetObject",
                "Resource": "arn:aws:s3:::{}/*".format(bucket_name)
            }
        ]
    }
    print("Policy: {}".format(policy))

def main():
    """Main deployment function"""
    
    bucket_name = "my-website-bucket"
    local_directory = "./dist"  # Your built website directory
    
    print("Website Deployment Example")
    print("=" * 40)
    
    # Deploy website files
    deploy_website(bucket_name, local_directory)
    
    # Show static hosting setup info
    print("\n" + "=" * 40)
    setup_static_website_hosting(bucket_name)
    
    print("\nDeployment example completed!")

if __name__ == "__main__":
    main() 