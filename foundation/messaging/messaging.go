package messaging

import (
	"context"
	"log"

	kafka "github.com/segmentio/kafka-go"
)

var kafkaWriter *kafka.Writer

// GetKafkaWriter
func GetKafkaWriter(kafkaURL, topic string, l *log.Logger) *kafka.Writer {
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaURL),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		Logger:   l,
	}
	return kafkaWriter
}

func ProduceMessage(ctx context.Context, msg kafka.Message) error {
	err := kafkaWriter.WriteMessages(ctx, msg)
	if err != nil {
		return err
	}
	return nil
}
