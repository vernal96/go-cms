package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"testing"
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/seeds"
)

func TestDevSiteSeedSource(t *testing.T) {
	sources := (&Database{}).SeedSources()
	if len(sources) != 1 {
		t.Fatalf("seed sources = %#v", sources)
	}

	source := sources[0]
	if source.ID != "sites_dev" ||
		len(source.Tags) != 1 ||
		source.Tags[0] != "dev" ||
		source.Schema != "core" {
		t.Fatalf("dev site source = %#v", source)
	}
	if err := seeds.ValidateSource(source); err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadDir(source.FS, source.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("dev seed files = %#v", entries)
	}
}

func TestPostgresMigrationsAndSiteRepository(t *testing.T) {
	host := os.Getenv("CMS_TEST_POSTGRES_HOST")
	if host == "" {
		t.Skip("set CMS_TEST_POSTGRES_HOST to run the PostgreSQL integration test")
	}

	port := 5432
	if value := os.Getenv("CMS_TEST_POSTGRES_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("parse CMS_TEST_POSTGRES_PORT: %v", err)
		}
		port = parsed
	}

	sslMode := os.Getenv("CMS_TEST_POSTGRES_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connector, err := connectorpostgres.New(ctx, connectorpostgres.Config{
		Code:            kernel.ConnectionCode("integration"),
		Host:            host,
		Port:            port,
		Database:        os.Getenv("CMS_TEST_POSTGRES_DB"),
		User:            os.Getenv("CMS_TEST_POSTGRES_USER"),
		Password:        os.Getenv("CMS_TEST_POSTGRES_PASSWORD"),
		SSLMode:         sslMode,
		MaxConns:        4,
		MinConns:        0,
		ConnMaxLifetime: time.Minute,
		ConnectTimeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connector.Close() })

	if err := connector.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	database, err := NewDatabase(connector)
	if err != nil {
		t.Fatal(err)
	}

	plan := migrations.Plan{
		Connection: string(connector.Code()),
		Target:     connector,
		Source:     database.MigrationSources()[0],
	}
	manager := migrations.NewManager()

	restoreMigration := false
	t.Cleanup(func() {
		if restoreMigration {
			_ = manager.Up(context.Background(), plan)
		}
	})

	if err := manager.Up(ctx, plan); err != nil {
		t.Fatalf("up: %v", err)
	}

	version, hasVersion, dirty, err := manager.Version(ctx, plan)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != 3 || !hasVersion || dirty {
		t.Fatalf(
			"version = %d, hasVersion = %t, dirty = %t",
			version,
			hasVersion,
			dirty,
		)
	}

	var sitesTable *string
	var resourcesTable *string
	var resourcePathIndex *string
	var fileFoldersTable *string
	var filesTable *string
	if err := connector.Pool().QueryRow(
		ctx,
		`
SELECT
    to_regclass('core.sites')::text,
    to_regclass('core.resources')::text,
    to_regclass('core.uq_resources_site_path')::text,
    to_regclass('core.file_folders')::text,
    to_regclass('core.files')::text;
`,
	).Scan(
		&sitesTable,
		&resourcesTable,
		&resourcePathIndex,
		&fileFoldersTable,
		&filesTable,
	); err != nil {
		t.Fatal(err)
	}
	if sitesTable == nil || *sitesTable != "core.sites" {
		t.Fatalf("core.sites = %#v", sitesTable)
	}
	if resourcesTable == nil ||
		*resourcesTable != "core.resources" {
		t.Fatalf("core.resources = %#v", resourcesTable)
	}
	if resourcePathIndex == nil ||
		*resourcePathIndex != "core.uq_resources_site_path" {
		t.Fatalf(
			"resource path index = %#v",
			resourcePathIndex,
		)
	}
	if fileFoldersTable == nil ||
		*fileFoldersTable != "core.file_folders" {
		t.Fatalf("core.file_folders = %#v", fileFoldersTable)
	}
	if filesTable == nil || *filesTable != "core.files" {
		t.Fatalf("core.files = %#v", filesTable)
	}

	if _, err := connector.Pool().Exec(ctx, `
DELETE FROM core.files;
DELETE FROM core.file_folders;
`); err != nil {
		t.Fatal(err)
	}
	fileRepository := database.Files()
	sourceFolder, err := fileRepository.CreateFolder(ctx, corefile.Folder{
		Storage: "public",
		Name:    "source",
	})
	if err != nil {
		t.Fatalf("create file folder: %v", err)
	}
	namespaceFolder, err := fileRepository.CreateFolder(ctx, corefile.Folder{
		Storage: "public",
		Name:    "shared",
	})
	if err != nil {
		t.Fatalf("create namespace folder: %v", err)
	}
	checksum := sha256.Sum256([]byte("original"))
	_, err = fileRepository.CreateFile(ctx, corefile.File{
		Storage:        "public",
		Name:           "shared",
		MIMEType:       "text/plain",
		Size:           8,
		ChecksumSHA256: hex.EncodeToString(checksum[:]),
		Path:           "objects/shared",
	})
	if !errors.Is(err, corefile.ErrConflict) {
		t.Fatalf("shared file/folder namespace error = %v", err)
	}
	original, err := fileRepository.CreateFile(ctx, corefile.File{
		FolderID:       &sourceFolder.ID,
		Storage:        "public",
		Name:           "original.txt",
		MIMEType:       "text/plain",
		Size:           8,
		ChecksumSHA256: hex.EncodeToString(checksum[:]),
		Path:           "objects/original",
	})
	if err != nil {
		t.Fatalf("create original file: %v", err)
	}
	derivedChecksum := sha256.Sum256([]byte("derived"))
	derived, err := fileRepository.CreateFile(ctx, corefile.File{
		Storage:        "private",
		Name:           "derived.txt",
		MIMEType:       "text/plain",
		Size:           7,
		ChecksumSHA256: hex.EncodeToString(derivedChecksum[:]),
		Path:           "objects/derived",
		ParentID:       &original.ID,
	})
	if err != nil {
		t.Fatalf("create cross-storage derived file: %v", err)
	}
	physicalFailure := errors.New("physical delete failed")
	if err := fileRepository.DeleteFolder(
		ctx,
		sourceFolder.ID,
		func(context.Context, []corefile.File) error {
			return physicalFailure
		},
	); !errors.Is(err, physicalFailure) {
		t.Fatalf("delete rollback error = %v", err)
	}
	if _, err := fileRepository.FileByID(
		ctx,
		original.ID,
	); err != nil {
		t.Fatalf("delete failure removed metadata: %v", err)
	}
	var deletionPlan []corefile.File
	if err := fileRepository.DeleteFolder(
		ctx,
		sourceFolder.ID,
		func(_ context.Context, items []corefile.File) error {
			deletionPlan = append(deletionPlan, items...)
			return nil
		},
	); err != nil {
		t.Fatalf("delete file folder: %v", err)
	}
	if len(deletionPlan) != 2 {
		t.Fatalf("cross-storage deletion plan = %#v", deletionPlan)
	}
	if _, err := fileRepository.FileByID(
		ctx,
		derived.ID,
	); !errors.Is(err, corefile.ErrNotFound) {
		t.Fatalf("derived metadata after folder delete = %v", err)
	}
	if err := fileRepository.DeleteFolder(
		ctx,
		namespaceFolder.ID,
		func(context.Context, []corefile.File) error { return nil },
	); err != nil {
		t.Fatalf("delete namespace folder: %v", err)
	}

	seedPlan := seeds.Plan{
		Connection: string(connector.Code()),
		Module:     core.ModuleCode,
		Target:     connector,
		Source:     database.SeedSources()[0],
	}
	seedManager := seeds.NewManager()
	if err := seedManager.Force(ctx, seedPlan, -1); err != nil {
		t.Fatalf("prepare seed state: %v", err)
	}
	if _, err := connector.Pool().Exec(ctx, `
DELETE
FROM core.sites
WHERE profile_code = 'dev'
  AND domain IN ('localhost', 'example.com');
`); err != nil {
		t.Fatalf("clean seeded sites: %v", err)
	}
	if err := seedManager.Up(ctx, seedPlan); err != nil {
		t.Fatalf("seed up: %v", err)
	}

	loadedSites, err := database.Sites().List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	found := make(map[string]bool, 2)
	siteIDs := make(map[string]site.ID, 2)
	for _, item := range loadedSites {
		if item.Domain != "localhost" && item.Domain != "example.com" {
			continue
		}

		found[item.Domain] = true
		siteIDs[item.Domain] = item.ID
		if item.ProfileCode != "dev" || item.Locale != "ru-RU" {
			t.Fatalf("seeded site = %#v", item)
		}
		rawSettings, err := json.Marshal(item.Settings)
		if err != nil {
			t.Fatal(err)
		}
		if string(rawSettings) != `{}` {
			t.Fatalf("settings = %s", rawSettings)
		}
	}
	if !found["localhost"] || !found["example.com"] {
		t.Fatalf("seeded domains = %#v", found)
	}

	if _, err := connector.Pool().Exec(ctx, `
UPDATE core.sites
SET updated_at = '2000-01-01T00:00:00Z'
WHERE id = $1;
`, siteIDs["localhost"]); err != nil {
		t.Fatalf("prepare site update timestamp: %v", err)
	}
	if err := database.Sites().UpdateSettings(
		ctx,
		siteIDs["localhost"],
		map[string]any{
			"count": int64(3),
			"flag":  false,
		},
	); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	loadedSites, err = database.Sites().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	updatedSiteFound := false
	for _, item := range loadedSites {
		if item.ID != siteIDs["localhost"] {
			continue
		}
		updatedSiteFound = true

		if item.Settings["count"] != json.Number("3") ||
			item.Settings["flag"] != false {
			t.Fatalf("updated settings = %#v", item.Settings)
		}

		var updatedAt time.Time
		if err := connector.Pool().QueryRow(
			ctx,
			"SELECT updated_at FROM core.sites WHERE id = $1",
			item.ID,
		).Scan(&updatedAt); err != nil {
			t.Fatal(err)
		}
		if !updatedAt.After(time.Date(
			2000,
			time.January,
			1,
			0,
			0,
			0,
			0,
			time.UTC,
		)) {
			t.Fatalf("updated_at was not changed: %v", updatedAt)
		}
	}
	if !updatedSiteFound {
		t.Fatal("updated site was not returned by repository")
	}

	resourceRepository := database.Resources()
	templateCode := template.Code("article")
	contentType := "html"
	rootPath := "/"
	root, err := resourceRepository.Create(ctx, resource.Resource{
		SiteID:       siteIDs["localhost"],
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Home",
		Path:         &rootPath,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Settings:     map[string]any{"headline": "Home"},
	})
	if err != nil {
		t.Fatalf("create root resource: %v", err)
	}

	for _, slug := range []string{"no-path-one", "no-path-two"} {
		noPath, err := resourceRepository.Create(
			ctx,
			resource.Resource{
				SiteID:       siteIDs["localhost"],
				Type:         "no_path",
				Title:        slug,
				Slug:         slug,
				IsPublic:     true,
				IsSearchable: true,
				InMenu:       true,
				InSitemap:    true,
				Settings:     map[string]any{},
			},
		)
		if err != nil {
			t.Fatalf("create nullable-path resource: %v", err)
		}
		if noPath.Path != nil {
			t.Fatalf("nullable resource path = %#v", noPath.Path)
		}
	}

	sectionPath := "/section"
	section, err := resourceRepository.Create(
		ctx,
		resource.Resource{
			SiteID:       siteIDs["localhost"],
			Type:         resourcetype.Page,
			Template:     &templateCode,
			ContentType:  &contentType,
			Title:        "Section",
			Slug:         "section",
			Path:         &sectionPath,
			IsPublic:     true,
			IsSearchable: true,
			InMenu:       true,
			InSitemap:    true,
			Settings:     map[string]any{},
		},
	)
	if err != nil {
		t.Fatalf("create section resource: %v", err)
	}

	childPath := "/child"
	child, err := resourceRepository.Create(ctx, resource.Resource{
		SiteID:       siteIDs["localhost"],
		ParentID:     &root.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Child",
		Slug:         "child",
		Path:         &childPath,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Settings:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("create child resource: %v", err)
	}

	grandchildPath := "/child/grandchild"
	grandchild, err := resourceRepository.Create(
		ctx,
		resource.Resource{
			SiteID:       siteIDs["localhost"],
			ParentID:     &child.ID,
			Type:         resourcetype.Page,
			Template:     &templateCode,
			ContentType:  &contentType,
			Title:        "Grandchild",
			Slug:         "grandchild",
			Path:         &grandchildPath,
			IsPublic:     true,
			IsSearchable: true,
			InMenu:       true,
			InSitemap:    true,
			Settings:     map[string]any{},
		},
	)
	if err != nil {
		t.Fatalf("create grandchild resource: %v", err)
	}

	loadedChild, err := resourceRepository.ByPath(
		ctx,
		siteIDs["localhost"],
		childPath,
	)
	if err != nil || loadedChild.ID != child.ID {
		t.Fatalf("resource by path = %#v, %v", loadedChild, err)
	}

	duplicatePath := "/child"
	_, err = resourceRepository.Create(ctx, resource.Resource{
		SiteID:       siteIDs["localhost"],
		ParentID:     &root.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Duplicate",
		Slug:         "child",
		Path:         &duplicatePath,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Settings:     map[string]any{},
	})
	if !errors.Is(err, resource.ErrConflict) {
		t.Fatalf("sibling conflict error = %v", err)
	}

	crossSitePath := "/cross-site"
	_, err = resourceRepository.Create(ctx, resource.Resource{
		SiteID:       siteIDs["example.com"],
		ParentID:     &root.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Cross-site",
		Slug:         "cross-site",
		Path:         &crossSitePath,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Settings:     map[string]any{},
	})
	if !errors.Is(err, resource.ErrInvalidReference) {
		t.Fatalf("cross-site parent error = %v", err)
	}

	if _, err := connector.Pool().Exec(ctx, `
INSERT INTO core.resources
(
    site_id,
    title,
    slug,
    settings
)
VALUES ($1, 'Invalid settings', 'invalid-settings', '[]'::jsonb);
`, siteIDs["localhost"]); err == nil {
		t.Fatal("resources accepted non-object settings")
	}

	child.Slug = "renamed"
	child.Title = "Renamed child"
	child.ParentID = &section.ID
	renamedPath := "/section/renamed"
	child.Path = &renamedPath
	child, err = resourceRepository.Update(ctx, child)
	if err != nil {
		t.Fatalf("rename resource: %v", err)
	}
	if child.Path == nil || *child.Path != renamedPath {
		t.Fatalf("renamed child path = %#v", child.Path)
	}
	grandchild, err = resourceRepository.ByID(ctx, grandchild.ID)
	if err != nil {
		t.Fatalf("load moved grandchild: %v", err)
	}
	if grandchild.Path == nil ||
		*grandchild.Path != "/section/renamed/grandchild" {
		t.Fatalf(
			"moved grandchild path = %#v",
			grandchild.Path,
		)
	}

	section.ParentID = &grandchild.ID
	section.Path = testStringPointer(
		"/section/renamed/grandchild/section",
	)
	if _, err := resourceRepository.Update(
		ctx,
		section,
	); !errors.Is(err, resource.ErrInvalidTree) {
		t.Fatalf("resource cycle error = %v", err)
	}

	internalLinkPath := "/section/renamed/internal-link"
	internalLinkTarget := grandchild.ID
	if _, err := resourceRepository.Create(
		ctx,
		resource.Resource{
			SiteID:           siteIDs["localhost"],
			ParentID:         &child.ID,
			Type:             resourcetype.ResourceLink,
			Title:            "Internal link",
			Slug:             "internal-link",
			Path:             &internalLinkPath,
			TargetResourceID: &internalLinkTarget,
			IsPublic:         true,
			IsSearchable:     true,
			InMenu:           true,
			InSitemap:        true,
			Settings:         map[string]any{},
		},
	); err != nil {
		t.Fatalf("create internal resource link: %v", err)
	}

	externalLinkPath := "/external-link"
	externalLinkTarget := grandchild.ID
	externalLink, err := resourceRepository.Create(
		ctx,
		resource.Resource{
			SiteID:           siteIDs["localhost"],
			Type:             resourcetype.ResourceLink,
			Title:            "External link",
			Slug:             "external-link",
			Path:             &externalLinkPath,
			TargetResourceID: &externalLinkTarget,
			IsPublic:         true,
			IsSearchable:     true,
			InMenu:           true,
			InSitemap:        true,
			Settings:         map[string]any{},
		},
	)
	if err != nil {
		t.Fatalf("create external resource link: %v", err)
	}

	if err := resourceRepository.Delete(
		ctx,
		child.ID,
	); !errors.Is(err, resource.ErrReferenced) {
		t.Fatalf("referenced subtree delete error = %v", err)
	}
	if err := resourceRepository.Delete(
		ctx,
		externalLink.ID,
	); err != nil {
		t.Fatalf("delete external resource link: %v", err)
	}
	if err := resourceRepository.Delete(ctx, child.ID); err != nil {
		t.Fatalf("delete resource subtree: %v", err)
	}
	if _, err := resourceRepository.ByID(
		ctx,
		grandchild.ID,
	); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("deleted grandchild error = %v", err)
	}

	if err := database.Sites().UpdateSettings(
		ctx,
		site.ID(1<<62),
		map[string]any{},
	); !errors.Is(err, site.ErrNotFound) {
		t.Fatalf("missing site update error = %v", err)
	}

	if err := seedManager.Down(ctx, seedPlan, 1); err != nil {
		t.Fatalf("seed down: %v", err)
	}
	loadedSites, err = database.Sites().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range loadedSites {
		if item.ProfileCode == "dev" &&
			(item.Domain == "localhost" || item.Domain == "example.com") {
			t.Fatalf("seed down kept site %#v", item)
		}
	}

	restoreMigration = true
	if err := manager.Down(ctx, plan, 3); err != nil {
		t.Fatalf("down: %v", err)
	}

	var schemaName *string
	var historyTable *string
	var devSeedHistoryTable *string
	if err := connector.Pool().QueryRow(ctx, `
SELECT
    to_regnamespace('core')::text,
    to_regclass('core.schema_migrations')::text,
    to_regclass('core.schema_seeds_sites_dev')::text;
`).Scan(
		&schemaName,
		&historyTable,
		&devSeedHistoryTable,
	); err != nil {
		t.Fatal(err)
	}
	if schemaName == nil || *schemaName != "core" {
		t.Fatalf("core schema was removed: %#v", schemaName)
	}
	if historyTable == nil || *historyTable != "core.schema_migrations" {
		t.Fatalf("migration history was removed: %#v", historyTable)
	}
	if devSeedHistoryTable == nil ||
		*devSeedHistoryTable != "core.schema_seeds_sites_dev" {
		t.Fatalf(
			"seed history was removed: %#v",
			devSeedHistoryTable,
		)
	}

	if err := manager.Up(ctx, plan); err != nil {
		t.Fatalf("restore up: %v", err)
	}
	restoreMigration = false
}

func testStringPointer(value string) *string {
	return &value
}
