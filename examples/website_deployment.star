load("s3", "create_client")
load("file", "exists", "read")
load("path", "join", "ext")

def deploy_website(bucket_name, local_dir):
    """Deploy a static website to S3"""
    
    s3 = create_client()
    
    # Content type mapping
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
        ".woff2": "font/woff2"
    }
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating bucket:", bucket_name)
        s3.create_bucket(bucket_name)
    
    # Files to upload (simplified - in real usage you'd scan directory)
    files_to_upload = [
        "index.html",
        "about.html",
        "css/style.css",
        "js/app.js",
        "images/logo.png"
    ]
    
    uploaded_count = 0
    
    for file_path in files_to_upload:
        local_path = join(local_dir, file_path)
        
        if not exists(local_path):
            print("File not found:", local_path)
            continue
        
        # Determine content type
        file_ext = ext(file_path)
        content_type = content_types.get(file_ext, "application/octet-stream")
        
        # Set cache control for static assets
        cache_control = "public, max-age=3600"
        if file_ext in [".css", ".js", ".png", ".jpg", ".jpeg", ".gif"]:
            cache_control = "public, max-age=86400"  # 24 hours
        
        print("Uploading: {} -> s3://{}/{}".format(local_path, bucket_name, file_path))
        
        s3.put_object_from_file(
            bucket_name,
            file_path,
            local_path,
            content_type=content_type,
            metadata={"cache-control": cache_control}
        )
        uploaded_count = uploaded_count + 1
    
    print("Successfully uploaded {} files".format(uploaded_count))
    
    # Generate website URL (if using S3 website hosting)
    website_url = "http://{}.s3-website-{}.amazonaws.com".format(
        bucket_name, 
        s3.get_bucket_location(bucket_name) or "us-east-1"
    )
    print("Website URL:", website_url)

def main():
    deploy_website("my-website-bucket", "./dist")

main() 