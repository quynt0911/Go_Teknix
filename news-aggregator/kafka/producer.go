package kafka

import (
    "context"
    "log"
    "github.com/segmentio/kafka-go"
)

var Writer *kafka.Writer

func InitProducer() {
    Writer = &kafka.Writer{
        Addr:     kafka.TCP("localhost:9092"),
        Topic:    "news_topic",
        Balancer: &kafka.LeastBytes{},
    }
    log.Println("Kafka producer initialized")
}

func SendNewsToKafka(newsJson string) error {
    msg := kafka.Message{Value: []byte(newsJson)}
    return Writer.WriteMessages(context.Background(), msg)
}
