package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alexander231/rabbitmq/internal"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASS")

	conn, err := internal.ConnectRabbitMQ(username, password, "localhost:5672", "customers")
	if err != nil {
		panic(err)
	}

	mqClient, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}

	// Create Unnamed Queue which will generate a random name, set AutoDelete to True
	queue, err := mqClient.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}

	// Create binding between the customer_events exchange and the new Random Queue
	// Can skip Binding key since fanout will skip that rule
	if err := mqClient.CreateBinding(queue.Name, "", "customer_events"); err != nil {
		panic(err)
	}

	messageBus, err := mqClient.Consume(queue.Name, "email-service", false)
	if err != nil {
		panic(err)
	}

	// blocking is used to block forever
	var blocking chan struct{}

	// go func() {
	// 	for message := range messageBus {
	// 		// breakpoint here
	// 		log.Printf("New Message: %v", string(message.Body))

	// 		// Multiple means that we acknowledge a batch of messages, leave false for now
	// 		if err := message.Ack(false); err != nil {
	// 			log.Printf("Acknowledged message failed: Retry ? Handle manually %s\n", message.MessageId)
	// 			continue
	// 		}
	// 		log.Printf("Acknowledged message %s\n", message.MessageId)
	// 	}
	// }()

	// Set a timeout for 15 secs
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// Create an Errgroup to manage concurrecy
	g, ctx := errgroup.WithContext(ctx)
	// Set amount of concurrent tasks
	g.SetLimit(10)
	go func() {
		for message := range messageBus {
			// Spawn a worker
			msg := message
			g.Go(func() error {
				log.Printf("New Message: %v", string(msg.Body))

				time.Sleep(10 * time.Second)
				// Multiple means that we acknowledge a batch of messages, leave false for now
				if err := msg.Ack(false); err != nil {
					log.Printf("Acknowledged message failed: Retry ? Handle manually %s\n", msg.MessageId)
					return err
				}
				log.Printf("Acknowledged message %s\n", msg.MessageId)
				return nil
			})
		}
	}()

	log.Println("Consuming, to close the program press CTRL+C")
	// This will block forever
	<-blocking

}
