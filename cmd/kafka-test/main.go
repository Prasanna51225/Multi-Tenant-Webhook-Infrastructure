package main

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("Testing Kafka connectivity on localhost:9092...")

	conn, err := kafka.DialLeader(ctx, "tcp", "localhost:9092", "webhook.events", 0)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("SUCCESS: Connected to Kafka!")
}
