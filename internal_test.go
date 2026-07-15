package s3

// Go-level unit tests for the module's pure logic — the parts that run with no
// TTY and no network: input validation, the Starlark<->Go conversion helpers,
// the option carriers and their AWS-input mappers, provider/URL construction,
// and the result marshalers. These complement the script-driven tests in
// example_test.go and the detection tests in detection_test.go.
//
// Sections:
//   - bucket-name / object-key validation (rules + edge cases)
//   - parseObjectOptions (the single option-parsing seam) + time parsing
//   - convertStarlarkDict / metadata conversion (non-string key/value errors)
//   - ObjectOptions / ListObjectsOptions Validate + ApplyTo* AWS-input mappers
//   - AWS<->our-struct converters (owner display name, object/bucket info)
//   - small Starlark conversion helpers (time, string map, string slice, tags)
//   - provider registry + GenerateURLWithProvider + URL parse helpers
//   - BucketInfo / ObjectInfo / ListObjectsResult MarshalStarlark
//   - ClientWrapper Starlark-value protocol (Type/Truth/Hash/Freeze/Attr/String)

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithymiddleware "github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"go.starlark.net/starlark"
)

// --- bucket-name / object-key validation ------------------------------------

func TestValidateBucketName(t *testing.T) {
	cases := []struct {
		name    string
		bucket  string
		wantErr string // substring expected in the error, "" => valid
	}{
		{"valid simple", "my-bucket", ""},
		{"valid with dots", "my.bucket.example", ""},
		{"valid min length", "abc", ""},
		{"valid digits", "bucket123", ""},
		{"empty", "", "cannot be empty"},
		{"too short", "ab", "between 3 and 63"},
		{"too long", strings.Repeat("a", 64), "between 3 and 63"},
		{"uppercase", "My-Bucket", "must start and end"},
		{"underscore", "bucket_name", "must start and end"},
		{"leading hyphen", "-bucket", "must start and end"},
		{"trailing dot", "bucket.", "must start and end"},
		{"consecutive dots", "my..bucket", "consecutive dots"},
		{"ip address", "192.168.1.1", "IP address"},
		{"xn-- prefix", "xn--bucket", "cannot start with 'xn--'"},
		{"-s3alias suffix", "bucket-s3alias", "cannot end with '-s3alias'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateBucketName(c.bucket)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("validateBucketName(%q) = %v, want nil", c.bucket, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateBucketName(%q) = nil, want error containing %q", c.bucket, c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("validateBucketName(%q) = %q, want substring %q", c.bucket, err.Error(), c.wantErr)
			}
		})
	}
}

func TestValidateObjectKey(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr string
	}{
		{"valid simple", "file.txt", ""},
		{"valid nested", "path/to/file.txt", ""},
		{"valid spaces", "file with spaces.txt", ""},
		{"valid max length", strings.Repeat("a", 1024), ""},
		{"empty", "", "cannot be empty"},
		{"too long", strings.Repeat("a", 1025), "exceed 1024"},
		{"null byte", "file\x00.txt", "control characters"},
		{"tab", "file\x09.txt", "control characters"},
		{"newline", "file\x0A.txt", "control characters"},
		{"carriage return", "file\x0D.txt", "control characters"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateObjectKey(c.key)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("validateObjectKey(%q) = %v, want nil", c.key, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("validateObjectKey(%q) = %v, want substring %q", c.key, err, c.wantErr)
			}
		})
	}
}

// --- parseObjectOptions + time parsing --------------------------------------

func TestParseObjectOptions(t *testing.T) {
	emptyDict := starlark.NewDict(0)

	t.Run("all string fields populate pointers", func(t *testing.T) {
		opt, err := parseObjectOptions("text/plain", "max-age=60", "inline", "gzip", "en", "", emptyDict, emptyDict)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opt.ContentType == nil || *opt.ContentType != "text/plain" {
			t.Errorf("ContentType = %v, want text/plain", opt.ContentType)
		}
		if opt.CacheControl == nil || *opt.CacheControl != "max-age=60" {
			t.Errorf("CacheControl = %v", opt.CacheControl)
		}
		if opt.ContentDisposition == nil || *opt.ContentDisposition != "inline" {
			t.Errorf("ContentDisposition = %v", opt.ContentDisposition)
		}
		if opt.ContentEncoding == nil || *opt.ContentEncoding != "gzip" {
			t.Errorf("ContentEncoding = %v", opt.ContentEncoding)
		}
		if opt.ContentLanguage == nil || *opt.ContentLanguage != "en" {
			t.Errorf("ContentLanguage = %v", opt.ContentLanguage)
		}
		// Empty fields stay nil; with no fields set Validate() must be false.
		if opt.Expires != nil || opt.Metadata != nil || opt.Tags != nil {
			t.Errorf("empty fields should stay nil")
		}
	})

	t.Run("empty strings leave everything nil", func(t *testing.T) {
		opt, err := parseObjectOptions("", "", "", "", "", "", emptyDict, emptyDict)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opt.Validate() {
			t.Errorf("all-empty options should report Validate()=false")
		}
	})

	t.Run("valid RFC3339 expires", func(t *testing.T) {
		opt, err := parseObjectOptions("", "", "", "", "", "2025-01-02T15:04:05Z", emptyDict, emptyDict)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opt.Expires == nil {
			t.Fatal("Expires should be set")
		}
		want := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
		if !opt.Expires.Equal(want) {
			t.Errorf("Expires = %v, want %v", opt.Expires, want)
		}
	})

	t.Run("invalid expires returns error", func(t *testing.T) {
		_, err := parseObjectOptions("", "", "", "", "", "not-a-time", emptyDict, emptyDict)
		if err == nil {
			t.Fatal("expected error for invalid expires")
		}
		if !strings.Contains(err.Error(), "failed to convert expires time") {
			t.Errorf("error = %q, want 'failed to convert expires time'", err.Error())
		}
	})

	t.Run("metadata and tags dicts populate maps", func(t *testing.T) {
		md := starlark.NewDict(1)
		md.SetKey(starlark.String("author"), starlark.String("ada"))
		tg := starlark.NewDict(1)
		tg.SetKey(starlark.String("env"), starlark.String("prod"))
		opt, err := parseObjectOptions("", "", "", "", "", "", md, tg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opt.Metadata == nil || (*opt.Metadata)["author"] != "ada" {
			t.Errorf("Metadata = %v", opt.Metadata)
		}
		if opt.Tags == nil || (*opt.Tags)["env"] != "prod" {
			t.Errorf("Tags = %v", opt.Tags)
		}
	})

	t.Run("non-string metadata value is a clean error", func(t *testing.T) {
		md := starlark.NewDict(1)
		md.SetKey(starlark.String("n"), starlark.MakeInt(5))
		_, err := parseObjectOptions("", "", "", "", "", "", md, starlark.NewDict(0))
		if err == nil || !strings.Contains(err.Error(), "value must be a string") {
			t.Fatalf("err = %v, want 'value must be a string'", err)
		}
	})

	t.Run("non-string metadata key is a clean error", func(t *testing.T) {
		md := starlark.NewDict(1)
		md.SetKey(starlark.MakeInt(5), starlark.String("v"))
		_, err := parseObjectOptions("", "", "", "", "", "", md, starlark.NewDict(0))
		if err == nil || !strings.Contains(err.Error(), "key must be a string") {
			t.Fatalf("err = %v, want 'key must be a string'", err)
		}
	})
}

func TestConvertStarlarkStringToTime(t *testing.T) {
	t.Run("empty returns zero time, no error", func(t *testing.T) {
		got, err := convertStarlarkStringToTime("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.IsZero() {
			t.Errorf("want zero time, got %v", got)
		}
	})
	t.Run("valid RFC3339", func(t *testing.T) {
		got, err := convertStarlarkStringToTime("2020-06-15T00:00:00Z")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Year() != 2020 || got.Month() != 6 {
			t.Errorf("parsed wrong: %v", got)
		}
	})
	t.Run("invalid format errors", func(t *testing.T) {
		_, err := convertStarlarkStringToTime("15/06/2020")
		if err == nil || !strings.Contains(err.Error(), "RFC3339") {
			t.Fatalf("err = %v, want RFC3339 mention", err)
		}
	})
}

// --- convertStarlarkDict ----------------------------------------------------

func TestConvertStarlarkDict(t *testing.T) {
	t.Run("nil dict", func(t *testing.T) {
		got, err := convertStarlarkDict(nil)
		if err != nil || got != nil {
			t.Fatalf("got %v, %v; want nil, nil", got, err)
		}
	})
	t.Run("empty dict", func(t *testing.T) {
		got, err := convertStarlarkDict(starlark.NewDict(0))
		if err != nil || got != nil {
			t.Fatalf("got %v, %v; want nil, nil", got, err)
		}
	})
	t.Run("string keys and values", func(t *testing.T) {
		d := starlark.NewDict(2)
		d.SetKey(starlark.String("a"), starlark.String("1"))
		d.SetKey(starlark.String("b"), starlark.String("2"))
		got, err := convertStarlarkDict(d)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["a"] != "1" || got["b"] != "2" {
			t.Errorf("got %v", got)
		}
	})
	t.Run("non-string key errors with type", func(t *testing.T) {
		d := starlark.NewDict(1)
		d.SetKey(starlark.MakeInt(1), starlark.String("v"))
		_, err := convertStarlarkDict(d)
		if err == nil || !strings.Contains(err.Error(), "key must be a string") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("non-string value errors with type", func(t *testing.T) {
		d := starlark.NewDict(1)
		d.SetKey(starlark.String("k"), starlark.None)
		_, err := convertStarlarkDict(d)
		if err == nil || !strings.Contains(err.Error(), "value must be a string") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("convertMetadataDict delegates", func(t *testing.T) {
		d := starlark.NewDict(1)
		d.SetKey(starlark.String("x"), starlark.String("y"))
		got, err := convertMetadataDict(d)
		if err != nil || got["x"] != "y" {
			t.Fatalf("got %v, %v", got, err)
		}
	})
}

// --- option carriers: Validate + ApplyTo* -----------------------------------

func TestObjectOptionsValidate(t *testing.T) {
	if NewObjectOptions().Validate() {
		t.Error("fresh ObjectOptions should report Validate()=false")
	}
	ct := "text/plain"
	if !(&ObjectOptions{ContentType: &ct}).Validate() {
		t.Error("ObjectOptions with ContentType should be valid")
	}
	now := time.Now()
	if !(&ObjectOptions{Expires: &now}).Validate() {
		t.Error("ObjectOptions with Expires should be valid")
	}
}

func TestObjectOptionsApplyToPutObject(t *testing.T) {
	ct, cc, cd, ce, cl := "text/plain", "max-age=60", "inline", "gzip", "en"
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	md := map[string]string{"a": "1"}
	tg := map[string]string{"k": "v"}
	o := &ObjectOptions{
		ContentType: &ct, CacheControl: &cc, ContentDisposition: &cd,
		ContentEncoding: &ce, ContentLanguage: &cl, Expires: &exp,
		Metadata: &md, Tags: &tg,
	}
	in := &awss3.PutObjectInput{}
	o.ApplyToPutObject(in)

	if aws.ToString(in.ContentType) != ct {
		t.Errorf("ContentType = %q", aws.ToString(in.ContentType))
	}
	if aws.ToString(in.CacheControl) != cc {
		t.Errorf("CacheControl = %q", aws.ToString(in.CacheControl))
	}
	if aws.ToString(in.ContentDisposition) != cd {
		t.Errorf("ContentDisposition = %q", aws.ToString(in.ContentDisposition))
	}
	if aws.ToString(in.ContentEncoding) != ce {
		t.Errorf("ContentEncoding = %q", aws.ToString(in.ContentEncoding))
	}
	if aws.ToString(in.ContentLanguage) != cl {
		t.Errorf("ContentLanguage = %q", aws.ToString(in.ContentLanguage))
	}
	if in.Expires == nil || !in.Expires.Equal(exp) {
		t.Errorf("Expires = %v", in.Expires)
	}
	if in.Metadata["a"] != "1" {
		t.Errorf("Metadata = %v", in.Metadata)
	}
	if aws.ToString(in.Tagging) != "k=v" {
		t.Errorf("Tagging = %q, want k=v", aws.ToString(in.Tagging))
	}
}

func TestObjectOptionsApplyToCopyObject(t *testing.T) {
	t.Run("non-empty sets metadata+tagging directives to REPLACE", func(t *testing.T) {
		ct := "application/json"
		tg := map[string]string{"k": "v"}
		o := &ObjectOptions{ContentType: &ct, Tags: &tg}
		in := &awss3.CopyObjectInput{}
		o.ApplyToCopyObject(in)
		if in.MetadataDirective != awstypes.MetadataDirectiveReplace {
			t.Errorf("MetadataDirective = %q, want REPLACE", in.MetadataDirective)
		}
		if in.TaggingDirective != awstypes.TaggingDirectiveReplace {
			t.Errorf("TaggingDirective = %q, want REPLACE", in.TaggingDirective)
		}
		if aws.ToString(in.Tagging) != "k=v" {
			t.Errorf("Tagging = %q", aws.ToString(in.Tagging))
		}
	})
	t.Run("empty options set no directives", func(t *testing.T) {
		in := &awss3.CopyObjectInput{}
		(&ObjectOptions{}).ApplyToCopyObject(in)
		if in.MetadataDirective != "" {
			t.Errorf("MetadataDirective = %q, want empty", in.MetadataDirective)
		}
		if in.TaggingDirective != "" {
			t.Errorf("TaggingDirective = %q, want empty", in.TaggingDirective)
		}
	})
	t.Run("metadata-only sets metadata directive but not tagging", func(t *testing.T) {
		md := map[string]string{"a": "1"}
		in := &awss3.CopyObjectInput{}
		(&ObjectOptions{Metadata: &md}).ApplyToCopyObject(in)
		if in.MetadataDirective != awstypes.MetadataDirectiveReplace {
			t.Errorf("MetadataDirective = %q, want REPLACE", in.MetadataDirective)
		}
		if in.TaggingDirective != "" {
			t.Errorf("TaggingDirective = %q, want empty (no tags)", in.TaggingDirective)
		}
	})
}

func TestListObjectsOptions(t *testing.T) {
	if NewListObjectsOptions().Validate() {
		t.Error("fresh ListObjectsOptions should report Validate()=false")
	}
	prefix, delim := "docs/", "/"
	maxKeys := 50
	token := "tok"
	o := &ListObjectsOptions{Prefix: &prefix, Delimiter: &delim, MaxKeys: &maxKeys, ContinuationToken: &token}
	if !o.Validate() {
		t.Error("populated ListObjectsOptions should be valid")
	}
	in := &awss3.ListObjectsV2Input{}
	o.ApplyToListObjects(in)
	if aws.ToString(in.Prefix) != prefix {
		t.Errorf("Prefix = %q", aws.ToString(in.Prefix))
	}
	if aws.ToString(in.Delimiter) != delim {
		t.Errorf("Delimiter = %q", aws.ToString(in.Delimiter))
	}
	if in.MaxKeys == nil || *in.MaxKeys != 50 {
		t.Errorf("MaxKeys = %v, want 50", in.MaxKeys)
	}
	if aws.ToString(in.ContinuationToken) != token {
		t.Errorf("ContinuationToken = %q", aws.ToString(in.ContinuationToken))
	}
}

// --- AWS <-> our-struct converters ------------------------------------------

func TestGetOwnerDisplayName(t *testing.T) {
	cases := []struct {
		name  string
		owner *awstypes.Owner
		want  string
	}{
		{"nil", nil, ""},
		{"display name", &awstypes.Owner{DisplayName: aws.String("Ada")}, "Ada"},
		{"id only", &awstypes.Owner{ID: aws.String("id-123")}, "id-123"},
		{"empty owner", &awstypes.Owner{}, ""},
		{"prefers display over id", &awstypes.Owner{DisplayName: aws.String("Ada"), ID: aws.String("id-123")}, "Ada"},
	}
	for _, c := range cases {
		if got := getOwnerDisplayName(c.owner); got != c.want {
			t.Errorf("%s: getOwnerDisplayName = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestConvertAWSBucketToBucketInfo(t *testing.T) {
	when := time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC)
	got := convertAWSBucketToBucketInfo(awstypes.Bucket{Name: aws.String("b"), CreationDate: aws.Time(when)})
	if got.Name != "b" {
		t.Errorf("Name = %q", got.Name)
	}
	if !got.CreationDate.Equal(when) {
		t.Errorf("CreationDate = %v", got.CreationDate)
	}
	// nil fields must not panic and yield zero values.
	zero := convertAWSBucketToBucketInfo(awstypes.Bucket{})
	if zero.Name != "" || !zero.CreationDate.IsZero() {
		t.Errorf("zero bucket = %+v", zero)
	}
}

func TestConvertAWSObjectToObjectInfo(t *testing.T) {
	obj := awstypes.Object{
		Key:               aws.String("k"),
		Size:              aws.Int64(42),
		ETag:              aws.String("etag"),
		StorageClass:      awstypes.ObjectStorageClassStandard,
		ChecksumAlgorithm: []awstypes.ChecksumAlgorithm{awstypes.ChecksumAlgorithmSha256},
	}
	got := convertAWSObjectToObjectInfo(obj)
	if got.Key != "k" || got.Size != 42 || got.ETag != "etag" {
		t.Errorf("got %+v", got)
	}
	if got.StorageClass != "STANDARD" {
		t.Errorf("StorageClass = %q", got.StorageClass)
	}
	// The checksum algorithm is reported in its own field, not smuggled into
	// version_id. A ListObjectsV2 entry has no VersionId, so it stays empty.
	if got.ChecksumAlgorithm != "SHA256" {
		t.Errorf("ChecksumAlgorithm = %q, want SHA256", got.ChecksumAlgorithm)
	}
	if got.VersionID != "" {
		t.Errorf("VersionID = %q, want empty (ListObjectsV2 carries no version)", got.VersionID)
	}
	// No checksum -> empty ChecksumAlgorithm/VersionID, no panic on nil pointers.
	plain := convertAWSObjectToObjectInfo(awstypes.Object{})
	if plain.ChecksumAlgorithm != "" || plain.VersionID != "" || plain.Key != "" {
		t.Errorf("plain = %+v", plain)
	}
	// The marshalled dict exposes both fields distinctly.
	v, err := got.MarshalStarlark()
	if err != nil {
		t.Fatalf("MarshalStarlark: %v", err)
	}
	d := v.(*starlark.Dict)
	if ca, _, _ := d.Get(starlark.String("checksum_algorithm")); ca.(starlark.String).GoString() != "SHA256" {
		t.Errorf("marshalled checksum_algorithm = %v", ca)
	}
	if vid, _, _ := d.Get(starlark.String("version_id")); vid.(starlark.String).GoString() != "" {
		t.Errorf("marshalled version_id = %v, want empty", vid)
	}
}

func TestClientBehaviorOptionsWiring(t *testing.T) {
	// Keep the retry assertions hermetic: an ambient AWS_MAX_ATTEMPTS would feed
	// a non-zero value into aws.Config wherever we intentionally leave it unset.
	t.Setenv("AWS_MAX_ATTEMPTS", "")
	// The timeout / max-retries / logging / user-agent settings must actually
	// reach the SDK config instead of being silently dropped.
	cc := &ClientConfig{
		Region:        "us-east-1",
		Timeout:       7,
		MaxRetries:    5,
		EnableLogging: true,
		UserAgent:     "Starlark-S3/test",
	}
	cfg, err := createAWSConfig(context.Background(), cc)
	if err != nil {
		t.Fatalf("createAWSConfig: %v", err)
	}
	if cfg.RetryMaxAttempts != 5 {
		t.Errorf("RetryMaxAttempts = %d, want 5", cfg.RetryMaxAttempts)
	}
	if cfg.HTTPClient == nil {
		t.Error("HTTPClient is nil — timeout was not applied")
	}
	if cfg.ClientLogMode&aws.LogRequest == 0 {
		t.Errorf("ClientLogMode = %v, want LogRequest set", cfg.ClientLogMode)
	}
	if len(cfg.APIOptions) == 0 {
		t.Error("APIOptions empty — user agent was not applied")
	}

	// With logging off and no custom user agent, those levers stay at the SDK
	// default (no log mode, no extra API option) — an unset value must not
	// change behavior.
	base := &ClientConfig{Region: "us-east-1", Timeout: 30, MaxRetries: 3}
	cfg2, err := createAWSConfig(context.Background(), base)
	if err != nil {
		t.Fatalf("createAWSConfig(base): %v", err)
	}
	if cfg2.ClientLogMode != 0 {
		t.Errorf("ClientLogMode = %v, want 0 when logging disabled", cfg2.ClientLogMode)
	}
	if len(cfg2.APIOptions) != 0 {
		t.Errorf("APIOptions = %d, want 0 when no custom user agent", len(cfg2.APIOptions))
	}
	if cfg2.RetryMaxAttempts != 3 {
		t.Errorf("RetryMaxAttempts = %d, want 3 (default preserved)", cfg2.RetryMaxAttempts)
	}
	// max_retries=0 is skipped so the SDK/env default applies (not forced to 0).
	cfg3, err := createAWSConfig(context.Background(), &ClientConfig{Region: "us-east-1", Timeout: 30, MaxRetries: 0})
	if err != nil {
		t.Fatalf("createAWSConfig(zero-retries): %v", err)
	}
	if cfg3.RetryMaxAttempts != 0 { // 0 == "unset" on aws.Config; SDK falls back to its default
		t.Errorf("RetryMaxAttempts = %d, want 0 (unset -> SDK default) when max_retries=0", cfg3.RetryMaxAttempts)
	}
}

func TestTimeoutDuration(t *testing.T) {
	if got := timeoutDuration(7); got != 7*time.Second {
		t.Errorf("timeoutDuration(7) = %v, want 7s", got)
	}
	// A value far above the cap must clamp to it and stay POSITIVE (a negative
	// duration would disable the bound). 1<<30 seconds is well past the cap yet
	// still fits a 32-bit int, so the test compiles on every target.
	big := timeoutDuration(1 << 30)
	if big <= 0 {
		t.Errorf("timeoutDuration(big) = %v, must stay positive", big)
	}
	if big != maxRequestTimeoutSeconds*time.Second {
		t.Errorf("timeoutDuration(big) = %v, want cap %v", big, maxRequestTimeoutSeconds*time.Second)
	}
}

func TestSplitUserAgent(t *testing.T) {
	// The routing decision this module owns: a "name/version" form must be
	// recognized as a pair (so it goes through AddUserAgentKeyValue and keeps its
	// '/'), while a bare string is a single key. Asserting the decision — not
	// just that the middleware installs — means reverting to the old
	// AddUserAgentKey(ua) path would fail this test.
	name, version, pair := splitUserAgent("Starlark-S3/1.0")
	if !pair || name != "Starlark-S3" || version != "1.0" {
		t.Errorf("splitUserAgent(slash) = (%q, %q, %v), want (Starlark-S3, 1.0, true)", name, version, pair)
	}
	if _, _, pair := splitUserAgent("PlainAgent"); pair {
		t.Error("splitUserAgent(bare) reported a pair, want single key")
	}
	// Both forms must also install onto a real middleware stack without error.
	for _, ua := range []string{"Starlark-S3/1.0", "PlainAgent"} {
		stack := smithymiddleware.NewStack("test", smithyhttp.NewStackRequest)
		if err := userAgentOption(ua)(stack); err != nil {
			t.Errorf("userAgentOption(%q) apply error: %v", ua, err)
		}
	}
}

func TestDeleteObjectsPartialError(t *testing.T) {
	// No per-object failures -> no error (the batch fully succeeded).
	if err := deleteObjectsPartialError(nil); err != nil {
		t.Errorf("empty errors should be nil, got %v", err)
	}
	// Any per-object failure must surface as an error naming the first one and
	// the total count, so a force delete does not falsely report success.
	errs := []awstypes.Error{
		{Key: aws.String("locked.txt"), Code: aws.String("AccessDenied"), Message: aws.String("object is locked")},
		{Key: aws.String("other.txt"), Code: aws.String("InternalError"), Message: aws.String("try again")},
	}
	err := deleteObjectsPartialError(errs)
	if err == nil {
		t.Fatal("per-object failures must produce an error")
	}
	for _, want := range []string{"2 object", "locked.txt", "AccessDenied", "object is locked"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
}

// --- small Starlark conversion helpers --------------------------------------

func TestTimeToStarlark(t *testing.T) {
	if got := timeToStarlark(time.Time{}); got != starlark.None {
		t.Errorf("zero time should map to None, got %v", got)
	}
	if got := timeToStarlark(time.Unix(1000, 0)); got == starlark.None {
		t.Error("non-zero time should not be None")
	}
}

func TestStringMapToStarlark(t *testing.T) {
	d := stringMapToStarlark(map[string]string{"a": "1"})
	v, found, err := d.Get(starlark.String("a"))
	if err != nil || !found {
		t.Fatalf("Get returned found=%v err=%v", found, err)
	}
	if v != starlark.String("1") {
		t.Errorf("value = %v", v)
	}
	// nil map -> empty dict, no panic.
	if stringMapToStarlark(nil).Len() != 0 {
		t.Error("nil map should produce empty dict")
	}
}

func TestStringSliceToStarlark(t *testing.T) {
	l := stringSliceToStarlark([]string{"x", "y"})
	if l.Len() != 2 || l.Index(0) != starlark.String("x") || l.Index(1) != starlark.String("y") {
		t.Errorf("slice = %v", l)
	}
	if stringSliceToStarlark(nil).Len() != 0 {
		t.Error("nil slice should produce empty list")
	}
}

func TestTagsToAWSTagSet(t *testing.T) {
	if tagsToAWSTagSet(nil) != nil {
		t.Error("nil tags should produce nil tagset")
	}
	if tagsToAWSTagSet(map[string]string{}) != nil {
		t.Error("empty tags should produce nil tagset")
	}
	ts := tagsToAWSTagSet(map[string]string{"k": "v"})
	if len(ts) != 1 || aws.ToString(ts[0].Key) != "k" || aws.ToString(ts[0].Value) != "v" {
		t.Errorf("tagset = %v", ts)
	}
}

// --- provider registry + URL construction -----------------------------------

func TestGetProviderConfig(t *testing.T) {
	if GetProviderConfig(ProviderAWS).Name != ProviderAWS {
		t.Error("aws config lookup failed")
	}
	// Unknown provider falls back to the custom config (never nil).
	got := GetProviderConfig("not-a-real-provider")
	if got == nil || got.Name != ProviderCustom {
		t.Errorf("unknown provider should fall back to custom, got %+v", got)
	}
}

func TestGetAllProvidersExcludesCustom(t *testing.T) {
	all := GetAllProviders()
	if len(all) == 0 {
		t.Fatal("expected non-empty provider list")
	}
	for _, p := range all {
		if p == ProviderCustom {
			t.Error("GetAllProviders must not expose the internal 'custom' fallback")
		}
	}
	// AWS must be present and first (priority order).
	if all[0] != ProviderAWS {
		t.Errorf("first provider = %q, want aws", all[0])
	}
}

func TestGenerateURLWithProvider(t *testing.T) {
	cases := []struct {
		name                       string
		bucket, key, region, endpt string
		ssl                        bool
		provider                   string
		want                       string
	}{
		{"aws us-east-1 virtual", "mb", "k.txt", "us-east-1", "", true, ProviderAWS, "https://mb.s3.amazonaws.com/k.txt"},
		{"aws regional virtual", "mb", "k.txt", "eu-west-1", "", true, ProviderAWS, "https://mb.s3.eu-west-1.amazonaws.com/k.txt"},
		{"aws http scheme", "mb", "k.txt", "us-east-1", "", false, ProviderAWS, "http://mb.s3.amazonaws.com/k.txt"},
		{"digitalocean", "mb", "k.txt", "nyc3", "", true, ProviderDigitalOcean, "https://mb.nyc3.digitaloceanspaces.com/k.txt"},
		{"cloudflare path style", "mb", "k.txt", "auto", "", true, ProviderCloudflare, "https://{account_id}.r2.cloudflarestorage.com/mb/k.txt"},
		{"google path style", "mb", "k.txt", "us-central1", "", true, ProviderGoogle, "https://storage.googleapis.com/mb/k.txt"},
		{"custom endpoint with scheme", "mb", "k.txt", "us-east-1", "https://my.endpoint.com", true, ProviderCustom, "https://my.endpoint.com/mb/k.txt"},
		{"custom endpoint no scheme https", "mb", "k.txt", "us-east-1", "my.endpoint.com", true, ProviderCustom, "https://my.endpoint.com/mb/k.txt"},
		{"custom endpoint no scheme http", "mb", "k.txt", "us-east-1", "my.endpoint.com", false, ProviderCustom, "http://my.endpoint.com/mb/k.txt"},
		{"unknown provider falls back", "mb", "k.txt", "us-east-1", "", true, "unknown", "https://localhost:9000/mb/k.txt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := GenerateURLWithProvider(c.bucket, c.key, c.region, c.endpt, c.ssl, c.provider)
			if got != c.want {
				t.Errorf("GenerateURLWithProvider = %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseURLHelpers(t *testing.T) {
	t.Run("virtual-hosted", func(t *testing.T) {
		b, k, ok := parseVirtualHostedURL("bucket.s3.amazonaws.com", "/path/to/key.txt")
		if !ok || b != "bucket" || k != "path/to/key.txt" {
			t.Errorf("got %q,%q,%v", b, k, ok)
		}
	})
	t.Run("virtual-hosted single label fails", func(t *testing.T) {
		_, _, ok := parseVirtualHostedURL("singlelabel", "/x")
		if ok {
			t.Error("single-label host should not parse")
		}
	})
	t.Run("path-style with key", func(t *testing.T) {
		b, k, ok := parsePathStyleURL("host", "/bucket/path/to/key.txt")
		if !ok || b != "bucket" || k != "path/to/key.txt" {
			t.Errorf("got %q,%q,%v", b, k, ok)
		}
	})
	t.Run("path-style bucket only", func(t *testing.T) {
		b, k, ok := parsePathStyleURL("host", "/onlybucket")
		if !ok || b != "onlybucket" || k != "" {
			t.Errorf("got %q,%q,%v", b, k, ok)
		}
	})
	t.Run("path-style empty fails", func(t *testing.T) {
		_, _, ok := parsePathStyleURL("host", "/")
		if ok {
			t.Error("empty path should not parse")
		}
	})
}

// --- result marshalers ------------------------------------------------------

func dictString(t *testing.T, d *starlark.Dict, key string) string {
	t.Helper()
	v, found, err := d.Get(starlark.String(key))
	if err != nil || !found {
		t.Fatalf("dict missing key %q (found=%v err=%v)", key, found, err)
	}
	s, ok := starlark.AsString(v)
	if !ok {
		t.Fatalf("key %q is not a string: %v", key, v)
	}
	return s
}

func TestBucketInfoMarshalStarlark(t *testing.T) {
	bi := &BucketInfo{
		Name: "mb", Region: "us-east-1", VersioningStatus: "Enabled",
		EncryptionEnabled: true, ObjectCount: 7, TotalSize: 4096,
		Tags: map[string]string{"env": "prod"},
	}
	v, err := bi.MarshalStarlark()
	if err != nil {
		t.Fatalf("MarshalStarlark error: %v", err)
	}
	d, ok := v.(*starlark.Dict)
	if !ok {
		t.Fatalf("expected *starlark.Dict, got %T", v)
	}
	if dictString(t, d, "name") != "mb" {
		t.Errorf("name = %q", dictString(t, d, "name"))
	}
	if dictString(t, d, "region") != "us-east-1" {
		t.Errorf("region wrong")
	}
	count, _, _ := d.Get(starlark.String("object_count"))
	if count.String() != "7" {
		t.Errorf("object_count = %v", count)
	}
	enc, _, _ := d.Get(starlark.String("encryption_enabled"))
	if enc != starlark.Bool(true) {
		t.Errorf("encryption_enabled = %v", enc)
	}
	tags, _, _ := d.Get(starlark.String("tags"))
	td := tags.(*starlark.Dict)
	if dictString(t, td, "env") != "prod" {
		t.Errorf("tags.env wrong")
	}
}

func TestObjectInfoMarshalStarlark(t *testing.T) {
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	oi := &ObjectInfo{
		Key: "k.txt", Size: 100, ETag: "etag", ContentType: "text/plain",
		Expires: &exp, Metadata: map[string]string{"a": "1"},
	}
	v, err := oi.MarshalStarlark()
	if err != nil {
		t.Fatalf("MarshalStarlark error: %v", err)
	}
	d := v.(*starlark.Dict)
	if dictString(t, d, "key") != "k.txt" {
		t.Errorf("key wrong")
	}
	if dictString(t, d, "content_type") != "text/plain" {
		t.Errorf("content_type wrong")
	}
	size, _, _ := d.Get(starlark.String("size"))
	if size.String() != "100" {
		t.Errorf("size = %v", size)
	}
	expVal, _, _ := d.Get(starlark.String("expires"))
	if expVal == starlark.None {
		t.Error("expires should be set, not None")
	}

	// Nil expires marshals as None (not a panic).
	oi2 := &ObjectInfo{Key: "k"}
	v2, _ := oi2.MarshalStarlark()
	d2 := v2.(*starlark.Dict)
	if got, _, _ := d2.Get(starlark.String("expires")); got != starlark.None {
		t.Errorf("nil expires should marshal as None, got %v", got)
	}
}

func TestListObjectsResultMarshalStarlark(t *testing.T) {
	res := &ListObjectsResult{
		Contents: []ObjectInfo{{Key: "a.txt", Size: 1}, {Key: "b.txt", Size: 2}},
	}
	v, err := res.MarshalStarlark()
	if err != nil {
		t.Fatalf("MarshalStarlark error: %v", err)
	}
	l, ok := v.(*starlark.List)
	if !ok {
		t.Fatalf("expected *starlark.List, got %T", v)
	}
	if l.Len() != 2 {
		t.Fatalf("len = %d, want 2", l.Len())
	}
	first := l.Index(0).(*starlark.Dict)
	if dictString(t, first, "key") != "a.txt" {
		t.Errorf("first key = %q", dictString(t, first, "key"))
	}

	// Empty result marshals to an empty list, not nil.
	empty, err := (&ListObjectsResult{}).MarshalStarlark()
	if err != nil {
		t.Fatalf("empty marshal error: %v", err)
	}
	if empty.(*starlark.List).Len() != 0 {
		t.Error("empty result should marshal to empty list")
	}
}

// --- ClientWrapper Starlark-value protocol ----------------------------------

func newTestWrapper(t *testing.T) *ClientWrapper {
	t.Helper()
	cfg := &ClientConfig{ServiceType: ProviderAWS, Region: "us-east-1"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validate: %v", err)
	}
	client, err := NewClient(nil, cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return NewClientWrapper(client)
}

func TestClientWrapperProtocol(t *testing.T) {
	cw := newTestWrapper(t)

	if cw.Type() != "s3.Client" {
		t.Errorf("Type() = %q", cw.Type())
	}
	if cw.Truth() != starlark.True {
		t.Error("client should be truthy")
	}
	cw.Freeze() // must not panic

	if _, err := cw.Hash(); err == nil {
		t.Error("client should be unhashable")
	} else if !strings.Contains(err.Error(), "unhashable") {
		t.Errorf("hash error = %q", err.Error())
	}

	s := cw.String()
	if !strings.Contains(s, "s3.Client") || !strings.Contains(s, "aws") || !strings.Contains(s, "us-east-1") {
		t.Errorf("String() = %q", s)
	}

	// Attr resolves a real method and rejects a bogus one.
	if v, err := cw.Attr("put_object"); err != nil || v == nil {
		t.Errorf("Attr(put_object) = %v, %v", v, err)
	}
	if _, err := cw.Attr("does_not_exist"); err == nil {
		t.Error("Attr on unknown name should error")
	} else if !strings.Contains(err.Error(), "no .does_not_exist attribute") {
		t.Errorf("attr error = %q", err.Error())
	}

	// AttrNames lists every registered method.
	names := cw.AttrNames()
	want := []string{
		"get_client_info", "create_bucket", "delete_bucket", "list_buckets",
		"bucket_exists", "get_bucket_info", "put_object", "put_object_file",
		"get_object", "get_object_file", "delete_object", "list_objects",
		"object_exists", "get_object_info", "set_object_info", "copy_object",
		"presign_url", "get_public_url",
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("AttrNames missing %q", w)
		}
	}
	if len(names) != len(want) {
		t.Errorf("AttrNames count = %d, want %d", len(names), len(want))
	}
}

// TestResolveFilePathSandbox verifies put_object_file / get_object_file paths are
// confined to file_root: a script cannot read or write arbitrary host files. A
// `..` escape is rejected; an "absolute" path is re-anchored UNDER the root
// (never reaching the real host path); allow_unsafe_file_paths bypasses.
func TestResolveFilePathSandbox(t *testing.T) {
	root := t.TempDir()
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = root
	}
	cw := &ClientWrapper{fileRoot: root}

	// Within the root: resolves under it.
	got, err := cw.resolveFilePath("sub/data.bin")
	if err != nil {
		t.Fatalf("in-root path rejected: %v", err)
	}
	if !strings.HasPrefix(got, realRoot) {
		t.Errorf("resolved %q should be under root %q", got, realRoot)
	}

	// A `..` escape must be rejected.
	if _, err := cw.resolveFilePath("../../../etc/passwd"); err == nil {
		t.Error("`..` escape must be rejected")
	}

	// An "absolute" path is re-anchored under root (not the real /etc/passwd).
	abs, err := cw.resolveFilePath("/etc/passwd")
	if err != nil {
		t.Fatalf("absolute path should re-anchor under root, got error: %v", err)
	}
	if !strings.HasPrefix(abs, realRoot) || abs == "/etc/passwd" {
		t.Errorf("absolute path must be confined under root, got %q", abs)
	}

	// The opt-out disables confinement.
	cwUnsafe := &ClientWrapper{allowUnsafeFilePaths: true}
	if p, err := cwUnsafe.resolveFilePath("/etc/passwd"); err != nil || p != "/etc/passwd" {
		t.Errorf("allow_unsafe_file_paths should pass the path through, got (%q, %v)", p, err)
	}

	// An empty / unresolved file_root fails CLOSED — every path is rejected, so a
	// misconfiguration can never silently fall back to a movable working-dir jail.
	cwClosed := &ClientWrapper{}
	if _, err := cwClosed.resolveFilePath("anything.txt"); err == nil {
		t.Error("an empty file_root must fail closed (reject all paths)")
	}
}

// TestFileRootSnapshotImmuneToChdir verifies the jail root is captured absolute at
// module construction (in Go, before any script runs), so a script cannot move it
// by changing the working directory — whether before or after create_client.
func TestFileRootSnapshotImmuneToChdir(t *testing.T) {
	m := NewModule()
	snapshot := m.fileRoot
	if !filepath.IsAbs(snapshot) {
		t.Fatalf("module file_root should be absolute, got %q", snapshot)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	// The snapshot taken at NewModule must not follow a later chdir.
	if m.fileRoot != snapshot {
		t.Errorf("module file_root must not follow chdir: was %q, now %q", snapshot, m.fileRoot)
	}
	// A wrapper built from the snapshot confines under the pre-chdir root, not the
	// new working directory.
	cw := &ClientWrapper{}
	cw.setFileAccess(m.fileRoot, false)
	if !strings.HasPrefix(cw.fileRoot, snapshot) {
		t.Errorf("wrapper root %q should be the pre-chdir snapshot %q", cw.fileRoot, snapshot)
	}
}

// TestS3HostOnlyOptions verifies the DoS/safety levers cannot be changed from a
// script: no set_<key> builtin is generated for the host-only options, while the
// getters (and a normal option's setter) remain.
func TestS3HostOnlyOptions(t *testing.T) {
	dict, err := NewModule().LoadModule()()
	if err != nil {
		t.Fatalf("LoadModule: %v", err)
	}
	mod, ok := dict[ModuleName].(starlark.HasAttrs)
	if !ok {
		t.Fatalf("module %q is not attr-accessible: %T", ModuleName, dict[ModuleName])
	}
	attrs := make(map[string]bool)
	for _, n := range mod.AttrNames() {
		attrs[n] = true
	}
	for _, absent := range []string{"set_file_root", "set_allow_unsafe_file_paths", "set_max_object_size"} {
		if attrs[absent] {
			t.Errorf("%s must not be script-settable (host-only)", absent)
		}
	}
	for _, present := range []string{"get_file_root", "get_allow_unsafe_file_paths", "get_max_object_size", "set_region"} {
		if !attrs[present] {
			t.Errorf("%s builtin should be present", present)
		}
	}
}

// TestMaxObjectSizeWiring verifies max_object_size reaches the wrapper get_object
// reads from — via create_client — and honors the env override.
func TestMaxObjectSizeWiring(t *testing.T) {
	newWrapper := func(t *testing.T) *ClientWrapper {
		t.Helper()
		m := NewModule()
		b := starlark.NewBuiltin("s3.create_client", m.starCreateClient)
		v, err := m.starCreateClient(&starlark.Thread{}, b, nil, nil)
		if err != nil {
			t.Fatalf("create_client: %v", err)
		}
		cw, ok := v.(*ClientWrapper)
		if !ok {
			t.Fatalf("create_client returned %T, want *ClientWrapper", v)
		}
		return cw
	}

	if got := newWrapper(t).maxObjectSize; got != defaultMaxObjectSize {
		t.Errorf("default wrapper max_object_size = %d, want %d", got, defaultMaxObjectSize)
	}

	t.Setenv("S3_MAX_OBJECT_SIZE", "1048576")
	if got := newWrapper(t).maxObjectSize; got != 1048576 {
		t.Errorf("env wrapper max_object_size = %d, want 1048576", got)
	}
}

// TestResolveMaxObjectSize verifies the fail-safe clamp: 0 stays unlimited
// (explicit), a positive value is honored, and a negative value (misconfiguration)
// falls back to the default rather than silently disabling the cap.
func TestResolveMaxObjectSize(t *testing.T) {
	cases := map[int]int{
		0:    0,                    // explicit unlimited
		1024: 1024,                 // honored
		-1:   defaultMaxObjectSize, // fail safe, not fail open
	}
	for in, want := range cases {
		if got := resolveMaxObjectSize(in); got != want {
			t.Errorf("resolveMaxObjectSize(%d) = %d, want %d", in, got, want)
		}
	}
}
