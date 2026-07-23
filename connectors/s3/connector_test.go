package s3

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

type mockS3 struct {
	put    *awss3.PutObjectInput
	get    *awss3.GetObjectInput
	delete *awss3.DeleteObjectInput
}

func (*mockS3) HeadBucket(
	context.Context,
	*awss3.HeadBucketInput,
	...func(*awss3.Options),
) (*awss3.HeadBucketOutput, error) {
	return &awss3.HeadBucketOutput{}, nil
}

func (m *mockS3) PutObject(
	_ context.Context,
	input *awss3.PutObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.PutObjectOutput, error) {
	m.put = input
	_, _ = io.ReadAll(input.Body)
	return &awss3.PutObjectOutput{}, nil
}

func (m *mockS3) GetObject(
	_ context.Context,
	input *awss3.GetObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.GetObjectOutput, error) {
	m.get = input
	return &awss3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader("content")),
	}, nil
}

func (m *mockS3) DeleteObject(
	_ context.Context,
	input *awss3.DeleteObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.DeleteObjectOutput, error) {
	m.delete = input
	return &awss3.DeleteObjectOutput{}, nil
}

type mockPresigner struct {
	input   *awss3.GetObjectInput
	expires time.Duration
}

func (m *mockPresigner) PresignGetObject(
	_ context.Context,
	input *awss3.GetObjectInput,
	options ...func(*awss3.PresignOptions),
) (*v4.PresignedHTTPRequest, error) {
	m.input = input
	config := &awss3.PresignOptions{}
	for _, option := range options {
		option(config)
	}
	m.expires = config.Expires
	return &v4.PresignedHTTPRequest{
		URL:    "https://signed.example.test/object",
		Method: "GET",
	}, nil
}

func TestConnectorShapesObjectKeysAndURLs(t *testing.T) {
	client := &mockS3{}
	presigner := &mockPresigner{}
	baseURL, _ := url.Parse("https://cdn.example.test/assets")
	public := newConnector(Config{
		Code:       "public",
		Visibility: filesystem.VisibilityPublic,
		Bucket:     "bucket",
		Prefix:     "cms",
	}, client, presigner, baseURL)

	if err := public.PutNew(
		context.Background(),
		"objects/item",
		strings.NewReader("hello"),
		"text/plain",
	); err != nil {
		t.Fatal(err)
	}
	if client.put == nil ||
		*client.put.Key != "cms/objects/item" ||
		client.put.IfNoneMatch == nil ||
		*client.put.IfNoneMatch != "*" {
		t.Fatalf("put input = %#v", client.put)
	}
	rawURL, err := public.URL(
		context.Background(),
		filesystem.Reference{ID: "1", Path: "objects/item"},
	)
	if err != nil || rawURL != "https://cdn.example.test/assets/cms/objects/item" {
		t.Fatalf("public URL = %q, %v", rawURL, err)
	}

	private := newConnector(Config{
		Code:       "private",
		Visibility: filesystem.VisibilityPrivate,
		Bucket:     "bucket",
		Prefix:     "secure",
	}, client, presigner, nil)
	expiresAt := time.Now().Add(30 * time.Minute)
	signed, err := private.TemporaryURL(
		context.Background(),
		filesystem.Reference{ID: "2", Path: "objects/private"},
		expiresAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	if signed != "https://signed.example.test/object" ||
		presigner.input == nil ||
		*presigner.input.Key != "secure/objects/private" ||
		presigner.expires <= 0 {
		t.Fatalf(
			"signed URL = %q, input = %#v, expires = %s",
			signed,
			presigner.input,
			presigner.expires,
		)
	}
}
