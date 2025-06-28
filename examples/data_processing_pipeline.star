#!/usr/bin/env starcli
"""
Data Processing Pipeline Example

This example demonstrates a data processing pipeline using S3:
- Reading data from source bucket
- Processing and transforming data
- Writing results to destination bucket
- Pipeline status tracking
- Error handling and recovery

Usage: starcli data_processing_pipeline.star [source-bucket] [dest-bucket]
"""

load("s3", "create_client")
load("json", "encode", "decode")
load("time")

def main():
    """Main data processing pipeline"""
    
    # Get arguments
    args = runtime.args[1:]
    source_bucket = args[0] if len(args) > 0 else "raw-data-bucket"
    dest_bucket = args[1] if len(args) > 1 else "processed-data-bucket"
    
    print("=== S3 Data Processing Pipeline Demo ===")
    print("Source bucket: {}".format(source_bucket))
    print("Destination bucket: {}".format(dest_bucket))
    print()
    
    # Create S3 client
    s3 = create_client(region="us-east-1")
    
    # Step 1: Setup buckets
    setup_pipeline_buckets(s3, source_bucket, dest_bucket)
    
    # Step 2: Create sample data
    create_sample_data(s3, source_bucket)
    
    # Step 3: Process data
    process_data_files(s3, source_bucket, dest_bucket)
    
    # Step 4: Generate pipeline report
    generate_pipeline_report(s3, dest_bucket)
    
    print("\n=== Pipeline completed successfully! ===")

def setup_pipeline_buckets(s3, source_bucket, dest_bucket):
    """Setup source and destination buckets"""
    print("1. Setting up pipeline buckets...")
    
    # Setup source bucket
    if not s3.bucket_exists(source_bucket):
        print("  Creating source bucket '{}'...".format(source_bucket))
        s3.create_bucket(source_bucket)
        print("  ✓ Source bucket created")
    else:
        print("  ✓ Source bucket '{}' exists".format(source_bucket))
    
    # Setup destination bucket
    if not s3.bucket_exists(dest_bucket):
        print("  Creating destination bucket '{}'...".format(dest_bucket))
        s3.create_bucket(dest_bucket)
        print("  ✓ Destination bucket created")
    else:
        print("  ✓ Destination bucket '{}' exists".format(dest_bucket))
    
    print()

def create_sample_data(s3, source_bucket):
    """Create sample data files for processing"""
    print("2. Creating sample data files...")
    
    # Sample JSON data files
    sample_data = {
        "raw-data/users-2024-01-15.json": [
            {"id": 1, "name": "Alice", "age": 30, "country": "USA"},
            {"id": 2, "name": "Bob", "age": 25, "country": "Canada"},
            {"id": 3, "name": "Charlie", "age": 35, "country": "UK"}
        ],
        "raw-data/sales-2024-01-15.json": [
            {"product": "Widget A", "quantity": 100, "price": 9.99, "date": "2024-01-15"},
            {"product": "Widget B", "quantity": 50, "price": 19.99, "date": "2024-01-15"},
            {"product": "Widget C", "quantity": 25, "price": 29.99, "date": "2024-01-15"}
        ],
        "raw-data/events-2024-01-15.json": [
            {"event": "login", "user_id": 1, "timestamp": "2024-01-15T10:30:00Z"},
            {"event": "purchase", "user_id": 2, "timestamp": "2024-01-15T11:00:00Z"},
            {"event": "logout", "user_id": 1, "timestamp": "2024-01-15T12:00:00Z"}
        ]
    }
    
    for file_path, data in sample_data.items():
        json_content = encode(data)
        
        s3.put_object(
            source_bucket,
            file_path,
            json_content,
            content_type="application/json",
            metadata={
                "data-type": "raw",
                "created-at": time.now().format("2006-01-02T15:04:05Z"),
                "record-count": str(len(data))
            }
        )
        
        print("  ✓ Created: {} ({} records)".format(file_path, len(data)))
    
    print()

def process_data_files(s3, source_bucket, dest_bucket):
    """Process data files from source to destination"""
    print("3. Processing data files...")
    
    # List all files in the raw-data directory
    result = s3.list_objects(source_bucket, prefix="raw-data/")
    
    if len(result["contents"]) == 0:
        print("  No data files found to process")
        return
    
    processed_count = 0
    error_count = 0
    
    for obj in result["contents"]:
        file_key = obj["key"]
        
        if not file_key.endswith(".json"):
            print("  ⏭️  Skipping non-JSON file: {}".format(file_key))
            continue
        
        print("  📊 Processing: {}".format(file_key))
        
        try:
            # Download and parse the file
            raw_content = s3.get_object(source_bucket, file_key)
            raw_data = decode(raw_content)
            
            # Process the data based on file type
            processed_data = process_file_data(file_key, raw_data)
            
            # Generate processed file path
            processed_key = file_key.replace("raw-data/", "processed-data/").replace(".json", "-processed.json")
            
            # Upload processed data
            s3.put_object(
                dest_bucket,
                processed_key,
                encode(processed_data),
                content_type="application/json",
                metadata={
                    "data-type": "processed",
                    "source-file": file_key,
                    "source-bucket": source_bucket,
                    "processed-at": time.now().format("2006-01-02T15:04:05Z"),
                    "record-count": str(len(processed_data.get("records", [])))
                },
                tags={
                    "pipeline": "data-processing",
                    "status": "processed"
                }
            )
            
            processed_count = processed_count + 1
            print("     ✓ Processed to: {}".format(processed_key))
            
        except Exception as e:
            error_count = error_count + 1
            print("     ❌ Processing failed: {}".format(e))
            
            # Log error for investigation
            error_key = "errors/{}-error.txt".format(file_key.replace("/", "_"))
            error_message = "Error processing {}: {}".format(file_key, e)
            
            s3.put_object(
                dest_bucket,
                error_key,
                error_message,
                content_type="text/plain",
                metadata={
                    "error-type": "processing-error",
                    "source-file": file_key,
                    "error-time": time.now().format("2006-01-02T15:04:05Z")
                }
            )
    
    print("  📈 Processing Summary:")
    print("     ✅ Processed: {} files".format(processed_count))
    print("     ❌ Errors: {} files".format(error_count))
    print()

def process_file_data(file_key, raw_data):
    """Process data based on file type"""
    
    processed_data = {
        "metadata": {
            "source_file": file_key,
            "processed_at": time.now().format("2006-01-02T15:04:05Z"),
            "processor_version": "1.0"
        },
        "summary": {},
        "records": []
    }
    
    if "users" in file_key:
        # Process user data
        processed_data["summary"] = {
            "total_users": len(raw_data),
            "countries": list(set([user["country"] for user in raw_data])),
            "average_age": sum([user["age"] for user in raw_data]) / len(raw_data)
        }
        
        # Add processed records
        for user in raw_data:
            processed_user = {
                "id": user["id"],
                "name": user["name"],
                "age": user["age"],
                "country": user["country"],
                "age_group": get_age_group(user["age"]),
                "processed": True
            }
            processed_data["records"].append(processed_user)
    
    elif "sales" in file_key:
        # Process sales data
        total_revenue = sum([item["quantity"] * item["price"] for item in raw_data])
        processed_data["summary"] = {
            "total_items": len(raw_data),
            "total_revenue": total_revenue,
            "products": list(set([item["product"] for item in raw_data]))
        }
        
        # Add processed records with calculated fields
        for item in raw_data:
            processed_item = {
                "product": item["product"],
                "quantity": item["quantity"],
                "price": item["price"],
                "total_value": item["quantity"] * item["price"],
                "date": item["date"],
                "revenue_category": get_revenue_category(item["quantity"] * item["price"])
            }
            processed_data["records"].append(processed_item)
    
    elif "events" in file_key:
        # Process event data
        event_types = {}
        for event in raw_data:
            event_type = event["event"]
            if event_type not in event_types:
                event_types[event_type] = 0
            event_types[event_type] = event_types[event_type] + 1
        
        processed_data["summary"] = {
            "total_events": len(raw_data),
            "event_types": event_types,
            "unique_users": len(set([event["user_id"] for event in raw_data]))
        }
        
        # Add processed records
        for event in raw_data:
            processed_event = {
                "event": event["event"],
                "user_id": event["user_id"],
                "timestamp": event["timestamp"],
                "hour": event["timestamp"].split("T")[1].split(":")[0],
                "processed": True
            }
            processed_data["records"].append(processed_event)
    
    return processed_data

def get_age_group(age):
    """Categorize age into groups"""
    if age < 25:
        return "young"
    elif age < 35:
        return "adult"
    else:
        return "senior"

def get_revenue_category(revenue):
    """Categorize revenue amounts"""
    if revenue < 500:
        return "low"
    elif revenue < 1000:
        return "medium"
    else:
        return "high"

def generate_pipeline_report(s3, dest_bucket):
    """Generate a summary report of the pipeline execution"""
    print("4. Generating pipeline report...")
    
    # List processed files
    processed_result = s3.list_objects(dest_bucket, prefix="processed-data/")
    
    # List error files
    error_result = s3.list_objects(dest_bucket, prefix="errors/")
    
    # Generate report
    report = {
        "pipeline_execution": {
            "timestamp": time.now().format("2006-01-02T15:04:05Z"),
            "status": "completed",
            "destination_bucket": dest_bucket
        },
        "summary": {
            "processed_files": len(processed_result["contents"]),
            "error_files": len(error_result["contents"]),
            "total_files": len(processed_result["contents"]) + len(error_result["contents"])
        },
        "processed_files": [],
        "errors": []
    }
    
    # Add details about processed files
    for obj in processed_result["contents"]:
        info = s3.get_object_info(dest_bucket, obj["key"])
        metadata = info.get("metadata", {})
        
        file_info = {
            "file": obj["key"],
            "size": obj["size"],
            "source_file": metadata.get("source-file", "unknown"),
            "record_count": metadata.get("record-count", "0"),
            "processed_at": metadata.get("processed-at", "unknown")
        }
        report["processed_files"].append(file_info)
    
    # Add details about errors
    for obj in error_result["contents"]:
        error_info = {
            "error_file": obj["key"],
            "size": obj["size"]
        }
        report["errors"].append(error_info)
    
    # Save report
    report_key = "reports/pipeline-report-{}.json".format(time.now().format("2006-01-02-15-04-05"))
    
    s3.put_object(
        dest_bucket,
        report_key,
        encode(report),
        content_type="application/json",
        metadata={
            "report-type": "pipeline-summary",
            "generated-at": time.now().format("2006-01-02T15:04:05Z")
        }
    )
    
    print("  📄 Pipeline report saved: {}".format(report_key))
    print("  📊 Report Summary:")
    print("     ✅ Processed files: {}".format(report["summary"]["processed_files"]))
    print("     ❌ Error files: {}".format(report["summary"]["error_files"]))
    print("     📁 Total files: {}".format(report["summary"]["total_files"]))
    
    # Display sample processed data
    if len(report["processed_files"]) > 0:
        print("  🔍 Sample processed files:")
        for file_info in report["processed_files"][:3]:  # Show first 3
            print("     {} ({} records)".format(file_info["file"], file_info["record_count"]))

# Run the pipeline
main() 