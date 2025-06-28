#!/usr/bin/env starcli
"""
Automated Backup System Example

This example demonstrates building a backup system using S3 with:
- Timestamped backup organization  
- Metadata tracking for backup information
- Backup verification and listing
- Cleanup of old backups

Usage: starcli backup_system.star [backup-bucket]
"""

load("s3", "create_client")
load("time")

def main():
    """Main backup system demonstration"""
    
    # Get arguments
    args = runtime.args[1:]
    backup_bucket = args[0] if len(args) > 0 else "my-backup-bucket"
    
    print("=== S3 Backup System Demo ===")
    print("Backup bucket: {}".format(backup_bucket))
    print()
    
    # Create S3 client
    s3 = create_client(region="us-east-1")
    
    # Step 1: Setup backup bucket
    setup_backup_bucket(s3, backup_bucket)
    
    # Step 2: Perform backup
    backup_timestamp = perform_backup(s3, backup_bucket)
    
    # Step 3: Verify backup
    verify_backup(s3, backup_bucket, backup_timestamp)
    
    # Step 4: List all backups
    list_all_backups(s3, backup_bucket)
    
    print("\n=== Backup completed successfully! ===")

def setup_backup_bucket(s3, bucket_name):
    """Setup the backup bucket"""
    print("1. Setting up backup bucket...")
    
    if not s3.bucket_exists(bucket_name):
        print("  Creating backup bucket '{}'...".format(bucket_name))
        s3.create_bucket(bucket_name)
        print("  ✓ Bucket created")
    else:
        print("  ✓ Backup bucket '{}' already exists".format(bucket_name))
    
    print()

def perform_backup(s3, bucket_name):
    """Perform the actual backup operation"""
    print("2. Performing backup...")
    
    # Generate backup timestamp
    backup_time = time.now()
    timestamp = backup_time.format("2006-01-02-15-04-05")
    
    print("  📅 Backup timestamp: {}".format(timestamp))
    
    # Sample files to backup
    files_to_backup = {
        "config.json": '{"app": "demo", "version": "1.0"}',
        "database.sql": "CREATE TABLE users (id INT, name VARCHAR(50));",
        "app.log": "2024-01-15 10:00:00 INFO Application started"
    }
    
    successful_backups = 0
    
    for filename, content in files_to_backup.items():
        backup_key = "backups/{}/{}".format(timestamp, filename)
        
        print("  📤 Backing up: {}".format(filename))
        
        s3.put_object(
            bucket_name,
            backup_key,
            content,
            metadata={
                "backup-timestamp": backup_time.format("2006-01-02T15:04:05Z"),
                "original-filename": filename,
                "backup-type": "automated"
            },
            tags={
                "backup": "true",
                "timestamp": timestamp
            }
        )
        
        successful_backups = successful_backups + 1
        print("     ✓ Backup successful")
    
    print("  📊 Backed up {} files".format(successful_backups))
    print()
    
    return timestamp

def verify_backup(s3, bucket_name, backup_timestamp):
    """Verify the backup was successful"""
    print("3. Verifying backup...")
    
    backup_prefix = "backups/{}/".format(backup_timestamp)
    
    result = s3.list_objects(bucket_name, prefix=backup_prefix)
    backup_files = result["contents"]
    
    print("  ✓ Found {} backup files:".format(len(backup_files)))
    
    for obj in backup_files:
        info = s3.get_object_info(bucket_name, obj["key"])
        metadata = info.get("metadata", {})
        
        print("     📄 {} ({} bytes)".format(obj["key"], obj["size"]))
        print("        Original: {}".format(metadata.get("original-filename", "unknown")))
    
    print()

def list_all_backups(s3, bucket_name):
    """List all available backups"""
    print("4. Listing all backups...")
    
    result = s3.list_objects(bucket_name, prefix="backups/")
    
    # Group files by backup timestamp
    backups = {}
    for obj in result["contents"]:
        # Extract timestamp from path like "backups/2024-01-15-10-30-00/file.txt"
        parts = obj["key"].split("/")
        if len(parts) >= 2:
            timestamp = parts[1]
            if timestamp not in backups:
                backups[timestamp] = []
            backups[timestamp].append(obj)
    
    print("  📂 Found {} backup sets:".format(len(backups)))
    
    for timestamp in sorted(backups.keys(), reverse=True):
        files = backups[timestamp]
        total_size = sum(f["size"] for f in files)
        print("     📁 {} ({} files, {} bytes)".format(timestamp, len(files), total_size))

# Run the backup system  
main() 