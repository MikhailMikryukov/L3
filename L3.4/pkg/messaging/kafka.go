package messaging

import (
	"encoding/json"
	kafkav2 "github.com/wb-go/wbf/kafka/kafka-v2"
	"github.com/wb-go/wbf/logger"
)

// WBKafka кафка WBF
type WBKafka struct {
	Producer *kafkav2.Producer
	Consumer *kafkav2.Consumer
}

// TaskMessage сообщение передаваемое в продюсер
type TaskMessage struct {
	ID           string
	OriginalPath string
	Params       json.RawMessage
}

// NewWBKafka создать
func NewWBKafka(brokers []string, topic string, log logger.Logger, groupID string) *WBKafka {
	producer := kafkav2.NewProducer(brokers, topic, log)
	consumer := kafkav2.NewConsumer(brokers, topic, groupID, log)

	return &WBKafka{
		Producer: producer,
		Consumer: consumer,
	}
}
