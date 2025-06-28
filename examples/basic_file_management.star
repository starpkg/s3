load("s3", "create_client")

def main():
    """Basic file management operations with S3"""
    
    # Create S3 client (uses environment variables by default)
    s3 = create_client(aws_region="us-east-1")
    
    bucket_name = "my-files-bucket"
    
    # Ensure bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating bucket: {}".format(bucket_name))
        s3.create_bucket(bucket_name)
    else:
        print("Bucket already exists: {}".format(bucket_name))
    
    # Upload a simple text file
    print("Uploading text file...")
    s3.put_object(bucket_name, "hello.txt", "Hello from Starlark!")
    
    # Upload with metadata
    print("Uploading PDF with metadata...")
    s3.put_object(
        bucket_name,
        "report.pdf",
        "PDF content here...",
        content_type="application/pdf",
        metadata={
            "author": "John Doe",
            "created": "2024-01-15",
            "version": "1.0"
        }
    )
    
    # List all objects in bucket
    print("\nObjects in bucket:")
    objects = s3.list_objects(bucket_name)
    for obj in objects["contents"]:
        print("  {} ({} bytes, modified: {})".format(
            obj["key"], 
            obj["size"], 
            obj["last_modified"]
        ))
    
    # Download and print content
    print("\nDownloading text file...")
    content = s3.get_object(bucket_name, "hello.txt")
    print("Downloaded content: {}".format(content))
    
    # Get object metadata
    print("\nGetting PDF metadata...")
    metadata = s3.get_object_metadata(bucket_name, "report.pdf")
    print("Author: {}".format(metadata.get("author", "Unknown")))
    print("Version: {}".format(metadata.get("version", "Unknown")))
    
    # Generate a download link
    print("\nGenerating pre-signed URL...")
    url = s3.presign_url(bucket_name, "report.pdf", expires_in=3600)
    print("Download URL (expires in 1 hour): {}".format(url))
    
    # Copy object to new location
    print("\nCopying object...")
    s3.copy_object(bucket_name, "hello.txt", bucket_name, "backup/hello_backup.txt")
    print("Copied hello.txt to backup/hello_backup.txt")
    
    # List objects with prefix
    print("\nListing backup objects:")
    backup_objects = s3.list_objects(bucket_name, prefix="backup/")
    for obj in backup_objects["contents"]:
        print("  {} ({} bytes)".format(obj["key"], obj["size"]))
    
    print("\nBasic file management example completed!")

if __name__ == "__main__":
    main() 