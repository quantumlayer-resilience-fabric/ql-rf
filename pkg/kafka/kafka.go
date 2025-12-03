// Package kafka provides Kafka producer and consumer functionality.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
)

// Producer is a Kafka message producer.
type Producer struct {
	producer sarama.SyncProducer
	logger   *slog.Logger
}

// Consumer is a Kafka message consumer.
type Consumer struct {
	consumer sarama.ConsumerGroup
	logger   *slog.Logger
}

// Message represents a Kafka message.
type Message struct {
	Key       string
	Value     []byte
	Topic     string
	Partition int32
	Offset    int64
	Timestamp time.Time
	Headers   map[string]string
}

// Event is the base structure for all events.
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// NewProducer creates a new Kafka producer.
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 5
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		logger:   slog.Default().With("component", "kafka-producer"),
	}, nil
}

// Publish publishes a message to the given topic.
func (p *Producer) Publish(ctx context.Context, topic string, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(data),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Debug("message published",
		"topic", topic,
		"key", key,
		"partition", partition,
		"offset", offset,
	)

	return nil
}

// PublishEvent publishes an event to the given topic.
func (p *Producer) PublishEvent(ctx context.Context, topic string, event Event) error {
	return p.Publish(ctx, topic, event.ID, event)
}

// Close closes the producer.
func (p *Producer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// MessageHandler handles incoming Kafka messages.
type MessageHandler func(ctx context.Context, msg Message) error

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler.
type ConsumerGroupHandler struct {
	handler MessageHandler
	logger  *slog.Logger
}

// Setup is called at the beginning of a new session.
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is called at the end of a session.
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a partition.
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := context.Background()

		headers := make(map[string]string)
		for _, header := range msg.Headers {
			headers[string(header.Key)] = string(header.Value)
		}

		message := Message{
			Key:       string(msg.Key),
			Value:     msg.Value,
			Topic:     msg.Topic,
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Timestamp: msg.Timestamp,
			Headers:   headers,
		}

		if err := h.handler(ctx, message); err != nil {
			h.logger.Error("failed to process message",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"error", err,
			)
			// Continue processing other messages
			continue
		}

		session.MarkMessage(msg, "")
	}

	return nil
}

// NewConsumer creates a new Kafka consumer.
func NewConsumer(cfg config.KafkaConfig) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
	saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

	consumer, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &Consumer{
		consumer: consumer,
		logger:   slog.Default().With("component", "kafka-consumer"),
	}, nil
}

// Subscribe subscribes to the given topics and processes messages with the handler.
func (c *Consumer) Subscribe(ctx context.Context, topics []string, handler MessageHandler) error {
	groupHandler := &ConsumerGroupHandler{
		handler: handler,
		logger:  c.logger,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.consumer.Consume(ctx, topics, groupHandler); err != nil {
				c.logger.Error("consumer error", "error", err)
				return fmt.Errorf("consumer error: %w", err)
			}
		}
	}
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	if c.consumer != nil {
		return c.consumer.Close()
	}
	return nil
}

// Health checks the Kafka connection health.
func (p *Producer) Health(ctx context.Context, brokers []string) error {
	config := sarama.NewConfig()
	config.Net.DialTimeout = 5 * time.Second

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer client.Close()

	return nil
}

// Client is a high-level Kafka client that combines producer and consumer.
type Client struct {
	Producer *Producer
	Consumer *Consumer
	Brokers  []string
	logger   *slog.Logger
}

// ClientConfig holds configuration for the Kafka client.
type ClientConfig struct {
	Brokers       []string
	ConsumerGroup string
	Topics        []string
}

// NewClient creates a new Kafka client.
func NewClient(cfg config.KafkaConfig) (*Client, error) {
	producer, err := NewProducer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Client{
		Producer: producer,
		Brokers:  cfg.Brokers,
		logger:   slog.Default().With("component", "kafka-client"),
	}, nil
}

// Health checks the Kafka connection.
func (c *Client) Health(ctx context.Context) error {
	if c.Producer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}
	return c.Producer.Health(ctx, c.Brokers)
}

// Close closes the Kafka client.
func (c *Client) Close() error {
	var err error
	if c.Producer != nil {
		if e := c.Producer.Close(); e != nil {
			err = e
		}
	}
	if c.Consumer != nil {
		if e := c.Consumer.Close(); e != nil {
			err = e
		}
	}
	return err
}
