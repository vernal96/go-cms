package maineventbus

import (
	"context"
	"fmt"
	"time"

	"github.com/vernal96/go-cms/connectors/kafkaeventbus"
	"github.com/vernal96/go-cms/connectors/rabbiteventbus"
	"github.com/vernal96/go-cms/kernel/eventbus"
)

type Config struct {
	Driver             string         `envconfig:"DRIVER" required:"true"`
	ConsumerRetryDelay time.Duration  `envconfig:"CONSUMER_RETRY_DELAY" default:"1s"`
	ShutdownTimeout    time.Duration  `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
	Kafka              KafkaConfig    `envconfig:"KAFKA"`
	RabbitMQ           RabbitMQConfig `envconfig:"RABBITMQ"`
}

type KafkaConfig struct {
	Brokers       []string      `envconfig:"BROKERS" default:"localhost:9092"`
	ClientID      string        `envconfig:"CLIENT_ID" default:"go-cms"`
	Username      string        `envconfig:"USERNAME"`
	Password      string        `envconfig:"PASSWORD"`
	TLSEnabled    bool          `envconfig:"TLS_ENABLED" default:"false"`
	TLSServerName string        `envconfig:"TLS_SERVER_NAME"`
	TLSInsecure   bool          `envconfig:"TLS_INSECURE" default:"false"`
	DialTimeout   time.Duration `envconfig:"DIAL_TIMEOUT" default:"5s"`
}

type RabbitMQConfig struct {
	URL               string        `envconfig:"URL"`
	Exchange          string        `envconfig:"EXCHANGE" default:"cms.events"`
	DialTimeout       time.Duration `envconfig:"DIAL_TIMEOUT" default:"5s"`
	ReconnectAttempts int           `envconfig:"RECONNECT_ATTEMPTS" default:"30"`
	ReconnectDelay    time.Duration `envconfig:"RECONNECT_DELAY" default:"1s"`
}

type Factory struct {
	config Config
}

func NewFactory(config Config) Factory {
	return Factory{config: config}
}

func (f Factory) Open(
	ctx context.Context,
) (eventbus.Connector, error) {
	switch f.config.Driver {
	case "kafka":
		return kafkaeventbus.New(ctx, kafkaeventbus.Config{
			Brokers:            f.config.Kafka.Brokers,
			ClientID:           f.config.Kafka.ClientID,
			Username:           f.config.Kafka.Username,
			Password:           f.config.Kafka.Password,
			TLSEnabled:         f.config.Kafka.TLSEnabled,
			TLSServerName:      f.config.Kafka.TLSServerName,
			TLSInsecure:        f.config.Kafka.TLSInsecure,
			DialTimeout:        f.config.Kafka.DialTimeout,
			ConsumerRetryDelay: f.config.ConsumerRetryDelay,
			ShutdownTimeout:    f.config.ShutdownTimeout,
		})
	case "rabbitmq":
		return rabbiteventbus.New(ctx, rabbiteventbus.Config{
			URL:                f.config.RabbitMQ.URL,
			Exchange:           f.config.RabbitMQ.Exchange,
			DialTimeout:        f.config.RabbitMQ.DialTimeout,
			ConsumerRetryDelay: f.config.ConsumerRetryDelay,
			ShutdownTimeout:    f.config.ShutdownTimeout,
			ReconnectAttempts:  f.config.RabbitMQ.ReconnectAttempts,
			ReconnectDelay:     f.config.RabbitMQ.ReconnectDelay,
		})
	default:
		return nil, fmt.Errorf(
			"unsupported event bus driver %q",
			f.config.Driver,
		)
	}
}

var _ eventbus.Factory = Factory{}
