package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/console"
)

type resourceFieldCommandProvider struct {
	app      *App
	migrator kernel.ResourceFieldMigrator
}

func (p resourceFieldCommandProvider) Commands() []console.Command {
	if p.app == nil || p.migrator == nil {
		return nil
	}
	return []console.Command{resourceFieldsCommand{app: p.app, migrator: p.migrator}}
}

type resourceFieldsCommand struct {
	app      *App
	migrator kernel.ResourceFieldMigrator
}

func (resourceFieldsCommand) Name() string { return "resource-fields" }
func (resourceFieldsCommand) Description() string {
	return "prepare, audit, or repair typed resource fields"
}
func (resourceFieldsCommand) RequiresBoot() bool { return false }

func (c resourceFieldsCommand) Run(ctx context.Context, args []string, streams console.IO) error {
	if len(args) == 0 || args[0] == "help" {
		_, err := fmt.Fprintln(streams.Out, "Usage: console resource-fields <prepare|audit|repair --input backup.jsonl>")
		return err
	}
	blueprints, err := c.compileBlueprints(ctx)
	if err != nil {
		return err
	}
	var report kernel.ResourceFieldMigrationReport
	switch args[0] {
	case "prepare":
		if len(args) != 1 {
			return errors.New("resource-fields prepare does not accept arguments")
		}
		report, err = c.migrator.PrepareResourceFields(ctx, blueprints)
	case "audit":
		if len(args) != 1 {
			return errors.New("resource-fields audit does not accept arguments")
		}
		report, err = c.migrator.AuditResourceFields(ctx, blueprints)
	case "repair":
		flags := flag.NewFlagSet("resource-fields repair", flag.ContinueOnError)
		flags.SetOutput(streams.Err)
		inputPath := flags.String("input", "", "trusted JSONL backup/export")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 || *inputPath == "" {
			return errors.New("resource-fields repair requires --input <jsonl>")
		}
		input, openErr := os.Open(*inputPath)
		if openErr != nil {
			return fmt.Errorf("open trusted repair input: %w", openErr)
		}
		defer input.Close()
		report, err = c.migrator.RepairResourceFields(ctx, blueprints, input)
	default:
		return fmt.Errorf("unknown resource-fields subcommand %q", args[0])
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(streams.Out).Encode(report)
}

func (c resourceFieldsCommand) compileBlueprints(ctx context.Context) (map[kernel.ProfileCode]*kernel.ProfileBlueprint, error) {
	factory, err := kernel.NewProfileRuntimeFactory(c.app, kernel.RuntimeServices{
		Caches: c.app.caches, Filesystems: c.app.filesystems, EventBus: c.app.eventBus, Logger: c.app.logger,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[kernel.ProfileCode]*kernel.ProfileBlueprint, len(c.app.definition.Profiles))
	for _, profile := range c.app.definition.Profiles {
		blueprint, err := factory.Compile(ctx, profile)
		if err != nil {
			return nil, fmt.Errorf("compile profile %q for resource fields: %w", profile.Code, err)
		}
		result[profile.Code] = blueprint
	}
	return result, nil
}
