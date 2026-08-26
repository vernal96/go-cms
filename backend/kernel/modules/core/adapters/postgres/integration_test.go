package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/security"
	"github.com/vernal96/go-cms/kernel/seeds"
)

func TestDevSiteSeedSource(t *testing.T) {
	sources := (&Database{}).SeedSources()
	if len(sources) != 2 {
		t.Fatalf("seed sources = %#v", sources)
	}

	shared := sources[0]
	if shared.ID != "identity_shared" ||
		len(shared.Tags) != 2 ||
		shared.Tags[0] != "dev" ||
		shared.Tags[1] != "prod" ||
		shared.Schema != "core" {
		t.Fatalf("shared identity source = %#v", shared)
	}
	if err := seeds.ValidateSource(shared); err != nil {
		t.Fatal(err)
	}

	source := sources[1]
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
	if len(entries) != 6 {
		t.Fatalf("dev seed files = %#v", entries)
	}
}

func TestMigrationSourceIncludesIdentityAndPermissions(t *testing.T) {
	t.Parallel()

	sources := (&Database{}).MigrationSources()
	if len(sources) != 1 {
		t.Fatalf("migration sources = %#v", sources)
	}
	entries, err := fs.ReadDir(sources[0].FS, sources[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 32 {
		t.Fatalf("migration files = %#v", entries)
	}
	expected := map[string]bool{
		"000016_resource_revisions.up.sql":                   false,
		"000016_resource_revisions.down.sql":                 false,
		"000005_identity.up.sql":                             false,
		"000005_identity.down.sql":                           false,
		"000006_permissions.up.sql":                          false,
		"000006_permissions.down.sql":                        false,
		"000007_resource_widgets.up.sql":                     false,
		"000007_resource_widgets.down.sql":                   false,
		"000008_user_blocking.up.sql":                        false,
		"000008_user_blocking.down.sql":                      false,
		"000009_file_field_references.up.sql":                false,
		"000009_file_field_references.down.sql":              false,
		"000010_user_preferences.up.sql":                     false,
		"000010_user_preferences.down.sql":                   false,
		"000011_user_accent_color.up.sql":                    false,
		"000011_user_accent_color.down.sql":                  false,
		"000012_resource_editor_tree.up.sql":                 false,
		"000012_resource_editor_tree.down.sql":               false,
		"000013_resource_widget_bindings.up.sql":             false,
		"000013_resource_widget_bindings.down.sql":           false,
		"000014_group_site_access.up.sql":                    false,
		"000014_group_site_access.down.sql":                  false,
		"000015_resource_entities_fields_libraries.up.sql":   false,
		"000015_resource_entities_fields_libraries.down.sql": false,
	}
	for _, entry := range entries {
		if _, exists := expected[entry.Name()]; exists {
			expected[entry.Name()] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("migration %q is not embedded", name)
		}
	}
	accentMigration, err := fs.ReadFile(
		sources[0].FS,
		sources[0].Path+"/000011_user_accent_color.up.sql",
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, expectedSQL := range []string{
		"DEFAULT 'blue'",
		"'blue', 'violet', 'indigo', 'emerald', 'amber', 'rose'",
	} {
		if !strings.Contains(string(accentMigration), expectedSQL) {
			t.Fatalf("accent migration does not contain %q", expectedSQL)
		}
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
	if version != 16 || !hasVersion || dirty {
		t.Fatalf(
			"version = %d, hasVersion = %t, dirty = %t",
			version,
			hasVersion,
			dirty,
		)
	}

	var sitesTable *string
	var resourcesTable *string
	var resourceWidgetsTable *string
	var resourcePathIndex *string
	var fileFoldersTable *string
	var filesTable *string
	var mediaTable *string
	if err := connector.Pool().QueryRow(
		ctx,
		`
SELECT
    to_regclass('core.sites')::text,
    to_regclass('core.resources')::text,
    to_regclass('core.resource_widgets')::text,
    to_regclass('core.uq_resources_site_path')::text,
    to_regclass('core.file_folders')::text,
    to_regclass('core.files')::text,
    to_regclass('core.media')::text;
`,
	).Scan(
		&sitesTable,
		&resourcesTable,
		&resourceWidgetsTable,
		&resourcePathIndex,
		&fileFoldersTable,
		&filesTable,
		&mediaTable,
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
	if resourceWidgetsTable == nil ||
		*resourceWidgetsTable != "core.resource_widgets" {
		t.Fatalf("core.resource_widgets = %#v", resourceWidgetsTable)
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
	if mediaTable == nil || *mediaTable != "core.media" {
		t.Fatalf("core.media = %#v", mediaTable)
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

	imageChecksum := sha256.Sum256([]byte("image"))
	imageFile, err := fileRepository.CreateFile(ctx, corefile.File{
		Storage:        "public",
		Name:           "image.png",
		MIMEType:       "image/png",
		Size:           5,
		ChecksumSHA256: hex.EncodeToString(imageChecksum[:]),
		Path:           "objects/image",
	})
	if err != nil {
		t.Fatalf("create image file: %v", err)
	}
	replacementChecksum := sha256.Sum256([]byte("replacement"))
	replacementFile, err := fileRepository.CreateFile(ctx, corefile.File{
		Storage:        "public",
		Name:           "replacement.webp",
		MIMEType:       "image/webp",
		Size:           11,
		ChecksumSHA256: hex.EncodeToString(replacementChecksum[:]),
		Path:           "objects/replacement",
	})
	if err != nil {
		t.Fatalf("create replacement image file: %v", err)
	}
	documentChecksum := sha256.Sum256([]byte("document"))
	documentFile, err := fileRepository.CreateFile(ctx, corefile.File{
		Storage:        "private",
		Name:           "document.pdf",
		MIMEType:       "application/pdf",
		Size:           8,
		ChecksumSHA256: hex.EncodeToString(documentChecksum[:]),
		Path:           "objects/document",
	})
	if err != nil {
		t.Fatalf("create document file: %v", err)
	}

	mediaRepository := database.Media()
	mediaTitle := "Hero"
	imageMedia, err := mediaRepository.Create(ctx, nil, media.Media{
		FileID: imageFile.ID,
		Title:  &mediaTitle,
		Params: map[string]any{
			"meta_alt": "Hero",
		},
	})
	if err != nil {
		t.Fatalf("create image media: %v", err)
	}
	replacementMedia, err := mediaRepository.Create(ctx, nil, media.Media{
		FileID: replacementFile.ID,
		Params: map[string]any{},
	})
	if err != nil {
		t.Fatalf("create replacement media: %v", err)
	}
	documentMedia, err := mediaRepository.Create(ctx, nil, media.Media{
		FileID: documentFile.ID,
		Params: map[string]any{},
	})
	if err != nil {
		t.Fatalf("create document media: %v", err)
	}
	concurrentMedia, err := mediaRepository.Create(ctx, nil, media.Media{
		FileID: imageFile.ID,
		Params: map[string]any{},
	})
	if err != nil {
		t.Fatalf("create concurrent media: %v", err)
	}
	sharedMedia, err := mediaRepository.Create(ctx, nil, media.Media{
		FileID: imageFile.ID,
		Params: map[string]any{},
	})
	if err != nil {
		t.Fatalf("create shared media: %v", err)
	}

	if _, err := mediaRepository.Create(ctx, nil, media.Media{
		FileID: corefile.ID(1 << 62),
		Params: map[string]any{},
	}); !errors.Is(err, media.ErrInvalidReference) {
		t.Fatalf("missing media file error = %v", err)
	}
	if _, err := connector.Pool().Exec(ctx, `
INSERT INTO core.media (file_id, params)
VALUES ($1, '[]'::jsonb);
`, imageFile.ID); err == nil {
		t.Fatal("media accepted non-object params")
	}

	sharedSeedPlan := seeds.Plan{
		Connection: string(connector.Code()),
		Module:     core.ModuleCode,
		Target:     connector,
		Source:     database.SeedSources()[0],
	}
	devSeedPlan := seeds.Plan{
		Connection: string(connector.Code()),
		Module:     core.ModuleCode,
		Target:     connector,
		Source:     database.SeedSources()[1],
	}
	seedManager := seeds.NewManager()
	for _, seedPlan := range []seeds.Plan{
		sharedSeedPlan,
		devSeedPlan,
	} {
		if err := seedManager.Force(ctx, seedPlan, -1); err != nil {
			t.Fatalf("prepare seed state: %v", err)
		}
	}
	if _, err := connector.Pool().Exec(ctx, `
DELETE
FROM core.sites
WHERE profile_code = 'dev'
  AND domain IN ('localhost', 'example.com');

DELETE FROM core.users
WHERE login = 'admin'
  AND email = 'admin@example.test';

DELETE FROM core.groups
WHERE code IN ('admin', 'manager');
`); err != nil {
		t.Fatalf("clean seed targets: %v", err)
	}
	if err := seedManager.Up(ctx, sharedSeedPlan); err != nil {
		t.Fatalf("shared seed up: %v", err)
	}
	if err := seedManager.Up(ctx, devSeedPlan); err != nil {
		t.Fatalf("dev seed up: %v", err)
	}

	var (
		adminMemberships int64
		managerSuper     bool
		groupGrants      int64
		guestGrants      int64
	)
	if err := connector.Pool().QueryRow(ctx, `
SELECT
    (
        SELECT count(*)
        FROM core.users u
        JOIN core.user_groups ug ON ug.user_id = u.id
        JOIN core.groups g ON g.id = ug.group_id
        WHERE u.login = 'admin'
          AND u.email = 'admin@example.test'
          AND u.blocked_at IS NULL
          AND g.code = 'admin'
          AND g.is_super
    ),
    (
        SELECT is_super
        FROM core.groups
        WHERE code = 'manager'
    ),
    (SELECT count(*) FROM core.group_permissions),
    (SELECT count(*) FROM core.guest_permissions);
`).Scan(
		&adminMemberships,
		&managerSuper,
		&groupGrants,
		&guestGrants,
	); err != nil {
		t.Fatalf("query identity seed: %v", err)
	}
	if adminMemberships != 1 ||
		managerSuper ||
		groupGrants != 0 ||
		guestGrants != 0 {
		t.Fatalf(
			"identity seed = memberships:%d manager_super:%t group_grants:%d guest_grants:%d",
			adminMemberships,
			managerSuper,
			groupGrants,
			guestGrants,
		)
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
		if item.ProfileCode != "dev" ||
			item.Locale != "ru-RU" ||
			!item.IsPublic {
			t.Fatalf("seeded site = %#v", item)
		}
		rawSettings, err := json.Marshal(item.Settings)
		if err != nil {
			t.Fatal(err)
		}
		expectedSettings := `{"checkbox_value":false,"email_value":"admin@example.test","float_value":1.5,"integer_value":10,"multi_select_value":["alpha","beta"],"phone_value":"+79991234567","radio_value":"first","select_value":"alpha","string_value":` + strconv.Quote(item.Domain) + `,"textarea_value":"Демонстрационные настройки сайта"}`
		if string(rawSettings) != expectedSettings {
			t.Fatalf("settings = %s", rawSettings)
		}
	}
	if !found["localhost"] || !found["example.com"] {
		t.Fatalf("seeded domains = %#v", found)
	}

	siteManagement, ok := database.Sites().(site.ManagementRepository)
	if !ok {
		t.Fatal("site management repository is unavailable")
	}
	page, err := siteManagement.ListPage(ctx, site.ListQuery{
		Search:  "LOCAL",
		Page:    1,
		PerPage: 10,
		Scope:   site.Scope{SiteIDs: []site.ID{siteIDs["localhost"]}},
	})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].Domain != "localhost" {
		t.Fatalf("site page = %#v, %v", page, err)
	}
	createdSite, err := siteManagement.Create(ctx, nil, site.Site{
		ProfileCode: "dev",
		Domain:      "admin-management.test",
		Locale:      "ru-RU",
		Settings:    map[string]any{},
	})
	if err != nil {
		t.Fatalf("create managed site: %v", err)
	}
	createdSite.Domain = "renamed-management.test"
	createdSite.IsPublic = true
	createdSite, err = siteManagement.Update(ctx, nil, createdSite)
	if err != nil || createdSite.Domain != "renamed-management.test" || !createdSite.IsPublic {
		t.Fatalf("updated managed site = %#v, %v", createdSite, err)
	}
	if foundSite, err := siteManagement.FindByDomain(ctx, createdSite.Domain); err != nil || foundSite.ID != createdSite.ID {
		t.Fatalf("find managed site = %#v, %v", foundSite, err)
	}
	if err := siteManagement.Delete(ctx, createdSite.ID); err != nil {
		t.Fatalf("delete managed site: %v", err)
	}

	if _, err := connector.Pool().Exec(ctx, `
UPDATE core.sites
SET updated_at = '2000-01-01T00:00:00Z'
WHERE id = $1;
`, siteIDs["localhost"]); err != nil {
		t.Fatalf("prepare site update timestamp: %v", err)
	}
	if _, err := database.Sites().Update(
		ctx,
		nil,
		site.Site{
			ID:          siteIDs["localhost"],
			ProfileCode: "dev",
			Domain:      "localhost",
			Locale:      "ru-RU",
			IsPublic:    true,
			Settings: map[string]any{
				"count": int64(3),
				"flag":  false,
			},
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
	validateImageMedia := func(
		ctx context.Context,
		id media.ID,
	) error {
		item, err := mediaRepository.ByID(ctx, id)
		if err != nil {
			return err
		}
		linkedFile, err := fileRepository.FileByID(ctx, item.FileID)
		if err != nil {
			return err
		}
		return resource.ValidateImageMediaFile(
			ctx,
			linkedFile,
			media.Usage{
				Kind: resource.ImageMediaUsage,
			},
		)
	}
	templateCode := template.Code("article")
	contentType := "html"
	rootPath := "/"
	var adminID security.UserID
	if err := connector.Pool().QueryRow(ctx, `SELECT id FROM core.users WHERE login='admin';`).Scan(&adminID); err != nil {
		t.Fatalf("load revision author: %v", err)
	}
	root, err := resourceRepository.Create(ctx, &adminID, resource.Resource{
		SiteID:       siteIDs["localhost"],
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Home",
		Path:         &rootPath,
		ImageMediaID: &imageMedia.ID,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Fields:       map[string]any{"headline": "Home"},
	}, validateImageMedia)

	if err != nil {
		t.Fatalf("create root resource: %v", err)
	}
	revisionRepository, ok := resourceRepository.(resource.RevisionRepository)
	if !ok {
		t.Fatal("resource revision repository is unavailable")
	}
	rootHistory, err := revisionRepository.ListRevisions(ctx, root.SiteID, root.ID, 1, 20)
	if err != nil || root.Version != 1 || rootHistory.Total != 1 || rootHistory.Items[0].Kind != resource.RevisionCreated {
		t.Fatalf("created resource history = %#v, resource = %#v, err = %v", rootHistory, root, err)
	}
	widgetRepository, ok := resourceRepository.(resource.WidgetRepository)
	if !ok {
		t.Fatal("resource widget repository is unavailable")
	}
	presentation := widget.DefaultPresentation()
	firstWidget, err := widgetRepository.CreateWidget(ctx, &adminID, root.ID, 1, widget.Binding{
		Code: "content_summary", Area: widget.AreaBody, Position: 0,
		Presentation: presentation, Params: map[string]any{"title": "Primary"},
	}, true)
	if err != nil {
		t.Fatalf("create first root widget: %v", err)
	}
	secondWidget, err := widgetRepository.CreateWidget(ctx, &adminID, root.ID, 2, widget.Binding{
		Code: "content_summary", Area: widget.AreaBody, Position: 1,
		Presentation: presentation, Params: map[string]any{"title": "Secondary"},
	}, true)
	if err != nil || firstWidget.ID <= 0 || secondWidget.ID <= 0 {
		t.Fatalf("created root widgets = %#v / %#v, %v", firstWidget, secondWidget, err)
	}
	loadedRoot, err := resourceRepository.ByID(ctx, root.ID)
	if err != nil || len(loadedRoot.Widgets) != 2 ||
		loadedRoot.Widgets[0].Code != "content_summary" {
		t.Fatalf("loaded root widgets = %#v, %v", loadedRoot.Widgets, err)
	}
	reordered, err := widgetRepository.ReorderWidgets(ctx, &adminID, root.ID, 3, []widget.Order{
		{ID: secondWidget.ID, Area: widget.AreaBody, Position: 0},
		{ID: firstWidget.ID, Area: widget.AreaSidebar, Position: 0},
	}, true)
	if err != nil || len(reordered) != 2 || reordered[0].ID != secondWidget.ID ||
		reordered[1].ID != firstWidget.ID || reordered[1].Area != widget.AreaSidebar {
		t.Fatalf("reordered root widgets = %#v, %v", reordered, err)
	}
	rootHistory, err = revisionRepository.ListRevisions(ctx, root.SiteID, root.ID, 1, 20)
	latest, detailErr := revisionRepository.Revision(ctx, root.SiteID, root.ID, 4)
	createdRevision, createdDetailErr := revisionRepository.Revision(ctx, root.SiteID, root.ID, 1)
	if err != nil || detailErr != nil || rootHistory.Total != 4 || latest.Snapshot == nil ||
		latest.CreatedBy == nil || *latest.CreatedBy != adminID || latest.CreatedByName != "Администратор" ||
		createdDetailErr != nil || createdRevision.Snapshot == nil || createdRevision.Snapshot.Fields["headline"] != "Home" ||
		len(latest.Snapshot.Widgets) != 2 {
		t.Fatalf("widget revision history = %#v, latest = %#v, created = %#v, errors = %v / %v / %v", rootHistory, latest, createdRevision, err, detailErr, createdDetailErr)
	}
	if _, err := widgetRepository.UpdateWidget(ctx, &adminID, root.ID, 3, firstWidget, true); !errors.Is(err, resource.ErrConflict) {
		t.Fatalf("stale widget update error = %v", err)
	}
	rootHistory, err = revisionRepository.ListRevisions(ctx, root.SiteID, root.ID, 1, 20)
	loadedAfterConflict, loadErr := resourceRepository.ByID(ctx, root.ID)
	if err != nil || loadErr != nil || rootHistory.Total != 4 || loadedAfterConflict.Version != 4 {
		t.Fatalf("state after stale widget update = history %#v, resource %#v, errors = %v / %v", rootHistory, loadedAfterConflict, err, loadErr)
	}
	root.Version = 4
	firstWidget.Area = widget.AreaSidebar
	firstWidget.Position = 0
	secondWidget.Position = 0

	documentRootPath := "/"
	if _, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID:       siteIDs["example.com"],
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Invalid image",
		Path:         &documentRootPath,
		ImageMediaID: &documentMedia.ID,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Fields:       map[string]any{"headline": "Invalid"},
	}, validateImageMedia); !errors.Is(err, resource.ErrInvalidReference) {
		t.Fatalf("non-image resource media error = %v", err)
	}

	duplicateImagePath := "/duplicate-image"
	if _, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID:       siteIDs["localhost"],
		ParentID:     &root.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  &contentType,
		Title:        "Duplicate image",
		Slug:         "duplicate-image",
		Path:         &duplicateImagePath,
		ImageMediaID: &imageMedia.ID,
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
		Fields:       map[string]any{"headline": "Duplicate"},
	}, validateImageMedia); !errors.Is(err, media.ErrAlreadyAttached) {
		t.Fatalf("duplicate media attachment error = %v", err)
	}

	nextRoot := resource.Clone(root)
	nextRoot.ImageMediaID = &replacementMedia.ID
	nextRoot.Widgets = reordered
	root, err = resourceRepository.Update(
		ctx,
		nil,
		root,
		nextRoot,
		validateImageMedia,
	)
	if err != nil {
		t.Fatalf("replace root media: %v", err)
	}
	firstWidget.Params = map[string]any{"title": "Replacement"}
	firstWidget.Presentation.Columns = 8
	firstWidget, err = widgetRepository.UpdateWidget(ctx, nil, root.ID, root.Version, firstWidget, true)
	if err != nil || firstWidget.ID <= 0 {
		t.Fatalf("update root widget: %#v, %v", firstWidget, err)
	}
	if err := widgetRepository.DeleteWidget(ctx, nil, root.ID, root.Version+1, secondWidget.ID, true); err != nil {
		t.Fatalf("delete root widget: %v", err)
	}
	loadedRoot, err = resourceRepository.ByID(ctx, root.ID)
	if err != nil || len(loadedRoot.Widgets) != 1 ||
		loadedRoot.Widgets[0].Position != 0 {
		t.Fatalf("reloaded root widgets = %#v, %v", loadedRoot.Widgets, err)
	}
	listedResources, err := resourceRepository.ListBySite(ctx, root.SiteID)
	if err != nil {
		t.Fatal(err)
	}
	listedRootFound := false
	for _, listed := range listedResources {
		if listed.ID != root.ID {
			continue
		}
		listedRootFound = true
		if len(listed.Widgets) != 1 ||
			listed.Widgets[0].Params["title"] != "Replacement" {
			t.Fatalf("listed root widgets = %#v", listed.Widgets)
		}
	}
	if !listedRootFound {
		t.Fatal("root resource missing from list")
	}
	resourceManagement, ok := resourceRepository.(resource.ManagementRepository)
	if !ok {
		t.Fatal("resource management repository is unavailable")
	}
	roots, err := resourceManagement.ListChildren(ctx, root.SiteID, nil)
	if err != nil {
		t.Fatalf("list root resources: %v", err)
	}
	rootFound := false
	for _, current := range roots {
		if current.ID == root.ID {
			rootFound = true
		}
	}
	if !rootFound {
		t.Fatal("root missing from lazy resource list")
	}
	widgetChildPath := "/widget-child"
	widgetChild, err := resourceRepository.Create(
		ctx,
		nil,
		resource.Resource{
			SiteID:       root.SiteID,
			ParentID:     &root.ID,
			Type:         resourcetype.Page,
			Title:        "Widget child",
			Slug:         "widget-child",
			Path:         &widgetChildPath,
			Annotation:   "Widget annotation",
			IsPublic:     true,
			IsSearchable: true,
			InMenu:       true,
			InSitemap:    true,
			Fields:       map[string]any{},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("create widget child: %v", err)
	}
	if _, err := widgetRepository.CreateWidget(ctx, nil, widgetChild.ID, widgetChild.Version, widget.Binding{
		Code: "content_summary", Area: widget.AreaSidebar, Position: 0,
		Presentation: presentation, Params: map[string]any{},
	}, true); err != nil {
		t.Fatalf("create child widget: %v", err)
	}
	children, err := resourceManagement.ListChildren(ctx, root.SiteID, &root.ID)
	if err != nil || len(children) == 0 {
		t.Fatalf("lazy resource children = %#v, %v", children, err)
	}
	if exists, err := resourceManagement.ExistsInSite(ctx, root.SiteID, root.ID); err != nil || !exists {
		t.Fatalf("resource site membership = %v, %v", exists, err)
	}
	wrongSiteID := root.SiteID + 1000
	if _, err := resourceManagement.ListChildren(ctx, wrongSiteID, &root.ID); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("wrong-site lazy parent error = %v", err)
	}
	childFound := false
	for _, child := range children {
		if child.ID == widgetChild.ID {
			childFound = true
		}
		if child.ParentID == nil || *child.ParentID != root.ID || child.SiteID != root.SiteID {
			t.Fatalf("lazy resource child = %#v", child)
		}
	}
	if !childFound {
		t.Fatal("created child missing from lazy resource list")
	}
	lifecycle, ok := resourceRepository.(resource.LifecycleRepository)
	if !ok {
		t.Fatal("resource lifecycle repository is unavailable")
	}
	if err := lifecycle.SoftDelete(ctx, nil, widgetChild.ID); err != nil {
		t.Fatalf("soft delete resource child: %v", err)
	}
	deletedChild, err := resourceRepository.ByID(ctx, widgetChild.ID)
	if err != nil || deletedChild.DeletedAt == nil || deletedChild.Annotation != "Widget annotation" {
		t.Fatalf("soft-deleted resource = %#v, %v", deletedChild, err)
	}
	if err := lifecycle.Restore(ctx, nil, widgetChild.ID, false); err != nil {
		t.Fatalf("restore resource child: %v", err)
	}
	restoredChild, err := resourceRepository.ByID(ctx, widgetChild.ID)
	if err != nil || restoredChild.DeletedAt != nil {
		t.Fatalf("restored resource = %#v, %v", restoredChild, err)
	}
	siblingPath := "/sibling"
	sibling, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID: root.SiteID, ParentID: &root.ID, Type: resourcetype.Page,
		Title: "Sibling", Slug: "sibling", Path: &siblingPath,
		IsPublic: true, IsSearchable: true, InMenu: true, InSitemap: true,
		Fields: map[string]any{},
	}, nil)
	if err != nil {
		t.Fatalf("create resource sibling: %v", err)
	}
	nextSibling := resource.Clone(sibling)
	nextSibling.Sort = 0
	if _, err := resourceRepository.Update(ctx, nil, sibling, nextSibling, nil); err != nil {
		t.Fatalf("move resource sibling: %v", err)
	}
	children, err = resourceManagement.ListChildren(ctx, root.SiteID, &root.ID)
	if err != nil || len(children) < 2 || children[0].ID != sibling.ID || children[0].Sort != 0 || children[1].Sort != 1 {
		t.Fatalf("reordered resource children = %#v, %v", children, err)
	}
	if err := resourceRepository.Delete(ctx, sibling.ID); err != nil {
		t.Fatalf("delete resource sibling: %v", err)
	}
	roots, err = resourceManagement.ListChildren(ctx, root.SiteID, nil)
	if err != nil {
		t.Fatalf("reload root resources: %v", err)
	}
	for _, current := range roots {
		if current.ID == root.ID && !current.HasChildren {
			t.Fatal("root has_children was not calculated with EXISTS")
		}
	}
	if err := resourceRepository.Delete(ctx, widgetChild.ID); err != nil {
		t.Fatalf("delete widget child: %v", err)
	}
	var widgetRows int
	if err := connector.Pool().QueryRow(ctx, `
SELECT count(*)
FROM core.resource_widgets
WHERE resource_id = $1;
`, widgetChild.ID).Scan(&widgetRows); err != nil {
		t.Fatal(err)
	}
	if widgetRows != 0 {
		t.Fatalf("widget rows after resource delete = %d", widgetRows)
	}
	if _, err := mediaRepository.ByID(
		ctx,
		imageMedia.ID,
	); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("old media after replacement = %v", err)
	}

	if _, err := mediaRepository.Update(
		ctx,
		nil,
		media.Media{
			ID:     replacementMedia.ID,
			FileID: documentFile.ID,
			Params: map[string]any{},
		},
		func(
			ctx context.Context,
			usages []media.Usage,
		) error {
			for _, usage := range usages {
				if err := resource.ValidateImageMediaFile(
					ctx,
					documentFile,
					usage,
				); err != nil {
					return err
				}
			}
			return nil
		},
	); !errors.Is(err, resource.ErrInvalidReference) {
		t.Fatalf("replace attached media with document error = %v", err)
	}

	replacementMedia, err = mediaRepository.Update(
		ctx,
		nil,
		media.Media{
			ID:     replacementMedia.ID,
			FileID: imageFile.ID,
			Params: map[string]any{"meta_alt": "Updated"},
		},
		func(
			ctx context.Context,
			usages []media.Usage,
		) error {
			for _, usage := range usages {
				if err := resource.ValidateImageMediaFile(
					ctx,
					imageFile,
					usage,
				); err != nil {
					return err
				}
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("replace attached media file: %v", err)
	}

	for _, slug := range []string{"no-path-one", "no-path-two"} {
		noPath, err := resourceRepository.Create(
			ctx,
			nil,
			resource.Resource{
				SiteID:       siteIDs["localhost"],
				Type:         "no_path",
				Title:        slug,
				Slug:         slug,
				IsPublic:     true,
				IsSearchable: true,
				InMenu:       true,
				InSitemap:    true,
				Fields:       map[string]any{},
			}, nil)

		if err != nil {
			t.Fatalf("create nullable-path resource: %v", err)
		}
		if noPath.Path != nil {
			t.Fatalf("nullable resource path = %#v", noPath.Path)
		}
	}

	type attachResult struct {
		item resource.Resource
		err  error
	}
	attachResults := make(chan attachResult, 2)
	startAttach := make(chan struct{})
	var attachWait sync.WaitGroup
	for index, slug := range []string{
		"concurrent-image-one",
		"concurrent-image-two",
	} {
		attachWait.Add(1)
		go func(index int, slug string) {
			defer attachWait.Done()
			<-startAttach
			path := "/" + slug
			item, err := resourceRepository.Create(
				ctx,
				nil,
				resource.Resource{
					SiteID:       siteIDs["localhost"],
					ParentID:     &root.ID,
					Type:         resourcetype.Page,
					Template:     &templateCode,
					ContentType:  &contentType,
					Title:        "Concurrent " + strconv.Itoa(index),
					Slug:         slug,
					Path:         &path,
					ImageMediaID: &concurrentMedia.ID,
					IsPublic:     true,
					IsSearchable: true,
					InMenu:       true,
					InSitemap:    true,
					Fields: map[string]any{
						"headline": "Concurrent",
					},
				},
				validateImageMedia,
			)
			attachResults <- attachResult{item: item, err: err}
		}(index, slug)
	}
	close(startAttach)
	attachWait.Wait()
	close(attachResults)

	attached := 0
	conflicted := 0
	for result := range attachResults {
		switch {
		case result.err == nil:
			attached++
		case errors.Is(result.err, media.ErrAlreadyAttached):
			conflicted++
		default:
			t.Fatalf("concurrent media attachment error = %v", result.err)
		}
	}
	if attached != 1 || conflicted != 1 {
		t.Fatalf(
			"concurrent media attachments = success:%d conflict:%d",
			attached,
			conflicted,
		)
	}

	sectionPath := "/section"
	section, err := resourceRepository.Create(
		ctx,
		nil,
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
			Fields:       map[string]any{},
		}, nil)

	if err != nil {
		t.Fatalf("create section resource: %v", err)
	}

	childPath := "/child"
	child, err := resourceRepository.Create(ctx, nil, resource.Resource{
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
		Fields:       map[string]any{},
	}, nil)

	if err != nil {
		t.Fatalf("create child resource: %v", err)
	}

	grandchildPath := "/child/grandchild"
	grandchild, err := resourceRepository.Create(
		ctx,
		nil,
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
			Fields:       map[string]any{},
		}, nil)

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
	_, err = resourceRepository.Create(ctx, nil, resource.Resource{
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
		Fields:       map[string]any{},
	}, nil)

	if !errors.Is(err, resource.ErrConflict) {
		t.Fatalf("sibling conflict error = %v", err)
	}

	crossSitePath := "/cross-site"
	_, err = resourceRepository.Create(ctx, nil, resource.Resource{
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
		Fields:       map[string]any{},
	}, nil)

	if !errors.Is(err, resource.ErrInvalidReference) {
		t.Fatalf("cross-site parent error = %v", err)
	}

	if _, err := connector.Pool().Exec(ctx, `
WITH entity AS (
    INSERT INTO core.resource_entities (site_id, storage_kind)
    VALUES ($1, 'tree')
    RETURNING id
)
INSERT INTO core.resources
(
    id,
    site_id,
    title,
    slug,
    type_settings
)
VALUES ((SELECT id FROM entity), $1, 'Invalid settings', 'invalid-settings', '[]'::jsonb);
`, siteIDs["localhost"]); err == nil {
		t.Fatal("resources accepted non-object type settings")
	}

	queryRepository, ok := resourceRepository.(resource.QueryRepository)
	if !ok {
		t.Fatal("resource query repository is unavailable")
	}
	typedLowPath := "/typed-low"
	typedLow, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID: siteIDs["localhost"], Type: resourcetype.Page,
		Title: "Typed low", Slug: "typed-low", Path: &typedLowPath,
		IsPublic: true, IsSearchable: true,
		Fields: map[string]any{
			"salary": int64(9), "city": "Moscow", "remote": false,
			"attachment": int64(11), "roles": []string{"editor", "author"},
		},
		FieldValues: []field.StoredValue{
			{Key: "salary", Kind: field.StorageInteger, Value: int64(9)},
			{Key: "city", Kind: field.StorageString, Value: "Moscow"},
			{Key: "remote", Kind: field.StorageBoolean, Value: false},
			{Key: "attachment", Kind: field.StorageReference, Value: int64(11)},
			{Key: "roles", Position: 0, Kind: field.StorageString, Multiple: true, Value: "editor"},
			{Key: "roles", Position: 1, Kind: field.StorageString, Multiple: true, Value: "author"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("create low typed resource: %v", err)
	}
	typedHighPath := "/typed-high"
	typedHigh, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID: siteIDs["localhost"], Type: resourcetype.Page,
		Title: "Typed high", Slug: "typed-high", Path: &typedHighPath,
		IsPublic: true, IsSearchable: true,
		Fields: map[string]any{
			"salary": int64(150000), "city": "Moscow", "remote": true,
			"attachment": int64(22),
		},
		FieldValues: []field.StoredValue{
			{Key: "salary", Kind: field.StorageInteger, Value: int64(150000)},
			{Key: "city", Kind: field.StorageString, Value: "Moscow"},
			{Key: "remote", Kind: field.StorageBoolean, Value: true},
			{Key: "attachment", Kind: field.StorageReference, Value: int64(22)},
		},
	}, nil)
	if err != nil {
		t.Fatalf("create high typed resource: %v", err)
	}
	loadedTypedLow, err := resourceRepository.ByID(ctx, typedLow.ID)
	if err != nil || !reflect.DeepEqual(loadedTypedLow.Fields["roles"], []string{"editor", "author"}) {
		t.Fatalf("loaded ordered multi-value fields = %#v, %v", loadedTypedLow.Fields, err)
	}
	for _, test := range []struct {
		name      string
		condition resource.FilterCondition
		wantID    resource.ID
	}{
		{name: "numeric", condition: resource.FilterCondition{Field: "resource.field.salary", Operator: resource.FilterGreaterThanOrEqual, Value: int64(100000), Kind: field.StorageInteger}, wantID: typedHigh.ID},
		{name: "boolean", condition: resource.FilterCondition{Field: "resource.field.remote", Operator: resource.FilterEqual, Value: true, Kind: field.StorageBoolean}, wantID: typedHigh.ID},
		{name: "reference", condition: resource.FilterCondition{Field: "resource.field.attachment", Operator: resource.FilterEqual, Value: int64(22), Kind: field.StorageReference}, wantID: typedHigh.ID},
	} {
		page, queryErr := queryRepository.Query(ctx, resource.Query{
			SiteID: siteIDs["localhost"], Limit: 100, Filters: []resource.FilterCondition{test.condition},
		})
		if queryErr != nil || len(page.Items) != 1 || page.Items[0].ID != test.wantID {
			t.Fatalf("%s typed filter = %#v, %v", test.name, page, queryErr)
		}
	}
	cityPage, err := queryRepository.Query(ctx, resource.Query{
		SiteID: siteIDs["localhost"], Limit: 100,
		Filters: []resource.FilterCondition{{Field: "resource.field.city", Operator: resource.FilterEqual, Value: "Moscow", Kind: field.StorageString}},
	})
	if err != nil || len(cityPage.Items) != 2 {
		t.Fatalf("string typed filter = %#v, %v", cityPage, err)
	}
	sortedPage, err := queryRepository.Query(ctx, resource.Query{
		SiteID: siteIDs["localhost"], Limit: 100,
		Sort: []resource.Sort{{Field: "resource.field.salary", Direction: resource.SortDescending, Kind: field.StorageInteger}},
	})
	if err != nil || len(sortedPage.Items) < 2 || sortedPage.Items[0].ID != typedHigh.ID || sortedPage.Items[1].ID != typedLow.ID {
		t.Fatalf("numeric typed sort = %#v, %v", sortedPage, err)
	}
	staleFree := typedLow
	staleFree.Fields = map[string]any{"city": "Moscow"}
	staleFree.FieldValues = []field.StoredValue{{Key: "city", Kind: field.StorageString, Value: "Moscow"}}
	staleFree, err = resourceRepository.Update(ctx, nil, typedLow, staleFree, nil)
	if err != nil {
		t.Fatalf("remove stale typed values: %v", err)
	}
	if _, exists := staleFree.Fields["salary"]; exists {
		t.Fatalf("stale salary field survived update: %#v", staleFree.Fields)
	}

	libraryItems, ok := resourceRepository.(resource.LibraryItemRepository)
	if !ok {
		t.Fatal("library item repository is unavailable")
	}
	libraryPath := "/catalog"
	library, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID: siteIDs["localhost"], Type: resourcetype.Library,
		Title: "Catalog", Slug: "catalog", Path: &libraryPath,
		IsPublic: true, IsSearchable: true, InMenu: true, InSitemap: true,
		Fields: map[string]any{}, TypeSettings: map[string]any{
			"item_url_pattern": "/{year}/{slug}",
		},
	}, nil)
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	archivePath := "/archive"
	archive, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID: siteIDs["localhost"], Type: resourcetype.Library,
		Title: "Archive", Slug: "archive", Path: &archivePath,
		IsPublic: true, IsSearchable: true, InMenu: true, InSitemap: true,
		Fields: map[string]any{}, TypeSettings: map[string]any{
			"item_url_pattern": "/{slug}",
		},
	}, nil)
	if err != nil {
		t.Fatalf("create target library: %v", err)
	}
	oldPublication := time.Date(1900, time.January, 2, 3, 4, 5, 0, time.UTC)
	createdLibraryItem, err := libraryItems.CreateLibraryItem(ctx, nil, resource.LibraryItem{
		SiteID: siteIDs["localhost"], LibraryID: library.ID,
		Title: "Typed item", Slug: "typed-item", ContentType: &contentType,
		IsPublic: true, IsSearchable: true, PublishedAt: &oldPublication,
		Fields: map[string]any{"headline": "Stored headline"},
		FieldValues: []field.StoredValue{{
			Key: "headline", Kind: field.StorageString, Value: "Stored headline",
		}},
	}, false)
	if err != nil {
		t.Fatalf("create partitioned library item: %v", err)
	}
	loadedLibraryItem, err := libraryItems.LibraryItemByID(ctx, createdLibraryItem.ID)
	if err != nil || loadedLibraryItem.Fields["headline"] != "Stored headline" {
		t.Fatalf("loaded library item = %#v, %v", loadedLibraryItem, err)
	}
	itemWidget, err := widgetRepository.CreateWidget(ctx, nil, createdLibraryItem.ID, createdLibraryItem.Version, widget.Binding{
		Code: "content_summary", Area: widget.AreaBody, Position: 0,
		Presentation: presentation, Params: map[string]any{"title": "Library item"},
	}, false)
	if err != nil || itemWidget.ID <= 0 {
		t.Fatalf("create library item widget = %#v, %v", itemWidget, err)
	}
	loadedLibraryItem, err = libraryItems.LibraryItemByID(ctx, createdLibraryItem.ID)
	if err != nil || len(loadedLibraryItem.Widgets) != 1 || loadedLibraryItem.Widgets[0].ID != itemWidget.ID {
		t.Fatalf("loaded library item widgets = %#v, %v", loadedLibraryItem.Widgets, err)
	}
	futurePublication := time.Date(2040, time.December, 31, 23, 59, 0, 0, time.UTC)
	currentLibraryItem := loadedLibraryItem
	loadedLibraryItem.Title = "Moved item"
	loadedLibraryItem.PublishedAt = &futurePublication
	loadedLibraryItem, err = libraryItems.UpdateLibraryItem(
		ctx, nil, currentLibraryItem, loadedLibraryItem, false,
	)
	if err != nil || loadedLibraryItem.Title != "Moved item" {
		t.Fatalf("update across time partitions = %#v, %v", loadedLibraryItem, err)
	}
	routedItem, routedLibrary, err := libraryItems.ResolveLibraryItemRoute(
		ctx, siteIDs["localhost"], "/catalog/2040/typed-item",
	)
	if err != nil || routedItem.ID != loadedLibraryItem.ID || routedLibrary.ID != library.ID {
		t.Fatalf("resolve dated library item route = %#v / %#v, %v", routedItem, routedLibrary, err)
	}
	if _, _, err := libraryItems.ResolveLibraryItemRoute(ctx, siteIDs["localhost"], "/catalog/2039/typed-item"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("wrong dated library item route error = %v", err)
	}
	loadedLibraryItem, err = libraryItems.MoveLibraryItem(
		ctx, nil, loadedLibraryItem.ID, archive.ID, loadedLibraryItem.Version, false,
	)
	if err != nil || loadedLibraryItem.LibraryID != archive.ID {
		t.Fatalf("move across library partitions = %#v, %v", loadedLibraryItem, err)
	}
	if routedItem, routedLibrary, err = libraryItems.ResolveLibraryItemRoute(ctx, siteIDs["localhost"], "/archive/typed-item"); err != nil || routedItem.ID != loadedLibraryItem.ID || routedLibrary.ID != archive.ID {
		t.Fatalf("resolve moved library item route = %#v / %#v, %v", routedItem, routedLibrary, err)
	}
	renamedArchive := archive
	renamedArchivePath := "/renamed-archive"
	renamedArchive.Path = &renamedArchivePath
	renamedArchive.Slug = "renamed-archive"
	renamedArchive, err = resourceRepository.Update(ctx, nil, archive, renamedArchive, nil)
	if err != nil {
		t.Fatalf("rename owning library: %v", err)
	}
	if routedItem, _, err = libraryItems.ResolveLibraryItemRoute(ctx, siteIDs["localhost"], "/renamed-archive/typed-item"); err != nil || routedItem.ID != loadedLibraryItem.ID {
		t.Fatalf("resolve item after library rename = %#v, %v", routedItem, err)
	}
	if _, _, err := libraryItems.ResolveLibraryItemRoute(ctx, siteIDs["localhost"], "/archive/typed-item"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("old library item route error = %v", err)
	}
	if _, err := libraryItems.MoveLibraryItem(ctx, nil, loadedLibraryItem.ID, root.ID, loadedLibraryItem.Version, false); !errors.Is(err, resource.ErrInvalidReference) {
		t.Fatalf("move to non-library error = %v", err)
	}
	crossSiteLibraryPath := "/cross-library"
	crossSiteLibrary, err := resourceRepository.Create(ctx, nil, resource.Resource{
		SiteID: siteIDs["example.com"], Type: resourcetype.Library,
		Title: "Cross-site library", Slug: "cross-library", Path: &crossSiteLibraryPath,
		IsPublic: true, IsSearchable: true, Fields: map[string]any{},
		TypeSettings: map[string]any{"item_url_pattern": "/{slug}"},
	}, nil)
	if err != nil {
		t.Fatalf("create cross-site library: %v", err)
	}
	if _, err := libraryItems.MoveLibraryItem(ctx, nil, loadedLibraryItem.ID, crossSiteLibrary.ID, loadedLibraryItem.Version, false); !errors.Is(err, resource.ErrInvalidReference) {
		t.Fatalf("cross-site library item move error = %v", err)
	}
	itemPage, err := libraryItems.QueryLibraryItems(ctx, resource.LibraryItemQuery{
		SiteID: siteIDs["localhost"], LibraryID: archive.ID, Limit: 10,
	})
	if err != nil || len(itemPage.Items) != 1 || itemPage.Items[0].ID != loadedLibraryItem.ID {
		t.Fatalf("query moved library item = %#v, %v", itemPage, err)
	}
	itemPage, err = libraryItems.QueryLibraryItems(ctx, resource.LibraryItemQuery{
		SiteID: siteIDs["localhost"], LibraryID: archive.ID, Limit: 10,
		Search: strconv.FormatInt(int64(loadedLibraryItem.ID), 10),
	})
	if err != nil || len(itemPage.Items) != 1 || itemPage.Items[0].ID != loadedLibraryItem.ID {
		t.Fatalf("query library item by id = %#v, %v", itemPage, err)
	}
	lifecycleRepository, ok := resourceRepository.(resource.LifecycleRepository)
	if !ok {
		t.Fatal("resource lifecycle repository is unavailable")
	}
	if err := lifecycleRepository.SoftDelete(ctx, nil, renamedArchive.ID); err != nil {
		t.Fatalf("soft-delete owning library: %v", err)
	}
	itemPage, err = libraryItems.QueryLibraryItems(ctx, resource.LibraryItemQuery{
		SiteID: siteIDs["localhost"], LibraryID: archive.ID, Limit: 10,
	})
	if err != nil || len(itemPage.Items) != 0 {
		t.Fatalf("query under deleted library = %#v, %v", itemPage, err)
	}
	if err := lifecycleRepository.Restore(ctx, nil, renamedArchive.ID, false); err != nil {
		t.Fatalf("restore owning library: %v", err)
	}
	if err := libraryItems.SoftDeleteLibraryItem(ctx, nil, loadedLibraryItem.ID); err != nil {
		t.Fatalf("soft-delete library item: %v", err)
	}
	if err := lifecycleRepository.SoftDelete(ctx, nil, renamedArchive.ID); err != nil {
		t.Fatalf("soft-delete library with individually deleted item: %v", err)
	}
	if err := lifecycleRepository.Restore(ctx, nil, renamedArchive.ID, false); err != nil {
		t.Fatalf("restore library with individually deleted item: %v", err)
	}
	stillDeleted, err := libraryItems.LibraryItemByID(ctx, loadedLibraryItem.ID)
	if err != nil || stillDeleted.DeletedAt == nil {
		t.Fatalf("individual deletion after library restore = %#v, %v", stillDeleted, err)
	}
	notDeleted := false
	itemPage, err = libraryItems.QueryLibraryItems(ctx, resource.LibraryItemQuery{
		SiteID: siteIDs["localhost"], LibraryID: archive.ID, Limit: 10, Deleted: &notDeleted,
	})
	if err != nil || len(itemPage.Items) != 0 {
		t.Fatalf("query after soft delete = %#v, %v", itemPage, err)
	}
	if err := libraryItems.RestoreLibraryItem(ctx, nil, loadedLibraryItem.ID); err != nil {
		t.Fatalf("restore library item: %v", err)
	}
	if err := libraryItems.DeleteLibraryItem(ctx, loadedLibraryItem.ID); err != nil {
		t.Fatalf("permanently delete library item: %v", err)
	}
	if _, err := libraryItems.LibraryItemByID(ctx, loadedLibraryItem.ID); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("deleted library item error = %v", err)
	}

	child.Slug = "renamed"
	child.Title = "Renamed child"
	child.ParentID = &section.ID
	renamedPath := "/section/renamed"
	child.Path = &renamedPath
	child, err = resourceRepository.Update(ctx, nil, child, child, nil)
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
		nil,
		section, section, nil); !errors.Is(err, resource.ErrInvalidTree) {
		t.Fatalf("resource cycle error = %v", err)
	}

	internalLinkPath := "/section/renamed/internal-link"
	internalLinkTarget := grandchild.ID
	if _, err := resourceRepository.Create(
		ctx,
		nil,
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
			Fields:           map[string]any{},
		}, nil); err != nil {
		t.Fatalf("create internal resource link: %v", err)
	}

	externalLinkPath := "/external-link"
	externalLinkTarget := grandchild.ID
	externalLink, err := resourceRepository.Create(
		ctx,
		nil,
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
			Fields:           map[string]any{},
		}, nil)

	if err != nil {
		t.Fatalf("create external resource link: %v", err)
	}

	if err := resourceRepository.Delete(
		ctx,
		child.ID,
	); !errors.Is(err, resource.ErrReferenced) {
		t.Fatalf("referenced subtree delete error = %v", err)
	}
	if _, err := connector.Pool().Exec(ctx, `
UPDATE core.resources
SET image_media_id = $1
WHERE id = ANY($2);
`, sharedMedia.ID, []int64{
		int64(child.ID),
		int64(externalLink.ID),
	}); err != nil {
		t.Fatalf("create corrupted shared media attachment: %v", err)
	}
	if err := resourceRepository.Delete(
		ctx,
		externalLink.ID,
	); err != nil {
		t.Fatalf("delete external resource link: %v", err)
	}
	if _, err := mediaRepository.ByID(
		ctx,
		sharedMedia.ID,
	); !errors.Is(err, media.ErrNotFound) {
		t.Fatalf("shared media after resource delete = %v", err)
	}
	childAfterSharedDelete, err := resourceRepository.ByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("load resource after shared media delete: %v", err)
	}
	if childAfterSharedDelete.ImageMediaID != nil {
		t.Fatalf(
			"shared media reference was not cleared: %#v",
			childAfterSharedDelete.ImageMediaID,
		)
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

	if err := fileRepository.DeleteFile(
		ctx,
		imageFile.ID,
		func(context.Context, []corefile.File) error {
			return nil
		},
	); !errors.Is(err, corefile.ErrInUse) {
		t.Fatalf("delete used media source file error = %v", err)
	}
	if _, err := mediaRepository.ByID(
		ctx,
		replacementMedia.ID,
	); err != nil {
		t.Fatalf("media after rejected file delete = %v", err)
	}
	rootAfterFileDelete, err := resourceRepository.ByID(ctx, root.ID)
	if err != nil {
		t.Fatalf("load resource after media file delete: %v", err)
	}
	if rootAfterFileDelete.ImageMediaID == nil ||
		*rootAfterFileDelete.ImageMediaID != replacementMedia.ID {
		t.Fatalf(
			"resource media after rejected file delete = %#v",
			rootAfterFileDelete.ImageMediaID,
		)
	}

	if _, err := database.Sites().Update(
		ctx,
		nil,
		site.Site{
			ID:       site.ID(1 << 62),
			Domain:   "missing.example.com",
			Locale:   "en-US",
			Settings: map[string]any{},
		},
	); !errors.Is(err, site.ErrNotFound) {
		t.Fatalf("missing site update error = %v", err)
	}

	if err := seedManager.Down(ctx, devSeedPlan, 3); err != nil {
		t.Fatalf("dev seed down: %v", err)
	}
	if err := seedManager.Down(ctx, sharedSeedPlan, 1); err != nil {
		t.Fatalf("shared seed down: %v", err)
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
	if err := manager.Down(ctx, plan, 9); err != nil {
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
