package s3

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/filesystem"
)

func TestS3CompatibleIntegration(t *testing.T) {
	endpoint := os.Getenv("CMS_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("set CMS_TEST_S3_ENDPOINT to run the S3 integration test")
	}
	region := os.Getenv("CMS_TEST_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connector, err := New(ctx, Config{
		Code:            "integration",
		Visibility:      filesystem.VisibilityPrivate,
		Region:          region,
		Bucket:          os.Getenv("CMS_TEST_S3_BUCKET"),
		Prefix:          "cms-integration",
		Endpoint:        endpoint,
		UsePathStyle:    true,
		AccessKeyID:     os.Getenv("CMS_TEST_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CMS_TEST_S3_SECRET_ACCESS_KEY"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := connector.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	key := "objects/" + time.Now().UTC().Format("20060102150405.000000000")
	t.Cleanup(func() {
		_ = connector.Delete(context.Background(), key)
	})
	if err := connector.PutNew(
		ctx,
		key,
		strings.NewReader("integration"),
		"text/plain",
	); err != nil {
		t.Fatal(err)
	}
	body, err := connector.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || string(content) != "integration" {
		t.Fatalf("content = %q, %v", content, err)
	}
	if _, err := connector.TemporaryURL(
		ctx,
		filesystem.Reference{ID: "1", Path: key},
		time.Now().Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := connector.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
}
