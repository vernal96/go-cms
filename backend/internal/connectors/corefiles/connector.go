package corefiles

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vernal96/go-cms-kernel/connectors/localstorage"
	connectors3 "github.com/vernal96/go-cms-kernel/connectors/s3"
	"github.com/vernal96/go-cms-kernel/filesystem"
)

type Config struct {
	Code       filesystem.Code
	Label      string
	Driver     string
	Visibility filesystem.Visibility
	Local      LocalConfig
	S3         S3Config
}

type LocalConfig struct {
	Root       string
	BaseURL    string
	SigningKey string
}

type S3Config struct {
	Region          string
	Bucket          string
	Prefix          string
	Endpoint        string
	UsePathStyle    bool
	PublicBaseURL   string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

type Factory struct {
	config Config
}

func NewFactory(config Config) Factory {
	return Factory{config: config}
}

func (f Factory) Code() filesystem.Code {
	return f.config.Code
}

func (f Factory) Label() string {
	return f.config.Label
}

func (f Factory) Open(ctx context.Context) (filesystem.Disk, error) {
	if err := validateConfig(f.config); err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(f.config.Driver)) {
	case "local", "localstorage":
		return localstorage.New(ctx, localstorage.Config{
			Code:       f.config.Code,
			Visibility: f.config.Visibility,
			Root:       f.config.Local.Root,
			BaseURL:    f.config.Local.BaseURL,
			SigningKey: f.config.Local.SigningKey,
		})
	case "s3":
		return connectors3.New(ctx, connectors3.Config{
			Code:            f.config.Code,
			Visibility:      f.config.Visibility,
			Region:          f.config.S3.Region,
			Bucket:          f.config.S3.Bucket,
			Prefix:          f.config.S3.Prefix,
			Endpoint:        f.config.S3.Endpoint,
			UsePathStyle:    f.config.S3.UsePathStyle,
			PublicBaseURL:   f.config.S3.PublicBaseURL,
			AccessKeyID:     f.config.S3.AccessKeyID,
			SecretAccessKey: f.config.S3.SecretAccessKey,
			SessionToken:    f.config.S3.SessionToken,
		})
	default:
		return nil, fmt.Errorf(
			"unsupported driver %q for filesystem disk %q",
			f.config.Driver,
			f.config.Code,
		)
	}
}

func validateConfig(config Config) error {
	if strings.TrimSpace(string(config.Code)) == "" {
		return errors.New("filesystem disk code is empty")
	}
	if strings.TrimSpace(config.Label) == "" {
		return fmt.Errorf("filesystem disk %q label is empty", config.Code)
	}
	if !filesystem.ValidVisibility(config.Visibility) {
		return fmt.Errorf(
			"filesystem disk %q: %w: %q",
			config.Code,
			filesystem.ErrInvalidVisibility,
			config.Visibility,
		)
	}
	if strings.TrimSpace(config.Driver) == "" {
		return fmt.Errorf("filesystem disk %q driver is empty", config.Code)
	}
	return nil
}

var _ filesystem.Factory = Factory{}
var _ filesystem.FactoryLabelProvider = Factory{}
