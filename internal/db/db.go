package db

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	BROKER_ADDR = "localhost:29092"
	TOPIC       = "links"
	PARTITION   = 0
)

var (
	kafkaConn *kafka.Conn
)

func init() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var err error
	kafkaConn, err = kafka.DialLeader(ctx, "tcp", BROKER_ADDR, TOPIC, PARTITION)
	if err != nil {
		log.Fatalf("on connecting to kafka: %s", err.Error())
	}

}

func GetKafkaConn() *kafka.Conn {
	return kafkaConn
}

func main() {
	kafkaConn := GetKafkaConn()

	brokers, err := kafkaConn.Brokers()
	if err != nil {
		log.Fatal(err)
	}

	for _, b := range brokers {
		log.Print(b)
	}

	_, err = kafkaConn.Write([]byte(`hello`))
	if err != nil {
		log.Fatal(err)
	}

	msg, err := kafkaConn.ReadMessage(100)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%s", msg.Value)
}
