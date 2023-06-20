# rabbitmq

## All commands should be run from the root of the project.

Use 
```docker run -d --name rabbitmq -v "$(pwd)"/rabbitmq_definitions.json:/etc/rabbitmq/rabbitmq_definitions.json:ro -v "$(pwd)"/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro -v "$(pwd)"/tls-gen/basic/result:/certs -p 5671:5671 -p 15672:15672 rabbitmq:3.11-management``` to run the rabbitmq server.

Use 
```go run cmd/consumer/main.go``` to run the consumer.

Use 
```go run cmd/producer/main.go``` to run the producer.