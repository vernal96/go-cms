// An isolated integration fixture, not a production authentication setup.
package main

import (
	"context"
	"embed"
	"errors"
	"example.org/cms-consumer/internal/notice"
	"fmt"
	"github.com/vernal96/go-cms-kernel"
	"github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/connectors/localstorage"
	"github.com/vernal96/go-cms-kernel/connectors/postgres"
	"github.com/vernal96/go-cms-kernel/eventbus"
	"github.com/vernal96/go-cms-kernel/filesystem"
	"github.com/vernal96/go-cms-kernel/logging"
	"github.com/vernal96/go-cms-kernel/migrations"
	"github.com/vernal96/go-cms-kernel/modules/admin"
	"github.com/vernal96/go-cms-kernel/modules/core"
	corepostgres "github.com/vernal96/go-cms-kernel/modules/core/adapters/postgres"
	"github.com/vernal96/go-cms-kernel/modules/core/field"
	"github.com/vernal96/go-cms-kernel/modules/core/user/adapters/argon2id"
	"github.com/vernal96/go-cms-kernel/modules/forms"
	formspostgres "github.com/vernal96/go-cms-kernel/modules/forms/adapters/postgres"
	"github.com/vernal96/go-cms-kernel/modules/mail"
	mailpostgres "github.com/vernal96/go-cms-kernel/modules/mail/adapters/postgres"
	"github.com/vernal96/go-cms-kernel/security"
	"github.com/vernal96/go-cms-kernel/seeds"
	"github.com/vernal96/go-cms-kernel/transport/httpserver"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed seeds/*.sql
var seedFiles embed.FS

func profile(code kernel.ProfileCode, extended bool) kernel.Profile {
	modules := []kernel.ProfileModule{
		{Module: core.Module{}},
		{Module: mail.Module{}, Config: mail.Config{MessageIDDomain: "example.test", SendMaxAttempts: 3, MaxRecipients: 10, MaxMessageSize: 1 << 20, MaxAttachmentSize: 1 << 20, UploadStorage: "private"}},
		{Module: forms.Module{}, Config: forms.Config{ActionMaxAttempts: 3, DefaultCaptchaProvider: "development", Public: forms.PublicLimits{MaxRequestSize: 1 << 20, MaxScalarFields: 100, MaxScalarValueSize: 10000, MaxUploadFileSize: 1 << 20, MaxUploadCount: 1, MaxTotalUploadBytes: 1 << 20, SubmissionTimeout: time.Second, RateLimit: 100, RateWindow: time.Minute, RateEntries: 100}}},
	}
	result := kernel.Profile{Code: code, Name: string(code)}
	if extended {
		modules = append(modules, kernel.ProfileModule{Module: notice.Module{}})
		result.Params = []field.Definition{
			{Key: "message", Label: "Message", Type: "example.text"},
			{Key: "messages", Label: "Messages", Type: "example.text", Options: field.StringOptions{Multiple: true, MaxItems: 3}},
		}
	}
	result.Modules = append(modules, kernel.ProfileModule{Module: admin.Module{}})
	return result
}

func newFixture(ctx context.Context, root string) (*app.App, http.Handler, error) {
	database := os.Getenv("EXAMPLE_DATABASE")
	if !strings.HasPrefix(database, "cms_extension_") {
		return nil, nil, errors.New("EXAMPLE_DATABASE must name an isolated cms_extension_* database")
	}
	token := os.Getenv("EXAMPLE_TOKEN")
	if token == "" {
		return nil, nil, errors.New("EXAMPLE_TOKEN is required")
	}
	port, err := strconv.Atoi(os.Getenv("PGPORT"))
	if err != nil {
		return nil, nil, err
	}
	definition := app.Definition{
		Logger: quietLogger{}, EventBus: discardBus{}, PasswordHasher: argon2id.Factory{},
		MainDatabase:       app.DatabaseDefinition{Connector: postgres.Factory{Config: postgres.Config{Code: "main", Host: os.Getenv("PGHOST"), Port: port, Database: database, User: os.Getenv("PGUSER"), Password: os.Getenv("PGPASSWORD"), SSLMode: "disable", MaxConns: 5, ConnMaxLifetime: time.Minute, ConnectTimeout: 5 * time.Second}}, Adapters: []kernel.ModuleDatabaseFactory{corepostgres.DatabaseFactory{}, mailpostgres.DatabaseFactory{}, formspostgres.DatabaseFactory{}}, Seeds: []app.ModuleSeedSource{{Module: core.ModuleCode, Source: seeds.Source{ID: "external_fixture", Schema: "core", Tags: []seeds.Tag{"test"}, FS: seedFiles, Path: "seeds"}}}},
		Filesystems:        []filesystem.Factory{localstorage.Factory{Config: localstorage.Config{Code: "private", Visibility: filesystem.VisibilityPrivate, Root: root, BaseURL: "http://localhost/_cms/files/private", SigningKey: "external-fixture-key-for-private-files"}}},
		ModuleApplications: []kernel.ModuleApplication{mail.Application{Transport: mail.NullTransport{}}, forms.Application{Providers: []forms.CaptchaProvider{forms.DevelopmentCaptchaProvider{ExpectedToken: "fixture"}}}},
		Profiles:           []kernel.Profile{profile("extended", true), profile("plain", false)},
	}
	application, err := app.New(ctx, definition)
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*app.App, http.Handler, error) {
		return nil, nil, errors.Join(err, application.Close())
	}
	if err = migrations.NewManager().UpAll(ctx, application.MigrationPlans()); err != nil {
		return fail(err)
	}
	if err = seeds.NewManager().UpAll(ctx, application.SeedPlans()); err != nil {
		return fail(err)
	}
	if err = application.Boot(ctx); err != nil {
		return fail(err)
	}
	handler, err := httpserver.NewHandler(application, httpserver.WithAccessTokens(fixtureTokens(token)))
	if err != nil {
		return fail(err)
	}
	return application, handler, nil
}
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	root, err := os.MkdirTemp("", "cms-extension-files-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	application, handler, err := newFixture(ctx, root)
	if err != nil {
		panic(err)
	}
	defer application.Close()
	address := os.Getenv("EXAMPLE_ADDRESS")
	if address == "" {
		address = "127.0.0.1:18081"
	}
	server, err := httpserver.NewServer(httpserver.Config{Address: address, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, ShutdownTimeout: 5 * time.Second}, handler, slog.Default())
	if err != nil {
		panic(err)
	}
	fmt.Println("external CMS fixture ready", address)
	if err = server.Run(ctx); err != nil {
		panic(err)
	}
}

// Test-only infrastructure: no external messages are sent by this fixture.
type quietLogger struct{}

func (quietLogger) Open(context.Context) (logging.Connector, error) { return quietLogger{}, nil }
func (quietLogger) Logger() *slog.Logger                            { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func (quietLogger) Ping(context.Context) error                      { return nil }
func (quietLogger) Close() error                                    { return nil }

type discardBus struct{}

func (discardBus) Open(context.Context) (eventbus.Connector, error) { return discardBus{}, nil }
func (discardBus) Publish(context.Context, eventbus.Message) error  { return nil }
func (discardBus) Consume(ctx context.Context, _ eventbus.Subscription, _ eventbus.Handler) error {
	<-ctx.Done()
	return ctx.Err()
}
func (discardBus) Ping(context.Context) error { return nil }
func (discardBus) Close() error               { return nil }

type fixtureTokens string

func (t fixtureTokens) IssueAccessToken(context.Context, security.Actor) (security.AccessToken, error) {
	return security.AccessToken{Value: string(t), ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (t fixtureTokens) VerifyAccessToken(_ context.Context, value string) (security.Actor, error) {
	if value != string(t) {
		return security.Actor{}, security.ErrInvalidAccessToken
	}
	return security.User(1), nil
}
