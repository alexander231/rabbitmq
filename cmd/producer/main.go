package main

import (
	"context"
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

	client, err := internal.NewRabbitMQClient(conn)
	failOnError(err, "failed to create new rabbitmq client")
	defer client.Close()

	err = client.CreateQueue("customers_created", true, false)
	failOnError(err, "failed to create customers_create queue")
	err = client.CreateQueue("customers_test", false, true)
	failOnError(err, "failed to create customers_test queue")

	// Create binding between the customer_events exchange and the customers-created queue
	err = client.CreateBinding("customers_created", "customers.created.*", "customer_events")
	failOnError(err, "failed to create binding between customers-created and customer_events")

	// Create binding between the customer_events exchange and the customers-test queue
	err = client.CreateBinding("customers_test", "customers.*", "customer_events")
	failOnError(err, "failed to create binding between customers-test and customer_events")
	// Create context to manage timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Create customer from sweden
	err = client.Send(ctx, "customer_events", "customers.created.se", amqp091.Publishing{
		ContentType:  "text/plain",       // The payload we send is plaintext, could be JSON or others..
		DeliveryMode: amqp091.Persistent, // This tells rabbitMQ that this message should be Saved if no resources accepts it before a restart (durable)
		Body:         []byte("An cool message between services"),
	})
	failOnError(err, "failedd to send message to customers_events exchange")

	err = client.Send(ctx, "customer_events", "customers.test", amqp091.Publishing{
		ContentType:  "text/plain",
		DeliveryMode: amqp091.Transient, // This tells rabbitMQ that this message can be deleted if no resources accepts it before a restart (non durable)
		Body:         []byte("A second cool message"),
	})
	failOnError(err, "failedd to send message to customers_events exchange")

	log.Info().Msgf("%v", client)
}
