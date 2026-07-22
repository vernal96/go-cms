package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	cmsapp "github.com/vernal96/go-cms/app"
	"github.com/vernal96/go-cms/internal/bootstrap"
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/kernel/migrations"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (resultErr error) {
	if len(args) == 0 {
		return errors.New(
			"usage: migrate <up|down|status|version|force> [flags]",
		)
	}

	projectConfig, err := config.Load()
	if err != nil {
		return err
	}

	infrastructure, err := bootstrap.NewInfrastructure(ctx, projectConfig)
	if err != nil {
		return err
	}
	defer func() {
		resultErr = errors.Join(resultErr, infrastructure.Close())
	}()

	plans, err := cmsapp.MigrationPlans(infrastructure.AppConfig())
	if err != nil {
		return err
	}

	manager := migrations.NewManager()

	switch args[0] {
	case "up":
		selected, err := parseSelection("up", args[1:], plans, false)
		if err != nil {
			return err
		}

		if err := manager.UpAll(ctx, selected); err != nil {
			return err
		}

		return printJSON(map[string]any{
			"command": "up",
			"plans":   planNames(selected),
		})

	case "status":
		selected, err := parseSelection("status", args[1:], plans, false)
		if err != nil {
			return err
		}

		statuses, err := manager.StatusAll(ctx, selected)
		if err != nil {
			return err
		}

		return printJSON(statuses)

	case "version":
		selected, err := parseSelection("version", args[1:], plans, false)
		if err != nil {
			return err
		}

		statuses, err := manager.StatusAll(ctx, selected)
		if err != nil {
			return err
		}

		type versionResult struct {
			Connection        string `json:"connection"`
			Module            string `json:"module"`
			Version           uint   `json:"version"`
			HasCurrentVersion bool   `json:"has_current_version"`
			Dirty             bool   `json:"dirty"`
		}

		result := make([]versionResult, 0, len(statuses))
		for _, status := range statuses {
			result = append(result, versionResult{
				Connection:        status.Connection,
				Module:            status.Source,
				Version:           status.CurrentVersion,
				HasCurrentVersion: status.HasCurrentVersion,
				Dirty:             status.Dirty,
			})
		}

		return printJSON(result)

	case "down":
		flags := flag.NewFlagSet("down", flag.ContinueOnError)
		connection := flags.String("connection", "", "connection code")
		module := flags.String("module", "", "module code")
		steps := flags.Int("steps", 1, "number of migrations to roll back")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		if *connection == "" || *module == "" {
			return errors.New("down requires -connection and -module")
		}

		selected, err := selectPlans(plans, *connection, *module)
		if err != nil {
			return err
		}

		if err := manager.Down(ctx, selected[0], *steps); err != nil {
			return err
		}

		return printJSON(map[string]any{
			"command": "down",
			"plan":    planNames(selected)[0],
			"steps":   *steps,
		})

	case "force":
		flags := flag.NewFlagSet("force", flag.ContinueOnError)
		connection := flags.String("connection", "", "connection code")
		module := flags.String("module", "", "module code")
		version := flags.String("version", "", "version to force, or -1")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		if *connection == "" || *module == "" || *version == "" {
			return errors.New(
				"force requires -connection, -module and -version",
			)
		}

		forcedVersion, err := strconv.Atoi(*version)
		if err != nil {
			return fmt.Errorf("parse force version: %w", err)
		}

		selected, err := selectPlans(plans, *connection, *module)
		if err != nil {
			return err
		}

		if err := manager.Force(ctx, selected[0], forcedVersion); err != nil {
			return err
		}

		return printJSON(map[string]any{
			"command": "force",
			"plan":    planNames(selected)[0],
			"version": forcedVersion,
		})

	default:
		return fmt.Errorf("unknown migration command %q", args[0])
	}
}

func parseSelection(
	command string,
	args []string,
	plans []migrations.Plan,
	requireBoth bool,
) ([]migrations.Plan, error) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	connection := flags.String("connection", "", "optional connection code")
	module := flags.String("module", "", "optional module code")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	if requireBoth && (*connection == "" || *module == "") {
		return nil, fmt.Errorf(
			"%s requires -connection and -module",
			command,
		)
	}

	return selectPlans(plans, *connection, *module)
}

func selectPlans(
	plans []migrations.Plan,
	connection string,
	module string,
) ([]migrations.Plan, error) {
	selected := make([]migrations.Plan, 0, len(plans))

	for _, plan := range plans {
		if connection != "" && plan.Connection != connection {
			continue
		}

		if module != "" && plan.Source.ID != module {
			continue
		}

		selected = append(selected, plan)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf(
			"no migration plans match connection=%q module=%q",
			connection,
			module,
		)
	}

	return selected, nil
}

func planNames(plans []migrations.Plan) []string {
	result := make([]string, 0, len(plans))
	for _, plan := range plans {
		result = append(
			result,
			plan.Connection+"/"+plan.Source.ID,
		)
	}

	return result
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
