package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alexander231/rabbitmq/internal"
	"github.com/joho/godotenv"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatal().Err(err).Msg(msg)
	}
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	err := godotenv.Load(".env")
	failOnError(err, "failed to load secrets")

	username := os.Getenv("USERNAME")
	password := os.Getenv("PASS")

	conn, err := internal.ConnectRabbitMQ(username, password, "localhost:5672", "customers")
	failOnError(err, "failed to create new rabittmq connection")
	defer conn.Close()

	consumeConn, err := internal.ConnectRabbitMQ(username, password, "localhost:5672", "customers")
	failOnError(err, "failed to create new rabittmq consume connection")
	defer consumeConn.Close()

	client, err := internal.NewRabbitMQClient(conn)
	failOnError(err, "failed to create new rabbitmq client")
	defer client.Close()

	consumeClient, err := internal.NewRabbitMQClient(conn)
	failOnError(err, "failed to create new rabbitmq consumeClient")
	defer consumeClient.Close()

	// Create Unnamed Queue which will generate a random name, set AutoDelete to True
	queue, err := consumeClient.CreateQueue("", true, true)
	failOnError(err, "failed to create queue")

	err = consumeClient.CreateBinding(queue.Name, queue.Name, "customer_callbacks")
	failOnError(err, "failed to create binding between queue and customer_callbacks exchange")

	messageBus, err := consumeClient.Consume(queue.Name, "customer-api", true)
	failOnError(err, "failed to create message bus to consume messages from the publisher side")

	go func() {
		for message := range messageBus {
			log.Printf("Message Callback %s\n", message.CorrelationId)
		}
	}()

	// Create context to manage timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create customer from sweden
	for i := 0; i <= 10; i++ {
		err := client.Send(ctx, "customer_events", "customers.created.se", amqp091.Publishing{
			ContentType:  "text/plain",       // The payload we send is plaintext, could be JSON or others..
			DeliveryMode: amqp091.Persistent, // This tells rabbitMQ that this message should be Saved if no resources accepts it before a restart (durable)
			Body:         []byte("An cool message between services"),
			// We add a REPLYTO which defines the
			ReplyTo: queue.Name,
			// CorrelationId can be used to know which Event this relates to
			CorrelationId: fmt.Sprintf("customer_created_%d", i),
		})
		failOnError(err, "failedd to send message to customers_events exchange")
	}

	var blocking chan struct{}

	log.Info().Msg("Waiting on Callbacks, to close the program press CTRL+C")
	// This will block forever
	<-blocking
}
