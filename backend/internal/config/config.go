package config

import (
	"net"
	"strconv"
	"time"

	"github.com/vernal96/go-cms/internal/connectors/corefiles"
	"github.com/vernal96/go-cms/internal/connectors/maineventbus"
	"github.com/vernal96/go-cms/internal/connectors/mainlogger"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	"github.com/vernal96/go-cms/internal/connectors/projectcache"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	jwtsecurity "github.com/vernal96/go-cms/internal/security/jwt"
	"github.com/vernal96/go-cms/kernel"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/filesystem"
	corepostgres "github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/user/adapters/argon2id"
	mailmodule "github.com/vernal96/go-cms/kernel/modules/mail"
	mailpostgres "github.com/vernal96/go-cms/kernel/modules/mail/adapters/postgres"
	seopostgres "github.com/vernal96/go-cms/kernel/modules/seo/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/outbox"
)

type Config struct {
	Logger   mainlogger.Config   `envconfig:"LOGGER"`
	EventBus maineventbus.Config `envconfig:"EVENT_BUS"`
	Server   ServerConfig        `envconfig:"SERVER"`
	Postgres mainpostgres.Config `envconfig:"POSTGRES"`
	Files    FilesConfig         `envconfig:"FILES"`
	Caches   projectcache.Config `envconfig:"CACHE"`
	JWT      jwtsecurity.Config  `envconfig:"JWT"`
	Outbox   OutboxConfig        `envconfig:"OUTBOX"`
	Mail     MailConfig          `envconfig:"MAIL"`
}

type MailConfig struct {
	DefaultTransport       mailmodule.TransportAlias `envconfig:"DEFAULT_TRANSPORT" default:"default"`
	DefaultDriver          string                    `envconfig:"TRANSPORT_DEFAULT_DRIVER" default:"log"`
	SMTPHost               string                    `envconfig:"SMTP_HOST"`
	SMTPPort               int                       `envconfig:"SMTP_PORT" default:"587"`
	SMTPUsername           string                    `envconfig:"SMTP_USERNAME"`
	SMTPPassword           string                    `envconfig:"SMTP_PASSWORD"`
	SMTPTLSEnabled         bool                      `envconfig:"SMTP_TLS_ENABLED" default:"true"`
	SMTPTLSServerName      string                    `envconfig:"SMTP_TLS_SERVER_NAME"`
	SMTPTimeout            time.Duration             `envconfig:"SMTP_TIMEOUT" default:"10s"`
	AllowedSenderAddresses []string                  `envconfig:"ALLOWED_SENDER_ADDRESSES"`
	AllowedSenderDomains   []string                  `envconfig:"ALLOWED_SENDER_DOMAINS"`
	MessageIDDomain        string                    `envconfig:"MESSAGE_ID_DOMAIN" default:"localhost"`
	HistoryRetention       time.Duration             `envconfig:"HISTORY_RETENTION" default:"720h"`
	HistoryCleanupInterval time.Duration             `envconfig:"HISTORY_CLEANUP_INTERVAL" default:"1h"`
	HistoryCleanupBatch    int                       `envconfig:"HISTORY_CLEANUP_BATCH" default:"100"`
	SendMaxAttempts        int                       `envconfig:"SEND_MAX_ATTEMPTS" default:"5"`
	MaxRecipients          int                       `envconfig:"MAX_RECIPIENTS" default:"100"`
	MaxMessageSize         int64                     `envconfig:"MAX_MESSAGE_SIZE" default:"26214400"`
	MaxAttachmentSize      int64                     `envconfig:"MAX_ATTACHMENT_SIZE" default:"20971520"`
	UploadStorage          filesystem.Code           `envconfig:"UPLOAD_STORAGE" default:"private"`
	UploadPath             string                    `envconfig:"UPLOAD_PATH" default:"mail"`
	SpoolEnabled           bool                      `envconfig:"SPOOL_ENABLED" default:"true"`
	SpoolTTL               time.Duration             `envconfig:"SPOOL_TTL" default:"24h"`
	SpoolCleanupInterval   time.Duration             `envconfig:"SPOOL_CLEANUP_INTERVAL" default:"1h"`
	SpoolCleanupBatch      int                       `envconfig:"SPOOL_CLEANUP_BATCH" default:"100"`
}

func (c MailConfig) ModuleConfig() mailmodule.Config {
	var transport mailmodule.Transport
	switch c.DefaultDriver {
	case "smtp":
		transport = mailmodule.ConfiguredSMTPTransport{Config: mailmodule.SMTPConfig{
			Host: c.SMTPHost, Port: c.SMTPPort, Username: c.SMTPUsername, Password: c.SMTPPassword,
			TLSEnabled: c.SMTPTLSEnabled, TLSServerName: c.SMTPTLSServerName, Timeout: c.SMTPTimeout,
		}}
	case "null":
		transport = mailmodule.NullTransport{}
	case "log":
		transport = mailmodule.LogTransport{}
	default:
		transport = mailmodule.InvalidTransport{Name: c.DefaultDriver}
	}
	return mailmodule.Config{
		DefaultTransport: c.DefaultTransport,
		Transports:       map[mailmodule.TransportAlias]mailmodule.Transport{c.DefaultTransport: transport},
		Renderer: mailmodule.RendererConfig{SenderPolicy: mailmodule.SenderPolicy{
			AllowedAddresses: append([]string(nil), c.AllowedSenderAddresses...),
			AllowedDomains:   append([]string(nil), c.AllowedSenderDomains...),
		}},
		MessageIDDomain: c.MessageIDDomain, HistoryRetention: c.HistoryRetention,
		CleanupInterval: c.HistoryCleanupInterval, CleanupBatchSize: c.HistoryCleanupBatch,
		SendMaxAttempts: c.SendMaxAttempts, MaxRecipients: c.MaxRecipients,
		MaxMessageSize: c.MaxMessageSize, MaxAttachmentSize: c.MaxAttachmentSize,
		UploadStorage: c.UploadStorage, UploadPath: c.UploadPath,
		SpoolEnabled: c.SpoolEnabled, SpoolTTL: c.SpoolTTL,
		SpoolCleanupInterval: c.SpoolCleanupInterval, SpoolCleanupBatch: c.SpoolCleanupBatch,
	}
}

type OutboxConfig struct {
	PollInterval       time.Duration `envconfig:"POLL_INTERVAL" default:"500ms"`
	BatchSize          int           `envconfig:"BATCH_SIZE" default:"100"`
	LeaseDuration      time.Duration `envconfig:"LEASE_DURATION" default:"30s"`
	InitialRetryDelay  time.Duration `envconfig:"INITIAL_RETRY_DELAY" default:"1s"`
	MaximumRetryDelay  time.Duration `envconfig:"MAXIMUM_RETRY_DELAY" default:"1h"`
	PublishedRetention time.Duration `envconfig:"PUBLISHED_RETENTION" default:"168h"`
	CleanupInterval    time.Duration `envconfig:"CLEANUP_INTERVAL" default:"1h"`
	CleanupMaxBatches  int           `envconfig:"CLEANUP_MAX_BATCHES" default:"100"`
}

func (c OutboxConfig) PublisherConfig() outbox.PublisherConfig {
	return outbox.PublisherConfig{
		PollInterval: c.PollInterval, BatchSize: c.BatchSize, LeaseDuration: c.LeaseDuration,
		InitialRetryDelay: c.InitialRetryDelay, MaximumRetryDelay: c.MaximumRetryDelay,
		PublishedRetention: c.PublishedRetention, CleanupInterval: c.CleanupInterval,
		CleanupMaxBatches: c.CleanupMaxBatches,
	}
}

type FilesConfig struct {
	Public        corefiles.Config `envconfig:"PUBLIC"`
	Private       corefiles.Config `envconfig:"PRIVATE"`
	MaxUploadSize int64            `envconfig:"MAX_UPLOAD_SIZE" default:"104857600"`
	UploadTimeout time.Duration    `envconfig:"UPLOAD_TIMEOUT" default:"10m"`
	AvatarStorage filesystem.Code  `envconfig:"AVATAR_STORAGE" default:"private"`
	AvatarMaxSize int64            `envconfig:"AVATAR_MAX_SIZE" default:"5242880"`
}

type ServerConfig struct {
	Host            string        `envconfig:"HOST" default:"localhost"`
	Port            int           `envconfig:"PORT" default:"8080"`
	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration `envconfig:"IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Application is a declarative description of this application instance.
func (c Config) Application() appkernel.Definition {
	return appkernel.Definition{
		Logger:   mainlogger.NewFactory(c.Logger),
		EventBus: maineventbus.NewFactory(c.EventBus),
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: mainpostgres.Factory(c.Postgres),
			Adapters: []kernel.ModuleDatabaseFactory{
				corepostgres.DatabaseFactory{},
				seopostgres.DatabaseFactory{},
				mailpostgres.DatabaseFactory{},
			},
		},
		Filesystems: []filesystem.Factory{
			corefiles.PublicFactory(c.Files.Public),
			corefiles.PrivateFactory(c.Files.Private),
		},
		Caches:          c.Caches.Factories(),
		Profiles:        []kernel.Profile{dev.ProfileWithMail(c.Mail.ModuleConfig())},
		PasswordHasher:  argon2id.Factory{},
		MaxUploadSize:   c.Files.MaxUploadSize,
		UploadTimeout:   c.Files.UploadTimeout,
		AvatarStorage:   c.Files.AvatarStorage,
		AvatarMaxSize:   c.Files.AvatarMaxSize,
		OutboxPublisher: c.Outbox.PublisherConfig(),
	}
}
