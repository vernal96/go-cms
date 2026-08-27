package postgres

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/mail"
)

func TestMigrationSourceDefinesSiteScopedMailHistoryAndOutboxDependencies(t *testing.T) {
	t.Parallel()
	sources := (&Database{}).MigrationSources()
	if len(sources) != 1 || sources[0].ID != "mail" || sources[0].Schema != "mail" {
		t.Fatalf("migration sources = %#v", sources)
	}
	entries, err := fs.ReadDir(sources[0].FS, sources[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("migration files = %#v", entries)
	}
	raw, err := fs.ReadFile(sources[0].FS, sources[0].Path+"/000001_mail.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, required := range []string{
		"CREATE TABLE mail.templates",
		"UNIQUE (site_id, code)",
		"CREATE TABLE mail.messages",
		"template_id       BIGINT      NULL REFERENCES mail.templates (id) ON DELETE SET NULL",
		"CREATE TABLE mail.delivery_attempts",
		"message_id        BIGINT      NOT NULL REFERENCES mail.messages (id) ON DELETE CASCADE",
		"WHERE status IN ('accepted', 'failed')",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("mail migration is missing %q", required)
		}
	}
	if (&Database{}).ModuleCode() != mail.ModuleCode {
		t.Fatal("database module code mismatch")
	}
}

func TestVariableCodecUsesStableTypedChoiceJSON(t *testing.T) {
	t.Parallel()
	required := false
	raw, err := encodeVariables([]field.Definition{{
		Key: "kind", Type: field.TypeSelect, Label: "Kind", Required: &required,
		Options: field.SelectOptions{Choices: []field.Choice{{Value: "news", Label: "News"}}, Multiple: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"choices":[{"value":"news","label":"News"}]`) || strings.Contains(string(raw), `"Value"`) {
		t.Fatalf("variable JSON = %s", raw)
	}
	decoded, err := decodeVariables(raw)
	if err != nil {
		t.Fatal(err)
	}
	options, ok := decoded[0].Options.(field.SelectOptions)
	if !ok || !options.Multiple || len(options.Choices) != 1 || options.Choices[0].Value != "news" {
		t.Fatalf("decoded variables = %#v", decoded)
	}
}
