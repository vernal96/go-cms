package postgres

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

type stagedResourceFields struct {
	resourceID   int64
	siteID       int64
	templateCode *string
	rawSettings  string
	values       []field.StoredValue
}

func (d *Database) PrepareResourceFields(ctx context.Context, blueprints map[kernel.ProfileCode]*kernel.ProfileBlueprint) (kernel.ResourceFieldMigrationReport, error) {
	if err := d.requireResourceFieldSchema(ctx, true); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	rows, err := d.connector.Pool().Query(ctx, `
SELECT resource.id, resource.site_id, site.profile_code, resource.template, resource.settings::text
FROM core.resources resource JOIN core.sites site ON site.id=resource.site_id ORDER BY resource.id;`)
	if err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	defer rows.Close()
	staged := make([]stagedResourceFields, 0)
	report := kernel.ResourceFieldMigrationReport{}
	for rows.Next() {
		var resourceID, siteID int64
		var profileCode string
		var templateCode *string
		var raw string
		if err := rows.Scan(&resourceID, &siteID, &profileCode, &templateCode, &raw); err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		values, err := compileLegacySettings(blueprints, kernel.ProfileCode(profileCode), templateCode, raw)
		if err != nil {
			return kernel.ResourceFieldMigrationReport{}, fmt.Errorf("resource %d: %w", resourceID, err)
		}
		staged = append(staged, stagedResourceFields{resourceID: resourceID, siteID: siteID, templateCode: templateCode, rawSettings: raw, values: values})
		report.Resources++
		report.Rows += len(values)
	}
	if err := rows.Err(); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	tx, err := d.connector.Pool().Begin(ctx)
	if err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS core.resource_field_migration_manifest (
 resource_id BIGINT PRIMARY KEY, site_id BIGINT NOT NULL, template_code TEXT NULL, source_digest TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS core.resource_field_migration_rows (
 resource_id BIGINT NOT NULL, site_id BIGINT NOT NULL, field_key TEXT NOT NULL, position INTEGER NOT NULL,
 is_multi BOOLEAN NOT NULL, value_kind TEXT NOT NULL, value_string TEXT NULL, value_integer BIGINT NULL,
 value_float DOUBLE PRECISION NULL, value_boolean BOOLEAN NULL, value_timestamp TIMESTAMPTZ NULL,
 value_reference BIGINT NULL, value_json JSONB NULL, PRIMARY KEY(resource_id,field_key,position)
);
TRUNCATE core.resource_field_migration_rows, core.resource_field_migration_manifest;`); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	for _, item := range staged {
		if _, err := tx.Exec(ctx, `INSERT INTO core.resource_field_migration_manifest(resource_id,site_id,template_code,source_digest) VALUES($1,$2,$3,md5($4::jsonb::text));`, item.resourceID, item.siteID, item.templateCode, item.rawSettings); err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		if err := insertMigrationFieldValues(ctx, tx, item.resourceID, item.siteID, item.values, "core.resource_field_migration_rows"); err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	return report, nil
}

func (d *Database) AuditResourceFields(ctx context.Context, blueprints map[kernel.ProfileCode]*kernel.ProfileBlueprint) (kernel.ResourceFieldMigrationReport, error) {
	if err := d.requireResourceFieldSchema(ctx, false); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	rows, err := d.connector.Pool().Query(ctx, `
SELECT resource.id, resource.site_id, site.profile_code, resource.template
FROM core.resources resource JOIN core.sites site ON site.id=resource.site_id ORDER BY resource.id;`)
	if err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	defer rows.Close()
	report := kernel.ResourceFieldMigrationReport{}
	for rows.Next() {
		var resourceID, siteID int64
		var profileCode string
		var templateCode *string
		if err := rows.Scan(&resourceID, &siteID, &profileCode, &templateCode); err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		actual, count, err := d.typedResourceFields(ctx, resourceID)
		if err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		report.Resources++
		report.Rows += count
		expected, err := normalizeResourceFields(blueprints, kernel.ProfileCode(profileCode), templateCode, actual)
		if err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("resource %d: %v", resourceID, err))
			continue
		}
		if count == 0 && templateCode != nil {
			if blueprint, exists := blueprints[kernel.ProfileCode(profileCode)]; exists {
				if runtime, exists := blueprint.Template(template.Code(*templateCode)); exists && len(runtime.FieldSchema().Definitions()) > 0 {
					report.Issues = append(report.Issues, fmt.Sprintf("resource %d: template defines persistent fields but no typed values are present; verify against a trusted backup", resourceID))
				}
			}
		}
		if !canonicalEqual(actual, expected) {
			report.Issues = append(report.Issues, fmt.Sprintf("resource %d: typed values do not match the current template schema", resourceID))
		}
	}
	return report, rows.Err()
}

type repairRecord struct {
	ResourceID   int64          `json:"resource_id"`
	SiteID       int64          `json:"site_id"`
	TemplateCode *string        `json:"template_code"`
	Settings     map[string]any `json:"settings"`
}

func (d *Database) RepairResourceFields(ctx context.Context, blueprints map[kernel.ProfileCode]*kernel.ProfileBlueprint, input io.Reader) (kernel.ResourceFieldMigrationReport, error) {
	if input == nil {
		return kernel.ResourceFieldMigrationReport{}, errors.New("trusted repair input is required")
	}
	if err := d.requireResourceFieldSchema(ctx, false); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	decoder := json.NewDecoder(bufio.NewReader(input))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	records := make([]repairRecord, 0)
	seen := make(map[int64]struct{})
	for {
		var record repairRecord
		if err := decoder.Decode(&record); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return kernel.ResourceFieldMigrationReport{}, fmt.Errorf("decode trusted repair input: %w", err)
		}
		if record.ResourceID <= 0 || record.SiteID <= 0 || record.Settings == nil {
			return kernel.ResourceFieldMigrationReport{}, errors.New("trusted repair record is invalid")
		}
		if _, duplicate := seen[record.ResourceID]; duplicate {
			return kernel.ResourceFieldMigrationReport{}, fmt.Errorf("trusted repair input repeats resource %d", record.ResourceID)
		}
		seen[record.ResourceID] = struct{}{}
		records = append(records, record)
	}
	tx, err := d.connector.Pool().Begin(ctx)
	if err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	report := kernel.ResourceFieldMigrationReport{}
	for _, record := range records {
		var profileCode string
		var currentTemplate *string
		if err := tx.QueryRow(ctx, `SELECT site.profile_code, resource.template FROM core.resources resource JOIN core.sites site ON site.id=resource.site_id WHERE resource.id=$1 AND resource.site_id=$2 FOR UPDATE OF resource;`, record.ResourceID, record.SiteID).Scan(&profileCode, &currentTemplate); errors.Is(err, pgx.ErrNoRows) {
			return kernel.ResourceFieldMigrationReport{}, fmt.Errorf("repair resource %d was not found", record.ResourceID)
		} else if err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		if !equalOptionalString(currentTemplate, record.TemplateCode) {
			return kernel.ResourceFieldMigrationReport{}, fmt.Errorf("repair resource %d template does not match the backup", record.ResourceID)
		}
		normalized, values, err := compileSettingsMap(blueprints, kernel.ProfileCode(profileCode), record.TemplateCode, record.Settings)
		if err != nil {
			return kernel.ResourceFieldMigrationReport{}, fmt.Errorf("repair resource %d: %w", record.ResourceID, err)
		}
		_ = normalized
		if _, err := tx.Exec(ctx, `DELETE FROM core.resource_field_values WHERE resource_id=$1;`, record.ResourceID); err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		if err := insertMigrationFieldValues(ctx, tx, record.ResourceID, record.SiteID, values, "core.resource_field_values"); err != nil {
			return kernel.ResourceFieldMigrationReport{}, err
		}
		report.Resources++
		report.Rows += len(values)
	}
	if err := tx.Commit(ctx); err != nil {
		return kernel.ResourceFieldMigrationReport{}, err
	}
	return report, nil
}

func (d *Database) requireResourceFieldSchema(ctx context.Context, legacy bool) error {
	var hasSettings, hasTyped bool
	if err := d.connector.Pool().QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='core' AND table_name='resources' AND column_name='settings'),
 to_regclass('core.resource_field_values') IS NOT NULL;`).Scan(&hasSettings, &hasTyped); err != nil {
		return err
	}
	if legacy && (!hasSettings || hasTyped) {
		return errors.New("resource-fields prepare requires a schema before migration 000015")
	}
	if !legacy && (hasSettings || !hasTyped) {
		return errors.New("resource-fields audit/repair requires migration 000015")
	}
	return nil
}

func compileLegacySettings(blueprints map[kernel.ProfileCode]*kernel.ProfileBlueprint, profile kernel.ProfileCode, code *string, raw string) ([]field.StoredValue, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var settings map[string]any
	if err := decoder.Decode(&settings); err != nil {
		return nil, err
	}
	_, values, err := compileSettingsMap(blueprints, profile, code, settings)
	return values, err
}

func compileSettingsMap(blueprints map[kernel.ProfileCode]*kernel.ProfileBlueprint, profile kernel.ProfileCode, code *string, settings map[string]any) (map[string]any, []field.StoredValue, error) {
	blueprint, exists := blueprints[profile]
	if !exists {
		return nil, nil, fmt.Errorf("unknown profile %q", profile)
	}
	if code == nil {
		if len(settings) != 0 {
			return nil, nil, errors.New("resource without a template has legacy settings")
		}
		return map[string]any{}, nil, nil
	}
	runtime, exists := blueprint.Template(template.Code(*code))
	if !exists {
		return nil, nil, fmt.Errorf("unknown template %q", *code)
	}
	normalized, err := runtime.FieldSchema().Validate(settings)
	if err != nil {
		return nil, nil, err
	}
	values, err := runtime.FieldSchema().StoredValues(normalized)
	return normalized, values, err
}

func normalizeResourceFields(blueprints map[kernel.ProfileCode]*kernel.ProfileBlueprint, profile kernel.ProfileCode, code *string, values map[string]any) (map[string]any, error) {
	normalized, _, err := compileSettingsMap(blueprints, profile, code, values)
	return normalized, err
}

func insertMigrationFieldValues(ctx context.Context, tx pgx.Tx, resourceID, siteID int64, values []field.StoredValue, table string) error {
	for _, value := range values {
		columns, err := migrationStoredColumns(value)
		if err != nil {
			return err
		}
		query := `INSERT INTO ` + table + `(resource_id,site_id,field_key,position,is_multi,value_kind,value_string,value_integer,value_float,value_boolean,value_timestamp,value_reference,value_json) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);`
		if _, err := tx.Exec(ctx, query, resourceID, siteID, value.Key, value.Position, value.Multiple, value.Kind, columns[0], columns[1], columns[2], columns[3], columns[4], columns[5], columns[6]); err != nil {
			return fmt.Errorf("stage field %q: %w", value.Key, err)
		}
	}
	return nil
}

func migrationStoredColumns(value field.StoredValue) ([7]any, error) {
	var result [7]any
	switch value.Kind {
	case field.StorageString:
		result[0] = value.Value
	case field.StorageInteger:
		result[1] = value.Value
	case field.StorageFloat:
		result[2] = value.Value
	case field.StorageBoolean:
		result[3] = value.Value
	case field.StorageTimestamp:
		result[4] = value.Value
	case field.StorageReference:
		result[5] = value.Value
	case field.StorageJSON:
		raw, err := json.Marshal(value.Value)
		if err != nil {
			return result, err
		}
		result[6] = string(raw)
	default:
		return result, fmt.Errorf("field %q has invalid storage kind %q", value.Key, value.Kind)
	}
	return result, nil
}

func (d *Database) typedResourceFields(ctx context.Context, resourceID int64) (map[string]any, int, error) {
	rows, err := d.connector.Pool().Query(ctx, `SELECT field_key,position,is_multi,value_kind,value_string,value_integer,value_float,value_boolean,value_timestamp,value_reference,value_json FROM core.resource_field_values WHERE resource_id=$1 ORDER BY field_key,position;`, resourceID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := make(map[string]any)
	count := 0
	for rows.Next() {
		var key, kind string
		var position int
		var multi bool
		var text *string
		var integer, reference *int64
		var floatValue *float64
		var boolean *bool
		var timestamp *time.Time
		var raw []byte
		if err := rows.Scan(&key, &position, &multi, &kind, &text, &integer, &floatValue, &boolean, &timestamp, &reference, &raw); err != nil {
			return nil, 0, err
		}
		var value any
		switch field.StorageKind(kind) {
		case field.StorageString:
			value = *text
		case field.StorageInteger:
			value = *integer
		case field.StorageFloat:
			value = *floatValue
		case field.StorageBoolean:
			value = *boolean
		case field.StorageTimestamp:
			value = *timestamp
		case field.StorageReference:
			value = *reference
		case field.StorageJSON:
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if err := decoder.Decode(&value); err != nil {
				return nil, 0, err
			}
		default:
			return nil, 0, fmt.Errorf("resource %d field %q has unknown storage kind %q", resourceID, key, kind)
		}
		if multi {
			items, _ := result[key].([]any)
			result[key] = append(items, value)
		} else {
			result[key] = value
		}
		count++
	}
	return result, count, rows.Err()
}

func canonicalEqual(left, right map[string]any) bool {
	leftRaw, _ := canonicalJSON(left)
	rightRaw, _ := canonicalJSON(right)
	return bytes.Equal(leftRaw, rightRaw)
}

func canonicalJSON(value map[string]any) ([]byte, error) {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]any, len(keys))
	for _, key := range keys {
		ordered[key] = value[key]
	}
	return json.Marshal(ordered)
}

func equalOptionalString(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

var _ kernel.ResourceFieldMigrator = (*Database)(nil)
