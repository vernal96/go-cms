package corefiles

import (
	"context"
	"fmt"
	"strings"

	"github.com/vernal96/go-cms/connectors/localstorage"
	connectors3 "github.com/vernal96/go-cms/connectors/s3"
	"github.com/vernal96/go-cms/kernel/filesystem"
)

const (
	PublicCode  filesystem.Code = "public"
	PrivateCode filesystem.Code = "private"
)

type Config struct {
	Driver string      `envconfig:"DRIVER" default:"local"`
	Local  LocalConfig `envconfig:"LOCAL"`
	S3     S3Config    `envconfig:"S3"`
}

type LocalConfig struct {
	Root       string `envconfig:"ROOT"`
	BaseURL    string `envconfig:"BASE_URL" default:"http://localhost:8080"`
	SigningKey string `envconfig:"SIGNING_KEY"`
}

type S3Config struct {
	Region          string `envconfig:"REGION"`
	Bucket          string `envconfig:"BUCKET"`
	Prefix          string `envconfig:"PREFIX"`
	Endpoint        string `envconfig:"ENDPOINT"`
	UsePathStyle    bool   `envconfig:"USE_PATH_STYLE" default:"false"`
	PublicBaseURL   string `envconfig:"PUBLIC_BASE_URL"`
	AccessKeyID     string `envconfig:"ACCESS_KEY_ID"`
	SecretAccessKey string `envconfig:"SECRET_ACCESS_KEY"`
	SessionToken    string `envconfig:"SESSION_TOKEN"`
}

type Factory struct {
	code       filesystem.Code
	visibility filesystem.Visibility
	config     Config
}

func PublicFactory(config Config) Factory {
	return Factory{
		code:       PublicCode,
		visibility: filesystem.VisibilityPublic,
		config:     config,
	}
}

func PrivateFactory(config Config) Factory {
	return Factory{
		code:       PrivateCode,
		visibility: filesystem.VisibilityPrivate,
		config:     config,
	}
}

func (f Factory) Code() filesystem.Code {
	return f.code
}

func (f Factory) Open(ctx context.Context) (filesystem.Disk, error) {
	switch strings.ToLower(strings.TrimSpace(f.config.Driver)) {
	case "local", "localstorage":
		if strings.TrimSpace(f.config.Local.Root) == "" {
			return nil, fmt.Errorf(
				"local root for filesystem disk %q is empty",
				f.code,
			)
		}
		return localstorage.New(ctx, localstorage.Config{
			Code:       f.code,
			Visibility: f.visibility,
			Root:       f.config.Local.Root,
			BaseURL:    f.config.Local.BaseURL,
			SigningKey: f.config.Local.SigningKey,
		})
	case "s3":
		return connectors3.New(ctx, connectors3.Config{
			Code:            f.code,
			Visibility:      f.visibility,
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
			f.code,
		)
	}
}

var _ filesystem.Factory = Factory{}
