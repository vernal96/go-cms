package resourcetype

import (
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

func TestPageTypeNormalizesDefaultsAndRejectsIncompatibleFields(
	t *testing.T,
) {
	page := standardType(t, Page)
	templateCode := template.Code("article")

	normalized, err := page.Normalize(Payload{
		Template: &templateCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if normalized.ContentType == nil ||
		*normalized.ContentType != "html" {
		t.Fatalf("content type = %#v", normalized.ContentType)
	}
	if normalized.Settings == nil {
		t.Fatal("settings map was not initialized")
	}

	invalidContentType := "HTML"
	_, err = page.Normalize(Payload{
		Template:    &templateCode,
		ContentType: &invalidContentType,
	})
	if err == nil || !strings.Contains(err.Error(), "content_type") {
		t.Fatalf("invalid content type error = %v", err)
	}

	externalURL := "/outside"
	_, err = page.Normalize(Payload{
		Template:    &templateCode,
		ExternalURL: &externalURL,
	})
	if err == nil || !strings.Contains(err.Error(), "external_url") {
		t.Fatalf("external url error = %v", err)
	}
}

func TestLinkTypeURLPolicy(t *testing.T) {
	link := standardType(t, Link)

	for _, value := range []string{
		"https://example.com/docs",
		"http://example.com",
		"mailto:editor@example.com",
		"tel:+79991234567",
		"/relative/path?draft=1",
	} {
		t.Run(value, func(t *testing.T) {
			normalized, err := link.Normalize(Payload{
				ExternalURL: stringPointer(value),
			})
			if err != nil {
				t.Fatal(err)
			}
			if normalized.ExternalURL == nil ||
				*normalized.ExternalURL != value {
				t.Fatalf(
					"normalized external url = %#v",
					normalized.ExternalURL,
				)
			}
		})
	}

	for _, value := range []string{
		"",
		"example.com",
		"ftp://example.com",
		"//example.com/path",
		" https://example.com",
	} {
		t.Run("invalid_"+value, func(t *testing.T) {
			_, err := link.Normalize(Payload{
				ExternalURL: stringPointer(value),
			})
			if err == nil {
				t.Fatalf("accepted invalid url %q", value)
			}
		})
	}

	contentType := "html"
	_, err := link.Normalize(Payload{
		ContentType: &contentType,
		ExternalURL: stringPointer("/docs"),
	})
	if err == nil || !strings.Contains(err.Error(), "content_type") {
		t.Fatalf("incompatible content type error = %v", err)
	}
}

func TestResourceLinkTypeRequiresTarget(t *testing.T) {
	resourceLink := standardType(t, ResourceLink)

	if _, err := resourceLink.Normalize(Payload{}); err == nil {
		t.Fatal("resource_link accepted missing target")
	}

	targetID := int64(42)
	normalized, err := resourceLink.Normalize(Payload{
		TargetResourceID: &targetID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if normalized.TargetResourceID == nil ||
		*normalized.TargetResourceID != targetID {
		t.Fatalf("target id = %#v", normalized.TargetResourceID)
	}

	externalURL := "/docs"
	if _, err := resourceLink.Normalize(Payload{
		TargetResourceID: &targetID,
		ExternalURL:      &externalURL,
	}); err == nil || !strings.Contains(err.Error(), "external_url") {
		t.Fatalf("incompatible external url error = %v", err)
	}
}

func standardType(t *testing.T, code Code) Type {
	t.Helper()

	for _, resourceType := range StandardTypes() {
		if resourceType.Code() == code {
			return resourceType
		}
	}
	t.Fatalf("standard type %q is missing", code)
	return nil
}

func stringPointer(value string) *string {
	return &value
}
