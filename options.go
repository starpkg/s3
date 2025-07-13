package s3

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectOptions contains options for object operations
type ObjectOptions struct {
	ContentType        *string
	Metadata           *map[string]string
	Tags               *map[string]string
	CacheControl       *string
	ContentEncoding    *string
	ContentDisposition *string
	ContentLanguage    *string
	Expires            *time.Time
}

// NewObjectOptions creates a new ObjectOptions instance
func NewObjectOptions() *ObjectOptions {
	return &ObjectOptions{}
}

// validate returns true if the options contain any non-nil values
func (o *ObjectOptions) validate() bool {
	return o.ContentType != nil ||
		o.Metadata != nil ||
		o.Tags != nil ||
		o.CacheControl != nil ||
		o.ContentEncoding != nil ||
		o.ContentDisposition != nil ||
		o.ContentLanguage != nil ||
		o.Expires != nil
}

// ApplyToPutObject applies the options to a PutObjectInput
func (o *ObjectOptions) ApplyToPutObject(input *s3.PutObjectInput) {
	if o.ContentType != nil {
		input.ContentType = o.ContentType
	}
	if o.Metadata != nil {
		input.Metadata = *o.Metadata
	}
	if o.CacheControl != nil {
		input.CacheControl = o.CacheControl
	}
	if o.ContentEncoding != nil {
		input.ContentEncoding = o.ContentEncoding
	}
	if o.ContentDisposition != nil {
		input.ContentDisposition = o.ContentDisposition
	}
	if o.ContentLanguage != nil {
		input.ContentLanguage = o.ContentLanguage
	}
	if o.Expires != nil {
		input.Expires = o.Expires
	}
	if o.Tags != nil {
		// Convert tags to URL-encoded string format
		var tagPairs []string
		for k, v := range *o.Tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		input.Tagging = aws.String(strings.Join(tagPairs, "&"))
	}
}

// ApplyToCopyObject applies the options to a CopyObjectInput and sets metadata directive
func (o *ObjectOptions) ApplyToCopyObject(input *s3.CopyObjectInput) {
	needsReplace := false

	if o.ContentType != nil {
		input.ContentType = o.ContentType
		needsReplace = true
	}
	if o.Metadata != nil {
		input.Metadata = *o.Metadata
		needsReplace = true
	}
	if o.CacheControl != nil {
		input.CacheControl = o.CacheControl
		needsReplace = true
	}
	if o.ContentEncoding != nil {
		input.ContentEncoding = o.ContentEncoding
		needsReplace = true
	}
	if o.ContentDisposition != nil {
		input.ContentDisposition = o.ContentDisposition
		needsReplace = true
	}
	if o.ContentLanguage != nil {
		input.ContentLanguage = o.ContentLanguage
		needsReplace = true
	}
	if o.Expires != nil {
		input.Expires = o.Expires
		needsReplace = true
	}

	// Only set metadata directive if we're actually changing something
	if needsReplace {
		input.MetadataDirective = types.MetadataDirectiveReplace
	}
}

// ListObjectsOptions configures ListObjects operations
type ListObjectsOptions struct {
	Prefix            *string
	Delimiter         *string
	MaxKeys           *int
	ContinuationToken *string
}

// NewListObjectsOptions creates a new ListObjectsOptions instance
func NewListObjectsOptions() *ListObjectsOptions {
	return &ListObjectsOptions{}
}

// validate returns true if the options contain any non-nil values
func (o *ListObjectsOptions) validate() bool {
	return o.Prefix != nil ||
		o.Delimiter != nil ||
		o.MaxKeys != nil ||
		o.ContinuationToken != nil
}

// ApplyToListObjects applies the options to a ListObjectsV2Input
func (o *ListObjectsOptions) ApplyToListObjects(input *s3.ListObjectsV2Input) {
	if o.Prefix != nil {
		input.Prefix = o.Prefix
	}
	if o.Delimiter != nil {
		input.Delimiter = o.Delimiter
	}
	if o.MaxKeys != nil {
		input.MaxKeys = aws.Int32(int32(*o.MaxKeys))
	}
	if o.ContinuationToken != nil {
		input.ContinuationToken = o.ContinuationToken
	}
}
