package s3

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectOptions contains options for object operations
type ObjectOptions struct {
	ContentType        string
	Metadata           map[string]string
	Tags               map[string]string
	CacheControl       string
	ContentEncoding    string
	ContentDisposition string
	ContentLanguage    string
	Expires            *time.Time
}

// NewObjectOptions creates a new ObjectOptions instance
func NewObjectOptions() *ObjectOptions {
	return &ObjectOptions{
		Metadata: make(map[string]string),
		Tags:     make(map[string]string),
	}
}

// WithContentType sets the content type
func (o *ObjectOptions) WithContentType(contentType string) *ObjectOptions {
	o.ContentType = contentType
	return o
}

// WithMetadata sets metadata
func (o *ObjectOptions) WithMetadata(metadata map[string]string) *ObjectOptions {
	o.Metadata = metadata
	return o
}

// WithTags sets tags
func (o *ObjectOptions) WithTags(tags map[string]string) *ObjectOptions {
	o.Tags = tags
	return o
}

// WithCacheControl sets cache control
func (o *ObjectOptions) WithCacheControl(cacheControl string) *ObjectOptions {
	o.CacheControl = cacheControl
	return o
}

// WithContentEncoding sets content encoding
func (o *ObjectOptions) WithContentEncoding(contentEncoding string) *ObjectOptions {
	o.ContentEncoding = contentEncoding
	return o
}

// WithContentDisposition sets content disposition
func (o *ObjectOptions) WithContentDisposition(contentDisposition string) *ObjectOptions {
	o.ContentDisposition = contentDisposition
	return o
}

// WithContentLanguage sets content language
func (o *ObjectOptions) WithContentLanguage(contentLanguage string) *ObjectOptions {
	o.ContentLanguage = contentLanguage
	return o
}

// WithExpires sets expires time
func (o *ObjectOptions) WithExpires(expires *time.Time) *ObjectOptions {
	o.Expires = expires
	return o
}

// ApplyToPutObject applies the options to a PutObjectInput
func (o *ObjectOptions) ApplyToPutObject(input *s3.PutObjectInput) {
	if o.ContentType != "" {
		input.ContentType = aws.String(o.ContentType)
	}
	if len(o.Metadata) > 0 {
		input.Metadata = o.Metadata
	}
	if o.CacheControl != "" {
		input.CacheControl = aws.String(o.CacheControl)
	}
	if o.ContentEncoding != "" {
		input.ContentEncoding = aws.String(o.ContentEncoding)
	}
	if o.ContentDisposition != "" {
		input.ContentDisposition = aws.String(o.ContentDisposition)
	}
	if o.ContentLanguage != "" {
		input.ContentLanguage = aws.String(o.ContentLanguage)
	}
	if o.Expires != nil {
		input.Expires = o.Expires
	}
}

// ApplyToCopyObject applies the options to a CopyObjectInput and sets metadata directive
func (o *ObjectOptions) ApplyToCopyObject(input *s3.CopyObjectInput) {
	needsReplace := false

	if o.ContentType != "" {
		input.ContentType = aws.String(o.ContentType)
		needsReplace = true
	}
	if len(o.Metadata) > 0 {
		input.Metadata = o.Metadata
		needsReplace = true
	}
	if o.CacheControl != "" {
		input.CacheControl = aws.String(o.CacheControl)
		needsReplace = true
	}
	if o.ContentEncoding != "" {
		input.ContentEncoding = aws.String(o.ContentEncoding)
		needsReplace = true
	}
	if o.ContentDisposition != "" {
		input.ContentDisposition = aws.String(o.ContentDisposition)
		needsReplace = true
	}
	if o.ContentLanguage != "" {
		input.ContentLanguage = aws.String(o.ContentLanguage)
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
	Prefix            string
	Delimiter         string
	MaxKeys           int
	ContinuationToken string
}

// NewListObjectsOptions creates a new ListObjectsOptions instance
func NewListObjectsOptions() *ListObjectsOptions {
	return &ListObjectsOptions{}
}

// WithPrefix sets the prefix for ListObjects
func (o *ListObjectsOptions) WithPrefix(prefix string) *ListObjectsOptions {
	o.Prefix = prefix
	return o
}

// WithDelimiter sets the delimiter for ListObjects
func (o *ListObjectsOptions) WithDelimiter(delimiter string) *ListObjectsOptions {
	o.Delimiter = delimiter
	return o
}

// WithMaxKeys sets the maximum number of keys for ListObjects
func (o *ListObjectsOptions) WithMaxKeys(maxKeys int) *ListObjectsOptions {
	o.MaxKeys = maxKeys
	return o
}

// WithContinuationToken sets the continuation token for ListObjects
func (o *ListObjectsOptions) WithContinuationToken(token string) *ListObjectsOptions {
	o.ContinuationToken = token
	return o
}

// ApplyToListObjects applies the options to a ListObjectsV2Input
func (o *ListObjectsOptions) ApplyToListObjects(input *s3.ListObjectsV2Input) {
	if o.Prefix != "" {
		input.Prefix = aws.String(o.Prefix)
	}
	if o.Delimiter != "" {
		input.Delimiter = aws.String(o.Delimiter)
	}
	if o.MaxKeys > 0 {
		input.MaxKeys = aws.Int32(int32(o.MaxKeys))
	}
	if o.ContinuationToken != "" {
		input.ContinuationToken = aws.String(o.ContinuationToken)
	}
}
