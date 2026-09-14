package corefiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vernal96/go-cms/connectors/localstorage"
	connectors3 "github.com/vernal96/go-cms/connectors/s3"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

type Config struct {
	Code       filesystem.Code       `json:"code"`
	Label      string                `json:"label"`
	Driver     string                `json:"driver"`
	Visibility filesystem.Visibility `json:"visibility"`
	Local      LocalConfig           `json:"local,omitempty"`
	S3         S3Config              `json:"s3,omitempty"`
}

type Configs []Config

func (c *Configs) Decode(value string) error {
	if c == nil {
		return errors.New("filesystem configs target is nil")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		*c = nil
		return nil
	}
	var decoded []Config
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return fmt.Errorf("decode filesystem disks: %w", err)
	}
	*c = decoded
	return nil
}

func (c Configs) Factories() []filesystem.Factory {
	result := make([]filesystem.Factory, len(c))
	for index, config := range c {
		result[index] = Factory{config: config}
	}
	return result
}

type LocalConfig struct {
	Root       string `json:"root"`
	BaseURL    string `json:"base_url"`
	SigningKey string `json:"signing_key,omitempty"`
}

type S3Config struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	UsePathStyle    bool   `json:"use_path_style,omitempty"`
	PublicBaseURL   string `json:"public_base_url,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty"`
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
