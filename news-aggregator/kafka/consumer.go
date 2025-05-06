package kafka

import (
    "context"
    "fmt"
    "log"
    "github.com/segmentio/kafka-go"
)

func StartConsumer(newsChan chan string) {
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers: []string{"localhost:9092"},
        Topic:   "news_topic",
        GroupID: "news-consumer-group",
    })

    go func() {
        for {
            m, err := r.ReadMessage(context.Background())
            if err != nil {
                log.Println("Consumer error:", err)
                continue
            }
            fmt.Println("Received message:", string(m.Value))
            newsChan <- string(m.Value)
        }
    }()
}
