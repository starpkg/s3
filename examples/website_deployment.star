#!/usr/bin/env starcli
"""
Website Static File Deployment Example

This example demonstrates deploying a static website to S3 with:
- Content type detection and mapping
- Cache control headers for CDN optimization
- Asset organization and deployment
- Website configuration setup

Usage: starcli website_deployment.star [bucket-name] [local-directory]
"""

load("s3", "create_client")

def main():
    """Main function for website deployment"""
    
    # Get arguments
    args = runtime.args[1:]
    bucket_name = args[0] if len(args) > 0 else "my-website-bucket"
    local_dir = args[1] if len(args) > 1 else "./dist"
    
    print("=== S3 Website Deployment Demo ===")
    print("Bucket: {}".format(bucket_name))
    print("Local directory: {}".format(local_dir))
    print()
    
    # Create S3 client
    s3 = create_client(region="us-east-1")
    
    # Step 1: Setup bucket for website hosting
    setup_website_bucket(s3, bucket_name)
    
    # Step 2: Deploy website files
    deploy_website_files(s3, bucket_name)
    
    # Step 3: Display website information
    display_website_info(s3, bucket_name)
    
    print("\n=== Website deployment completed! ===")

def setup_website_bucket(s3, bucket_name):
    """Setup bucket for website hosting"""
    print("1. Setting up website bucket...")
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        print("  Creating bucket '{}'...".format(bucket_name))
        s3.create_bucket(bucket_name)
        print("  ✓ Bucket created")
    else:
        print("  ✓ Bucket '{}' already exists".format(bucket_name))
    
    print()

def deploy_website_files(s3, bucket_name):
    """Deploy website files to S3 with optimization"""
    print("2. Deploying website files...")
    
    # Content type mapping
    content_types = {
        ".html": "text/html",
        ".css": "text/css",
        ".js": "application/javascript",
        ".json": "application/json",
        ".png": "image/png",
        ".jpg": "image/jpeg"
    }
    
    # Cache control settings by file type
    cache_settings = {
        ".html": "public, max-age=3600",           # 1 hour for HTML
        ".css": "public, max-age=86400",          # 24 hours for CSS
        ".js": "public, max-age=86400",           # 24 hours for JS
        ".png": "public, max-age=604800",         # 1 week for images
        ".jpg": "public, max-age=604800"          # 1 week for images
    }
    
    # Sample website files to deploy
    files = {
        "index.html": """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>My S3 Website</title>
    <link rel="stylesheet" href="css/style.css">
</head>
<body>
    <h1>Welcome to My S3 Website</h1>
    <p>Deployed with Starlark S3 module!</p>
    <script src="js/app.js"></script>
</body>
</html>""",
        "css/style.css": """body {
    font-family: Arial, sans-serif;
    margin: 40px;
    background-color: #f5f5f5;
}
h1 {
    color: #333;
    text-align: center;
}""",
        "js/app.js": """console.log('S3 Website loaded!');
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM ready');
});"""
    }
    
    uploaded_count = 0
    
    for file_path, content in files.items():
        # Determine content type and cache settings
        file_extension = get_extension(file_path)
        content_type = content_types.get(file_extension, "text/plain")
        cache_control = cache_settings.get(file_extension, "public, max-age=3600")
        
        print("  📤 Uploading: {}".format(file_path))
        print("     Content-Type: {}".format(content_type))
        print("     Cache-Control: {}".format(cache_control))
        
        # Upload with optimization
        s3.put_object(
            bucket_name,
            file_path,
            content,
            content_type=content_type,
            metadata={
                "cache-control": cache_control,
                "deployed-by": "starlark-s3-demo"
            }
        )
        
        uploaded_count = uploaded_count + 1
        print("     ✓ Uploaded successfully")
    
    print("\n  📊 Summary: {} files uploaded successfully".format(uploaded_count))
    print()

def get_extension(file_path):
    """Get file extension from path"""
    parts = file_path.split(".")
    if len(parts) > 1:
        return "." + parts[-1]
    return ""

def display_website_info(s3, bucket_name):
    """Display website information and URLs"""
    print("3. Website deployment information...")
    
    # Get bucket region
    try:
        region = s3.get_bucket_location(bucket_name)
    except Exception:
        region = "us-east-1"  # Default region
    
    print("  📍 Bucket: {}".format(bucket_name))
    print("  🌍 Region: {}".format(region))
    print()
    
    print("  🔗 Access URLs:")
    print("     S3 Direct: https://{}.s3.{}.amazonaws.com/index.html".format(bucket_name, region))
    print("     S3 Website: http://{}.s3-website-{}.amazonaws.com".format(bucket_name, region))
    print()
    
    # List uploaded files
    print("  📄 Deployed files:")
    try:
        result = s3.list_objects(bucket_name)
        for obj in result["contents"]:
            print("     {}".format(obj["key"]))
    except Exception as e:
        print("     Error listing files: {}".format(e))

# Run the deployment
main() 