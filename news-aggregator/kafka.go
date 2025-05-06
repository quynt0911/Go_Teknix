package main

import (
	"log"

	"github.com/segmentio/kafka-go"
)

func ProduceArticle(article string) {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9093"},
		Topic:   "news-updates",
	})

	err := writer.WriteMessages(nil, kafka.Message{
		Value: []byte(article),
	})
	if err != nil {
		log.Fatal("failed to produce message: ", err)
	}

	writer.Close()
}

func ConsumeNews() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9093"},
		Topic:   "news-updates",
		GroupID: "news-group",
	})

	for {
		m, err := reader.ReadMessage(nil)
		if err != nil {
			log.Fatal("failed to read message: ", err)
		}
		log.Printf("Consumed message: %s\n", string(m.Value))
	}
}
