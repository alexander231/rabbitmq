package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alexander231/rabbitmq/internal"
	"github.com/joho/godotenv"
	"github.com/rabbitmq/amqp091-go"
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
	// best practice but since it is blocking forever will never be called
	defer conn.Close()

	publishConn, err := internal.ConnectRabbitMQ(username, password, "localhost:5672", "customers")
	if err != nil {
		panic(err)
	}
	// best practice but since it is blocking forever will never be called
	defer publishConn.Close()

	client, err := internal.NewRabbitMQClient(conn)
	if err != nil {
		panic(err)
	}
	// best practice but since it is blocking forever will never be called
	defer client.Close()

	publishClient, err := internal.NewRabbitMQClient(publishConn)
	if err != nil {
		panic(err)
	}
	// best practice but since it is blocking forever will never be called
	defer publishClient.Close()

	// Create Unnamed Queue which will generate a random name, set AutoDelete to True
	queue, err := client.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}

	// Create binding between the customer_events exchange and the new Random Queue
	// Can skip Binding key since fanout will skip that rule
	if err := client.CreateBinding(queue.Name, "", "customer_events"); err != nil {
		panic(err)
	}

	messageBus, err := client.Consume(queue.Name, "email-service", false)
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

	// Apply Qos to limit amount of messages to consume
	if err := client.ApplyQos(10, 0, true); err != nil {
		panic(err)
	}
	go func() {
		for message := range messageBus {
			// Spawn a worker
			msg := message
			g.Go(func() error {
				// Multiple means that we acknowledge a batch of messages, leave false for now
				if err := msg.Ack(false); err != nil {
					log.Printf("Acknowledged message failed: Retry ? Handle manually %s\n", msg.MessageId)
					return err
				}

				log.Printf("Acknowledged message, replying to %s\n", msg.ReplyTo)

				// Use the msg.ReplyTo to send the message to the proper Queue
				if err := publishClient.Send(ctx, "customer_callbacks", msg.ReplyTo, amqp091.Publishing{
					ContentType:   "text/plain",      // The payload we send is plaintext, could be JSON or others..
					DeliveryMode:  amqp091.Transient, // This tells rabbitMQ to drop messages if restarted
					Body:          []byte("RPC Complete"),
					CorrelationId: msg.CorrelationId,
				}); err != nil {
					panic(err)
				}
				return nil
			})
		}
	}()

	log.Println("Consuming, to close the program press CTRL+C")
	// This will block forever
	<-blocking

}
