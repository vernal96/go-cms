package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/migrations"
	corepostgres "github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/mail"
)

func TestPostgresMailCRUDOutboxAttemptsRetentionAndSiteIsolation(t *testing.T) {
	host := os.Getenv("CMS_TEST_MAIL_POSTGRES_HOST")
	if host == "" {
		t.Skip("set CMS_TEST_MAIL_POSTGRES_HOST to run the Mail PostgreSQL integration test")
	}
	port := 5432
	if raw := os.Getenv("CMS_TEST_MAIL_POSTGRES_PORT"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("parse CMS_TEST_MAIL_POSTGRES_PORT: %v", err)
		}
		port = value
	}
	sslMode := os.Getenv("CMS_TEST_MAIL_POSTGRES_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	connector, err := connectorpostgres.New(ctx, connectorpostgres.Config{
		Code: "mail-integration", Host: host, Port: port,
		Database: os.Getenv("CMS_TEST_MAIL_POSTGRES_DB"), User: os.Getenv("CMS_TEST_MAIL_POSTGRES_USER"), Password: os.Getenv("CMS_TEST_MAIL_POSTGRES_PASSWORD"),
		SSLMode: sslMode, MaxConns: 4, ConnectTimeout: 5 * time.Second, ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connector.Close() })
	coreDatabase, err := corepostgres.NewDatabase(connector)
	if err != nil {
		t.Fatal(err)
	}
	database, err := NewDatabase(connector)
	if err != nil {
		t.Fatal(err)
	}
	manager := migrations.NewManager()
	for _, source := range []migrations.Source{coreDatabase.MigrationSources()[0], database.MigrationSources()[0]} {
		if err := manager.Up(ctx, migrations.Plan{Connection: string(connector.Code()), Target: connector, Source: source}); err != nil {
			t.Fatalf("migrate %s: %v", source.ID, err)
		}
	}

	suffix := time.Now().UnixNano()
	siteIDs := make([]site.ID, 2)
	for index := range siteIDs {
		if err := connector.Pool().QueryRow(ctx, `INSERT INTO core.sites(profile_code,domain,locale,settings,is_public) VALUES('mail-test',$1,'en-US','{}'::jsonb,true) RETURNING id;`, fmt.Sprintf("mail-%d-%d.example.test", suffix, index)).Scan(&siteIDs[index]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = connector.Pool().Exec(context.Background(), `DELETE FROM core.sites WHERE id=ANY($1::bigint[]);`, []int64{int64(siteIDs[0]), int64(siteIDs[1])})
	})

	repository := database.Mail()
	template, err := repository.CreateTemplate(ctx, mail.Template{
		SiteID: siteIDs[0], Code: "invoice", Name: "Invoice", Enabled: true, Transport: "default",
		From: mail.AddressTemplate{Email: "noreply@example.test"}, To: []mail.AddressTemplate{{Email: "{{data.email}}"}},
		Subject: "Invoice", ContentType: mail.ContentText, TextBody: "Ready",
		Variables: []field.Definition{{Key: "email", Type: field.TypeEmail, Label: "Email"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateTemplate(ctx, mail.Template{SiteID: siteIDs[0], Code: "invoice", Name: "Duplicate", Enabled: true, Transport: "default", From: mail.AddressTemplate{}, ContentType: mail.ContentText}); !errors.Is(err, mail.ErrConflict) {
		t.Fatalf("duplicate template error = %v", err)
	}
	if _, err := repository.TemplateByID(ctx, siteIDs[1], template.ID); !errors.Is(err, mail.ErrNotFound) {
		t.Fatalf("cross-site template read error = %v", err)
	}

	messageTemplateID := template.ID
	message := mail.Message{
		SiteID: siteIDs[0], TemplateID: &messageTemplateID, TemplateCode: template.Code, TemplateName: template.Name,
		Transport: "default", RFCMessageID: fmt.Sprintf("<mail-%d@example.test>", suffix),
		From: mail.Address{Email: "noreply@example.test"}, To: []mail.Address{{Email: "person@example.test"}},
		Subject: "Invoice", ContentType: mail.ContentText, TextBody: "Ready", Origin: mail.OriginManual, RequestedAt: time.Now().UTC(),
	}
	queuedJob, err := job.NewScoped(mail.SendJobName, 1, fmt.Sprint(siteIDs[0]), struct {
		MessageID mail.MessageID `json:"message_id"`
	}{})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(queuedJob)
	stored, err := repository.CreateMessageAndJob(ctx, message, eventbus.Message{Topic: job.Topic(mail.SendJobName), Key: []byte(queuedJob.ID), Body: body})
	if err != nil {
		t.Fatal(err)
	}
	var outboxCount int
	if err := connector.Pool().QueryRow(ctx, `SELECT count(*) FROM core.outbox_messages WHERE message_id=$1;`, queuedJob.ID).Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("outbox count = %d, %v", outboxCount, err)
	}
	if _, err := repository.MessageDetail(ctx, siteIDs[1], stored.ID); !errors.Is(err, mail.ErrNotFound) {
		t.Fatalf("cross-site message read error = %v", err)
	}

	claimed, firstAttempt, ok, err := repository.ClaimMessage(ctx, siteIDs[0], stored.ID, 5)
	if err != nil || !ok || claimed.ID != stored.ID {
		t.Fatalf("first claim = %#v %#v %t %v", claimed, firstAttempt, ok, err)
	}
	if err := repository.FinishAttempt(ctx, stored.ID, firstAttempt.AttemptNumber, mail.DeliveryResult{Driver: "smtp"}, &mail.DeliveryError{Retryable: true, Code: "temporary", Err: errors.New("temporary")}, false); err != nil {
		t.Fatal(err)
	}
	_, secondAttempt, ok, err := repository.ClaimMessage(ctx, siteIDs[0], stored.ID, 5)
	if err != nil || !ok || secondAttempt.AttemptNumber != 2 {
		t.Fatalf("retry claim = %#v %t %v", secondAttempt, ok, err)
	}
	if err := repository.FinishAttempt(ctx, stored.ID, secondAttempt.AttemptNumber, mail.DeliveryResult{Driver: "smtp", ResponseCode: "250", RemoteMessageID: "provider-1"}, nil, true); err != nil {
		t.Fatal(err)
	}
	page, err := repository.ListMessages(ctx, siteIDs[0], mail.MessageQuery{PageQuery: mail.PageQuery{Page: 1, PerPage: 20}})
	if err != nil || len(page.Items) != 1 || page.Items[0].AttemptCount != 2 || page.Items[0].LatestAttempt == nil || page.Items[0].LatestAttempt.Status != mail.AttemptAccepted {
		t.Fatalf("message list = %#v, %v", page, err)
	}
	if err := repository.DeleteTemplate(ctx, siteIDs[0], template.ID); err != nil {
		t.Fatal(err)
	}
	detail, err := repository.MessageDetail(ctx, siteIDs[0], stored.ID)
	if err != nil || detail.Message.TemplateID != nil || detail.Message.TemplateCode != "invoice" || len(detail.Attempts) != 2 {
		t.Fatalf("historical snapshot = %#v, %v", detail, err)
	}
	if _, err := connector.Pool().Exec(ctx, `UPDATE mail.messages SET updated_at=clock_timestamp()-interval '48 hours' WHERE id=$1;`, stored.ID); err != nil {
		t.Fatal(err)
	}
	removed, err := repository.Cleanup(ctx, 24*time.Hour, 100)
	if err != nil || removed != 1 {
		t.Fatalf("cleanup removed = %d, %v", removed, err)
	}
	if _, err := repository.MessageDetail(ctx, siteIDs[0], stored.ID); !errors.Is(err, mail.ErrNotFound) {
		t.Fatalf("message after retention = %v", err)
	}
}

var _ kernel.ModuleDatabase = (*Database)(nil)
