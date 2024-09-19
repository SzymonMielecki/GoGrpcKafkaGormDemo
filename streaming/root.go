package streaming

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/SzymonMielecki/chatApp/chatServer/persistance"
	"github.com/SzymonMielecki/chatApp/types"
	"github.com/segmentio/kafka-go"
)

type Streaming struct {
	conn      *kafka.Conn
	topic     string
	partition int
	brokers   []string
}

func NewStreaming(topic string, partition int) *Streaming {
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, partition)
	if err != nil {
		log.Fatal("failed to dial leader:", err)
	}
	return &Streaming{conn: conn, topic: topic, partition: partition}
}

func (s *Streaming) Close() {
	s.conn.Close()
}

func (s *Streaming) UploadMessages(ctx context.Context, db *persistance.DB, wg *sync.WaitGroup) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   s.brokers,
		Topic:     s.topic,
		Partition: s.partition,
	})
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return
		default:
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}

			var message types.Message
			err = json.Unmarshal(m.Value, &message)
			if err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			_, err = db.CreateMessage(&message)
			if err != nil {
				log.Printf("Error creating message in DB: %v", err)
				continue
			}

			log.Printf("Message processed: %v", message)
		}
	}
}

func (s *Streaming) SendMessage(ctx context.Context, message *types.Message) error {
	json, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = s.conn.WriteMessages(kafka.Message{Value: json})
	return err
}

func (s *Streaming) ReceiveMessages(ctx context.Context, ch chan<- *types.Message, wg *sync.WaitGroup) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   s.brokers,
		Topic:     s.topic,
		Partition: s.partition,
	})
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return
		default:
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message: %v", err)
				continue
			}
			var message types.Message
			err = json.Unmarshal(m.Value, &message)
			if err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}
			ch <- &message
		}
	}
}
