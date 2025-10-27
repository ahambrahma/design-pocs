package main

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
		Protocol: 2,  // Connection protocol
	})

	ctx := context.Background()
	// result := client.Ping(ctx)
	// if result != nil {
	// 	fmt.Println(result.Result())
	// }

	go publish(ctx, client, "chat:room1")
	go subscribe(ctx, client, "chat:room1", 1)
	go subscribe(ctx, client, "chat:room1", 2)
	go subscribe(ctx, client, "chat:room1", 3)

	time.Sleep(100 * time.Millisecond)
}

func publish(ctx context.Context, client *redis.Client, name string) {
	for i := 0; i < 20; i++ {
		result := client.Publish(ctx, name, i)
		if result != nil {
			fmt.Printf("Published message: %v\n", result)
		}
	}
}

func subscribe(ctx context.Context, client *redis.Client, name string, id int) {
	pubsub := client.Subscribe(ctx, name)
	if pubsub != nil {
		for {
			msg, err := pubsub.Receive(ctx)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			fmt.Printf("Subscriber %d: Received message: %s\n", id, msg)
		}
	}
}
