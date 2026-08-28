package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel/migrations"
	corepostgres "github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/forms"
)

func TestPostgresFormsSiteIsolationResultsActionsAndCascade(t *testing.T) {
	host := os.Getenv("CMS_TEST_FORMS_POSTGRES_HOST")
	if host == "" {
		t.Skip("set CMS_TEST_FORMS_POSTGRES_HOST to run the Forms PostgreSQL integration test")
	}
	port := 5432
	if raw := os.Getenv("CMS_TEST_FORMS_POSTGRES_PORT"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("parse PostgreSQL port: %v", err)
		}
		port = value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	connector, err := connectorpostgres.New(ctx, connectorpostgres.Config{
		Code: "forms-integration", Host: host, Port: port,
		Database: os.Getenv("CMS_TEST_FORMS_POSTGRES_DB"), User: os.Getenv("CMS_TEST_FORMS_POSTGRES_USER"), Password: os.Getenv("CMS_TEST_FORMS_POSTGRES_PASSWORD"),
		SSLMode: "disable", MaxConns: 4, ConnectTimeout: 5 * time.Second, ConnMaxLifetime: time.Minute,
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
		if err := connector.Pool().QueryRow(ctx, `INSERT INTO core.sites(profile_code,domain,locale,settings,is_public) VALUES('forms-test',$1,'ru-RU','{}'::jsonb,true) RETURNING id;`, fmt.Sprintf("forms-%d-%d.example.test", suffix, index)).Scan(&siteIDs[index]); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = connector.Pool().Exec(context.Background(), `DELETE FROM core.sites WHERE id=ANY($1::bigint[]);`, []int64{int64(siteIDs[0]), int64(siteIDs[1])})
	})

	repository := database.Forms()
	create := func(siteID site.ID, name string) (forms.FormDetail, error) {
		return repository.CreateForm(ctx, forms.CreateFormInput{
			Form:    forms.Form{SiteID: siteID, Code: "feedback", Name: name, Enabled: true},
			Consent: forms.FormField{Code: forms.MandatoryConsentCode, Type: forms.FieldTypeConsent, Label: "Согласие", Required: true, Options: forms.ConsentOptions{}, ResultLabel: "Согласие", ShowInResults: true},
			Captcha: forms.FormField{Code: forms.MandatoryCaptchaCode, Type: forms.FieldTypeCaptcha, Label: "CAPTCHA", Required: true, Options: forms.CaptchaOptions{}},
			Submit:  forms.Element{Code: forms.MandatorySubmitCode, Type: forms.ElementSubmitButton, Config: []byte(`{"label":"Отправить"}`)},
			Status:  forms.Status{Code: forms.DefaultStatusCode, Name: "Новый", IsDefault: true},
		})
	}
	first, err := create(siteIDs[0], "Feedback A")
	if err != nil {
		t.Fatal(err)
	}
	second, err := create(siteIDs[1], "Feedback B")
	if err != nil {
		t.Fatalf("same code on another site: %v", err)
	}
	if len(first.Fields) != 2 || len(first.Elements) != 1 || len(first.Layout) != 3 || len(first.Statuses) != 1 || !first.Statuses[0].IsDefault {
		t.Fatalf("atomic default structure = %#v", first)
	}
	for _, item := range first.Fields {
		if item.VisibleWhen != nil {
			t.Fatalf("nil visibility condition round trip = %#v", item.VisibleWhen)
		}
	}
	if _, err := create(siteIDs[0], "Duplicate"); !errors.Is(err, forms.ErrConflict) {
		t.Fatalf("duplicate same-site form error = %v", err)
	}
	if _, err := repository.FormDetail(ctx, siteIDs[1], first.Form.ID); !errors.Is(err, forms.ErrNotFound) {
		t.Fatalf("cross-site form read error = %v", err)
	}

	email, _, err := repository.CreateField(ctx, siteIDs[0], first.Form.ID, forms.FormField{Code: "email", Type: field.TypeEmail, Label: "Email", ResultLabel: "Контакт", ShowInResults: true, ResultPosition: 2})
	if err != nil {
		t.Fatal(err)
	}
	action, err := repository.CreateAction(ctx, siteIDs[0], first.Form.ID, forms.Action{Code: "notify", Name: "Notify", Enabled: true, Trigger: forms.Trigger{Type: forms.TriggerSubmitted}, ActionType: "test", Config: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	created, err := repository.CreateResult(ctx, forms.SubmissionRecord{
		Result: forms.Result{SiteID: siteIDs[0], FormID: first.Form.ID, FormCode: first.Form.Code, FormName: first.Form.Name, StatusID: first.Statuses[0].ID, UserAgent: "integration"},
		Values: []forms.ResultValue{{FieldID: &email.ID, FieldCode: email.Code, FieldLabel: email.Label, ResultLabel: email.ResultLabel, FieldType: email.Type, StorageKind: field.StorageString, Value: "person@example.test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Values) != 1 || created.Values[0].Value != "person@example.test" || len(created.Executions) != 1 || created.Executions[0].ActionID == nil || *created.Executions[0].ActionID != action.ID {
		t.Fatalf("created result = %#v", created)
	}
	var outboxCount int
	if err := connector.Pool().QueryRow(ctx, `SELECT count(*) FROM core.outbox_messages WHERE topic=$1;`, "job.forms.execute_action").Scan(&outboxCount); err != nil || outboxCount != 1 {
		t.Fatalf("Forms outbox count = %d, %v", outboxCount, err)
	}
	work, claimed, err := repository.ClaimExecution(ctx, siteIDs[0], created.Executions[0].ID, 3)
	if err != nil || !claimed || work.Execution.AttemptCount != 1 || len(work.Values) != 1 {
		t.Fatalf("claim = %#v, %t, %v", work, claimed, err)
	}
	if _, claimed, err := repository.ClaimExecution(ctx, siteIDs[0], created.Executions[0].ID, 3); !errors.Is(err, forms.ErrExecutionBusy) || claimed {
		t.Fatalf("concurrent duplicate claim = %t, %v", claimed, err)
	}
	if _, err := connector.Pool().Exec(ctx, `UPDATE forms.action_executions SET updated_at=clock_timestamp()-interval '11 minutes' WHERE id=$1;`, created.Executions[0].ID); err != nil {
		t.Fatal(err)
	}
	work, claimed, err = repository.ClaimExecution(ctx, siteIDs[0], created.Executions[0].ID, 3)
	if err != nil || !claimed || work.Execution.AttemptCount != 2 {
		t.Fatalf("expired execution lease claim = %#v, %t, %v", work, claimed, err)
	}
	if err := repository.FinishExecution(ctx, created.Executions[0].ID, forms.ExecutionSucceeded, "", "external-1"); err != nil {
		t.Fatal(err)
	}
	if _, claimed, err := repository.ClaimExecution(ctx, siteIDs[0], created.Executions[0].ID, 3); err != nil || claimed {
		t.Fatalf("succeeded duplicate claim = %t, %v", claimed, err)
	}

	doneStatus, err := repository.CreateStatus(ctx, siteIDs[0], first.Form.ID, forms.Status{Code: "done", Name: "Готово", Position: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateAction(ctx, siteIDs[0], first.Form.ID, forms.Action{Code: "on_done", Name: "On done", Enabled: true, Trigger: forms.Trigger{Type: forms.TriggerStatusChanged, To: "done"}, ActionType: "test", Config: []byte(`{}`), Position: 1}); err != nil {
		t.Fatal(err)
	}
	changed, err := repository.ChangeResultStatus(ctx, forms.ResultStatusChange{SiteID: siteIDs[0], ResultID: created.Result.ID, FromStatusID: first.Statuses[0].ID, ToStatusID: doneStatus.ID})
	if err != nil || changed.Result.StatusCode != "done" || len(changed.Executions) != 2 {
		t.Fatalf("status change = %#v, %v", changed, err)
	}
	if _, err := repository.ChangeResultStatus(ctx, forms.ResultStatusChange{SiteID: siteIDs[0], ResultID: created.Result.ID, FromStatusID: first.Statuses[0].ID, ToStatusID: second.Statuses[0].ID}); !errors.Is(err, forms.ErrConflict) {
		t.Fatalf("cross-form status change error = %v", err)
	}

	if _, err := connector.Pool().Exec(ctx, `UPDATE core.sites SET profile_code='without-forms' WHERE id=$1;`, siteIDs[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FormByCode(ctx, siteIDs[1], "feedback", false); err != nil {
		t.Fatalf("profile removal deleted Forms data: %v", err)
	}
	if _, err := connector.Pool().Exec(ctx, `DELETE FROM core.sites WHERE id=$1;`, siteIDs[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.FormByID(ctx, siteIDs[1], second.Form.ID); !errors.Is(err, forms.ErrNotFound) {
		t.Fatalf("deleted site form remains: %v", err)
	}
	if _, err := repository.FormByID(ctx, siteIDs[0], first.Form.ID); err != nil {
		t.Fatalf("deleting Site B affected Site A: %v", err)
	}
}
