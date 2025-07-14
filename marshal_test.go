package s3

import (
	"testing"
	"time"

	startime "go.starlark.net/lib/time"
	"go.starlark.net/starlark"
)

func TestBucketInfoMarshalStarlark(t *testing.T) {
	// Create a test BucketInfo
	bucketInfo := &BucketInfo{
		Name:                "test-bucket",
		CreationDate:        time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Region:              "us-east-1",
		VersioningStatus:    "Enabled",
		PublicAccessBlocked: true,
		HasPolicy:           false,
		EncryptionEnabled:   true,
		ObjectCount:         42,
		TotalSize:           1024000,
	}

	// Marshal to Starlark
	value, err := bucketInfo.MarshalStarlark()
	if err != nil {
		t.Fatalf("Failed to marshal BucketInfo: %v", err)
	}

	// Verify it's a dictionary
	dict, ok := value.(*starlark.Dict)
	if !ok {
		t.Fatalf("Expected starlark.Dict, got %T", value)
	}

	// Check key values
	tests := []struct {
		key      string
		expected interface{}
	}{
		{"name", "test-bucket"},
		{"region", "us-east-1"},
		{"versioning_status", "Enabled"},
		{"public_access_blocked", true},
		{"has_policy", false},
		{"encryption_enabled", true},
		{"object_count", int64(42)},
		{"total_size", int64(1024000)},
		{"creation_date", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	for _, test := range tests {
		val, found, err := dict.Get(starlark.String(test.key))
		if err != nil {
			t.Errorf("Error getting key %s: %v", test.key, err)
			continue
		}
		if !found {
			t.Errorf("Key %s not found in dictionary", test.key)
			continue
		}

		switch expected := test.expected.(type) {
		case string:
			if str, ok := val.(starlark.String); ok {
				if str.GoString() != expected {
					t.Errorf("Key %s: expected %s, got %s", test.key, expected, str.GoString())
				}
			} else {
				t.Errorf("Key %s: expected string, got %T", test.key, val)
			}
		case bool:
			if b, ok := val.(starlark.Bool); ok {
				if bool(b) != expected {
					t.Errorf("Key %s: expected %t, got %t", test.key, expected, bool(b))
				}
			} else {
				t.Errorf("Key %s: expected bool, got %T", test.key, val)
			}
		case int64:
			if i, ok := val.(starlark.Int); ok {
				if val, _ := i.Int64(); val != expected {
					t.Errorf("Key %s: expected %d, got %d", test.key, expected, val)
				}
			} else {
				t.Errorf("Key %s: expected int, got %T", test.key, val)
			}
		case time.Time:
			if timeVal, ok := val.(startime.Time); ok {
				// Convert startime.Time to time.Time for comparison
				actualTime := time.Time(timeVal)
				if actualTime != expected {
					t.Errorf("Key %s: expected %v, got %v", test.key, expected, actualTime)
				}
			} else {
				t.Errorf("Key %s: expected time.Time, got %T", test.key, val)
			}
		}
	}
}

func TestObjectInfoMarshalStarlark(t *testing.T) {
	// Create a test ObjectInfo
	objectInfo := &ObjectInfo{
		Key:          "test-key.txt",
		Size:         1024,
		LastModified: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		ETag:         "\"abc123\"",
		ContentType:  "text/plain",
		Metadata: map[string]string{
			"author": "test",
			"type":   "document",
		},
	}

	// Marshal to Starlark
	value, err := objectInfo.MarshalStarlark()
	if err != nil {
		t.Fatalf("Failed to marshal ObjectInfo: %v", err)
	}

	// Verify it's a dictionary
	dict, ok := value.(*starlark.Dict)
	if !ok {
		t.Fatalf("Expected starlark.Dict, got %T", value)
	}

	// Check key values
	tests := []struct {
		key      string
		expected interface{}
	}{
		{"key", "test-key.txt"},
		{"size", int64(1024)},
		{"etag", "\"abc123\""},
		{"content_type", "text/plain"},
		{"last_modified", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
	}

	for _, test := range tests {
		val, found, err := dict.Get(starlark.String(test.key))
		if err != nil {
			t.Errorf("Error getting key %s: %v", test.key, err)
			continue
		}
		if !found {
			t.Errorf("Key %s not found in dictionary", test.key)
			continue
		}

		switch expected := test.expected.(type) {
		case string:
			if str, ok := val.(starlark.String); ok {
				if str.GoString() != expected {
					t.Errorf("Key %s: expected %s, got %s", test.key, expected, str.GoString())
				}
			} else {
				t.Errorf("Key %s: expected string, got %T", test.key, val)
			}
		case int64:
			if i, ok := val.(starlark.Int); ok {
				if val, _ := i.Int64(); val != expected {
					t.Errorf("Key %s: expected %d, got %d", test.key, expected, val)
				}
			} else {
				t.Errorf("Key %s: expected int, got %T", test.key, val)
			}
		case time.Time:
			if timeVal, ok := val.(startime.Time); ok {
				// Convert startime.Time to time.Time for comparison
				actualTime := time.Time(timeVal)
				if actualTime != expected {
					t.Errorf("Key %s: expected %v, got %v", test.key, expected, actualTime)
				}
			} else {
				t.Errorf("Key %s: expected time.Time, got %T", test.key, val)
			}
		}
	}

	// Check metadata
	metadataVal, found, err := dict.Get(starlark.String("metadata"))
	if err != nil {
		t.Errorf("Error getting metadata: %v", err)
	}
	if !found {
		t.Errorf("Metadata not found in dictionary")
	}
	if metadataDict, ok := metadataVal.(*starlark.Dict); ok {
		if metadataDict.Len() != 2 {
			t.Errorf("Expected metadata to have 2 items, got %d", metadataDict.Len())
		}
	} else {
		t.Errorf("Expected metadata to be a dict, got %T", metadataVal)
	}
}

func TestListObjectsResultMarshalStarlark(t *testing.T) {
	// Create test objects
	objects := []ObjectInfo{
		{
			Key:          "file1.txt",
			Size:         100,
			LastModified: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			ETag:         "\"etag1\"",
			ContentType:  "text/plain",
		},
		{
			Key:          "file2.txt",
			Size:         200,
			LastModified: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			ETag:         "\"etag2\"",
			ContentType:  "text/plain",
		},
	}

	// Create a test ListObjectsResult
	result := &ListObjectsResult{
		Contents:       objects,
		CommonPrefixes: []string{"prefix1/", "prefix2/"},
		IsTruncated:    false,
		NextMarker:     "",
		MaxKeys:        1000,
		Prefix:         "test-",
		Delimiter:      "/",
	}

	// Marshal to Starlark
	value, err := result.MarshalStarlark()
	if err != nil {
		t.Fatalf("Failed to marshal ListObjectsResult: %v", err)
	}

	// Verify it's a dictionary
	dict, ok := value.(*starlark.Dict)
	if !ok {
		t.Fatalf("Expected starlark.Dict, got %T", value)
	}

	// Check contents
	contentsVal, found, err := dict.Get(starlark.String("contents"))
	if err != nil {
		t.Errorf("Error getting contents: %v", err)
	}
	if !found {
		t.Errorf("Contents not found in dictionary")
	}
	if contentsList, ok := contentsVal.(*starlark.List); ok {
		if contentsList.Len() != 2 {
			t.Errorf("Expected contents to have 2 items, got %d", contentsList.Len())
		}
	} else {
		t.Errorf("Expected contents to be a list, got %T", contentsVal)
	}

	// Check common prefixes
	prefixesVal, found, err := dict.Get(starlark.String("common_prefixes"))
	if err != nil {
		t.Errorf("Error getting common_prefixes: %v", err)
	}
	if !found {
		t.Errorf("Common prefixes not found in dictionary")
	}
	if prefixesList, ok := prefixesVal.(*starlark.List); ok {
		if prefixesList.Len() != 2 {
			t.Errorf("Expected common_prefixes to have 2 items, got %d", prefixesList.Len())
		}
	} else {
		t.Errorf("Expected common_prefixes to be a list, got %T", prefixesVal)
	}

	// Check other fields
	tests := []struct {
		key      string
		expected interface{}
	}{
		{"is_truncated", false},
		{"next_marker", ""},
		{"max_keys", 1000},
		{"prefix", "test-"},
		{"delimiter", "/"},
	}

	for _, test := range tests {
		val, found, err := dict.Get(starlark.String(test.key))
		if err != nil {
			t.Errorf("Error getting key %s: %v", test.key, err)
			continue
		}
		if !found {
			t.Errorf("Key %s not found in dictionary", test.key)
			continue
		}

		switch expected := test.expected.(type) {
		case string:
			if str, ok := val.(starlark.String); ok {
				if str.GoString() != expected {
					t.Errorf("Key %s: expected %s, got %s", test.key, expected, str.GoString())
				}
			} else {
				t.Errorf("Key %s: expected string, got %T", test.key, val)
			}
		case bool:
			if b, ok := val.(starlark.Bool); ok {
				if bool(b) != expected {
					t.Errorf("Key %s: expected %t, got %t", test.key, expected, bool(b))
				}
			} else {
				t.Errorf("Key %s: expected bool, got %T", test.key, val)
			}
		case int:
			if i, ok := val.(starlark.Int); ok {
				if val, _ := i.Int64(); val != int64(expected) {
					t.Errorf("Key %s: expected %d, got %d", test.key, expected, val)
				}
			} else {
				t.Errorf("Key %s: expected int, got %T", test.key, val)
			}
		}
	}
}
