load("s3", "create_client")
load("time", "now")
load("file", "read", "exists")
load("path", "join")

def backup_files(bucket_name, files_to_backup):
    """Backup files to S3 with timestamp and metadata"""
    
    s3 = create_client()
    current_time = now()
    timestamp = current_time.format("2006-01-02-15-04-05")
    
    # Ensure backup bucket exists
    if not s3.bucket_exists(bucket_name):
        print("Creating backup bucket: {}".format(bucket_name))
        s3.create_bucket(bucket_name)
    else:
        print("Using existing backup bucket: {}".format(bucket_name))
    
    backup_results = {
        "success": [],
        "failed": [],
        "total_size": 0
    }
    
    print("Starting backup at {}".format(current_time.format("2006-01-02 15:04:05")))
    print("-" * 60)
    
    for local_file in files_to_backup:
        try:
            if not exists(local_file):
                print("File not found, skipping: {}".format(local_file))
                backup_results["failed"].append({
                    "file": local_file,
                    "error": "File not found"
                })
                continue
            
            # Read file content
            content = read(local_file)
            file_size = len(content)
            backup_results["total_size"] = backup_results["total_size"] + file_size
            
            # Create backup key with timestamp and organized path
            sanitized_path = local_file.replace("/", "_").replace("\\", "_")
            backup_key = "backups/{}/{}".format(timestamp, sanitized_path)
            
            print("Backing up: {} -> s3://{}/{} ({} bytes)".format(
                local_file, 
                bucket_name, 
                backup_key,
                file_size
            ))
            
            # Upload with comprehensive backup metadata
            s3.put_object(
                bucket_name,
                backup_key,
                content,
                metadata={
                    "backup-date": current_time.format("2006-01-02T15:04:05Z"),
                    "original-path": local_file,
                    "original-size": str(file_size),
                    "backup-type": "manual",
                    "backup-version": "1.0",
                    "checksum": "placeholder-md5",  # In real implementation, calculate MD5
                },
                tags={
                    "backup": "true",
                    "date": timestamp.split("-")[0] + "-" + timestamp.split("-")[1],  # YYYY-MM
                    "retention": "30days",
                    "priority": "high" if file_size > 1024*1024 else "normal",  # >1MB = high priority
                    "source": "starlark-backup"
                }
            )
            
            backup_results["success"].append({
                "file": local_file,
                "backup_key": backup_key,
                "size": file_size
            })
            
        except Exception as e:
            print("Failed to backup {}: {}".format(local_file, e))
            backup_results["failed"].append({
                "file": local_file,
                "error": str(e)
            })
    
    print("-" * 60)
    print("Backup completed!")
    print("Successfully backed up: {} files".format(len(backup_results["success"])))
    print("Failed backups: {} files".format(len(backup_results["failed"])))
    print("Total size: {:.2f} MB".format(backup_results["total_size"] / (1024*1024)))
    
    return backup_results

def list_backups(bucket_name, days=30):
    """List recent backups with details"""
    
    s3 = create_client()
    
    if not s3.bucket_exists(bucket_name):
        print("Backup bucket does not exist: {}".format(bucket_name))
        return []
    
    print("Recent backups (last {} days):".format(days))
    print("-" * 80)
    
    # List backup objects
    result = s3.list_objects(bucket_name, prefix="backups/", max_keys=1000)
    
    backups = []
    total_backup_size = 0
    
    for obj in result["contents"]:
        try:
            # Get object metadata to show backup info
            info = s3.get_object_info(bucket_name, obj["key"])
            metadata = info.get("metadata", {})
            
            backup_info = {
                "key": obj["key"],
                "size": obj["size"],
                "date": obj["last_modified"],
                "original_path": metadata.get("original-path", "unknown"),
                "backup_type": metadata.get("backup-type", "unknown")
            }
            
            backups.append(backup_info)
            total_backup_size = total_backup_size + obj["size"]
            
            print("  {} ({:.2f} MB)".format(
                obj["key"],
                obj["size"] / (1024*1024)
            ))
            print("    Original: {}".format(metadata.get("original-path", "unknown")))
            print("    Backup Date: {}".format(metadata.get("backup-date", "unknown")))
            print("    Type: {}".format(metadata.get("backup-type", "unknown")))
            print()
            
        except Exception as e:
            print("  {} - Error getting metadata: {}".format(obj["key"], e))
    
    print("-" * 80)
    print("Total backups: {}".format(len(backups)))
    print("Total backup size: {:.2f} MB".format(total_backup_size / (1024*1024)))
    
    return backups

def cleanup_old_backups(bucket_name, retention_days=30):
    """Clean up backups older than specified days"""
    
    s3 = create_client()
    
    if not s3.bucket_exists(bucket_name):
        print("Backup bucket does not exist: {}".format(bucket_name))
        return
    
    current_time = now()
    cutoff_time = current_time.add(-retention_days * 24 * 60 * 60 * 1000000000)  # nanoseconds
    
    print("Cleaning up backups older than {} days...".format(retention_days))
    print("Cutoff date: {}".format(cutoff_time.format("2006-01-02 15:04:05")))
    
    # List backup objects
    result = s3.list_objects(bucket_name, prefix="backups/")
    old_objects = []
    old_size = 0
    
    for obj in result["contents"]:
        # Check if object is older than cutoff
        # Note: In real implementation, you'd parse the timestamp properly
        obj_date = obj["last_modified"]
        if obj_date < cutoff_time:
            old_objects.append(obj["key"])
            old_size = old_size + obj["size"]
            print("  Marking for deletion: {} ({:.2f} MB)".format(
                obj["key"], 
                obj["size"] / (1024*1024)
            ))
    
    if len(old_objects) == 0:
        print("No old backups to delete")
        return
    
    print("\nDeleting {} old backup objects ({:.2f} MB)...".format(
        len(old_objects), 
        old_size / (1024*1024)
    ))
    
    # Delete in batches (S3 supports up to 1000 objects per batch)
    batch_size = 100
    deleted_count = 0
    
    for i in range(0, len(old_objects), batch_size):
        batch = old_objects[i:i + batch_size]
        
        try:
            delete_result = s3.delete_objects(bucket_name, batch)
            deleted_count = deleted_count + len(delete_result["deleted"])
            
            if "errors" in delete_result and len(delete_result["errors"]) > 0:
                print("Errors occurred during deletion:")
                for error in delete_result["errors"]:
                    print("  {}: {}".format(error["key"], error["message"]))
        
        except Exception as e:
            print("Failed to delete batch: {}".format(e))
    
    print("Cleanup completed. Deleted {} objects.".format(deleted_count))

def restore_backup(bucket_name, backup_key, restore_path):
    """Restore a backup to local filesystem"""
    
    s3 = create_client()
    
    if not s3.object_exists(bucket_name, backup_key):
        print("Backup not found: s3://{}/{}".format(bucket_name, backup_key))
        return False
    
    print("Restoring backup: s3://{}/{} -> {}".format(bucket_name, backup_key, restore_path))
    
    try:
        # Get backup metadata
        info = s3.get_object_info(bucket_name, backup_key)
        metadata = info.get("metadata", {})
        
        print("Backup information:")
        print("  Original path: {}".format(metadata.get("original-path", "unknown")))
        print("  Backup date: {}".format(metadata.get("backup-date", "unknown")))
        print("  Size: {:.2f} MB".format(info["size"] / (1024*1024)))
        
        # Download and save to restore path
        s3.get_object_to_file(bucket_name, backup_key, restore_path)
        
        print("Restore completed successfully!")
        return True
        
    except Exception as e:
        print("Restore failed: {}".format(e))
        return False

def main():
    """Main backup system demonstration"""
    
    bucket_name = "my-backup-bucket"
    
    print("S3 Backup System Example")
    print("=" * 50)
    
    # Example files to backup (in practice, these would be real files)
    files_to_backup = [
        "/important/config.json",
        "/data/database.sql", 
        "/logs/app.log",
        "/documents/report.pdf",
        "/scripts/backup.sh"
    ]
    
    print("Step 1: Performing backup...")
    backup_results = backup_files(bucket_name, files_to_backup)
    
    print("\nStep 2: Listing recent backups...")
    backups = list_backups(bucket_name, days=7)
    
    print("\nStep 3: Cleaning up old backups...")
    cleanup_old_backups(bucket_name, retention_days=7)  # Short retention for demo
    
    # Example restore operation
    if len(backup_results["success"]) > 0:
        first_backup = backup_results["success"][0]
        restore_path = "/tmp/restored_{}".format(first_backup["file"].split("/")[-1])
        
        print("\nStep 4: Demonstrating restore...")
        restore_backup(bucket_name, first_backup["backup_key"], restore_path)
    
    print("\nBackup system example completed!")

if __name__ == "__main__":
    main() 