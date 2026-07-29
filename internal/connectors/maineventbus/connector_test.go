package maineventbus

import (
	"context"
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Driver:             "kafka",
		ConsumerRetryDelay: time.Millisecond,
		ShutdownTimeout:    time.Second,
		Kafka: KafkaConfig{
			Brokers:     []string{"localhost:1"},
			ClientID:    "cms-test",
			DialTimeout: time.Millisecond,
		},
		RabbitMQ: RabbitMQConfig{
			URL:               "not an AMQP URL",
			Exchange:          "",
			DialTimeout:       -1,
			ReconnectAttempts: -1,
			ReconnectDelay:    -1,
		},
	}
}

func TestFactoryValidatesOnlySelectedDriver(t *testing.T) {
	config := validConfig()
	connector, err := NewFactory(config).Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if connector == nil {
		t.Fatal("Kafka connector is nil")
	}
	if err := connector.Close(); err != nil {
		t.Fatal(err)
	}

	config.Driver = "rabbitmq"
	if _, err := NewFactory(config).Open(
		context.Background(),
	); err == nil || !strings.Contains(err.Error(), "URL") {
		t.Fatalf("RabbitMQ config error = %v", err)
	}

	config.Kafka = KafkaConfig{}
	config.RabbitMQ = RabbitMQConfig{
		URL:               "amqp://localhost:1/cms",
		Exchange:          "cms.events",
		DialTimeout:       time.Millisecond,
		ReconnectAttempts: 1,
		ReconnectDelay:    time.Millisecond,
	}
	if _, err := NewFactory(config).Open(
		context.Background(),
	); err == nil || strings.Contains(err.Error(), "Kafka") {
		t.Fatalf("inactive Kafka config was validated: %v", err)
	}
}

func TestFactoryRejectsMissingUnknownAndAliasedDriver(t *testing.T) {
	for _, driver := range []string{"", "RabbitMQ", "rabbit", "amqp"} {
		t.Run(driver, func(t *testing.T) {
			config := validConfig()
			config.Driver = driver
			if _, err := NewFactory(config).Open(
				context.Background(),
			); err == nil || !strings.Contains(
				err.Error(),
				"unsupported event bus driver",
			) {
				t.Fatalf("driver error = %v", err)
			}
		})
	}
}
