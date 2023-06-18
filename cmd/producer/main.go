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

	// Create context to manage timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create customer from sweden
	for i := 0; i <= 10; i++ {
		err := client.Send(ctx, "customer_events", "customers.created.se", amqp091.Publishing{
			ContentType:  "text/plain",       // The payload we send is plaintext, could be JSON or others..
			DeliveryMode: amqp091.Persistent, // This tells rabbitMQ that this message should be Saved if no resources accepts it before a restart (durable)
			Body:         []byte("An cool message between services"),
		})
		failOnError(err, "failedd to send message to customers_events exchange")
	}

	log.Info().Msgf("%v", client)
}
