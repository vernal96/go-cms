package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/forms"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestExternalModuleHTTPPersistenceAndSiteIsolation(t *testing.T) {
	if os.Getenv("EXAMPLE_DATABASE") == "" {
		t.Skip("set EXAMPLE_DATABASE and PG* for the isolated external application test")
	}
	application, handler, err := newFixture(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	server := httptest.NewServer(handler)
	defer server.Close()
	call := func(method, path string, body any, want int) []byte {
		t.Helper()
		var data []byte
		if body != nil {
			data, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req, _ := http.NewRequest(method, server.URL+path, bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+os.Getenv("EXAMPLE_TOKEN"))
		req.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		if response.StatusCode != want {
			t.Fatalf("%s %s: %d %s", method, path, response.StatusCode, raw)
		}
		return raw
	}
	metadata := call("GET", "/api/site-profiles", nil, 200)
	if !bytes.Contains(metadata, []byte(`"editor":"example.text"`)) {
		t.Fatalf("custom field missing: %s", metadata)
	}
	navigation := call("GET", "/api/admin/navigation?site_id=1", nil, 200)
	if !bytes.Contains(navigation, []byte(`"code":"example.notice"`)) {
		t.Fatalf("configured permission missing from navigation: %s", navigation)
	}
	for id := 1; id <= 2; id++ {
		runtime, ok := application.Sites().RuntimeByID(site.ID(id))
		if !ok {
			t.Fatalf("missing site %d", id)
		}
		_, exists := runtime.Profile().Registry().Module("example")
		if exists != (id == 1) {
			t.Fatalf("module leaked for site %d", id)
		}
	}
	call("PATCH", "/api/sites/1", map[string]any{"profile_code": "extended", "domain": "extended.example.test", "locale": "ru-RU", "is_public": true, "settings": map[string]any{"message": "backend round trip", "messages": []string{"first", "second"}}}, 200)
	if raw := call("GET", "/api/sites/1", nil, 200); !bytes.Contains(raw, []byte("backend round trip")) || !bytes.Contains(raw, []byte(`"messages":["first","second"]`)) {
		t.Fatalf("custom field was not persisted: %s", raw)
	}
	if raw := call("GET", "/api/sites/2", nil, 200); bytes.Contains(raw, []byte("backend round trip")) {
		t.Fatal("site settings leaked")
	}
	for id := 1; id <= 2; id++ {
		root := fmt.Sprintf("/api/sites/%d/forms/forms", id)
		var list struct {
			Items []forms.Form `json:"items"`
		}
		json.Unmarshal(call("GET", root, nil, 200), &list)
		var form forms.Form
		for _, item := range list.Items {
			if item.Code == "extension" {
				form = item
			}
		}
		if form.ID == 0 {
			json.Unmarshal(call("POST", root, map[string]any{"code": "extension", "name": "External extension", "description": "SDK example", "enabled": true}, 201), &form)
		}
		editorPath := fmt.Sprintf("%s/%d/editor", root, form.ID)
		metadata := call("GET", editorPath, nil, 200)
		if bytes.Contains(metadata, []byte(`"code":"example.notice"`)) != (id == 1) {
			t.Fatalf("element metadata scope: %s", metadata)
		}
		elementsPath := fmt.Sprintf("%s/%d/elements", root, form.ID)
		var editor struct {
			Elements []forms.Element `json:"elements"`
		}
		json.Unmarshal(metadata, &editor)
		var element forms.Element
		for _, item := range editor.Elements {
			if item.Type == "example.notice" {
				element = item
			}
		}
		payload := map[string]any{"code": "notice", "type": "example.notice", "config": map[string]any{"text": "element round trip"}}
		if id == 2 {
			payload["position"] = 3
			call("POST", elementsPath, payload, 422)
			continue
		}
		if element.ID == 0 {
			payload["position"] = 3
			call("POST", elementsPath, payload, 201)
		} else {
			call("PATCH", fmt.Sprintf("%s/%d", elementsPath, element.ID), payload, 200)
		}
		if raw := call("GET", editorPath, nil, 200); !strings.Contains(string(raw), "element round trip") {
			t.Fatalf("element not persisted: %s", raw)
		}
	}
}
