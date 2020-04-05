package services

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

type Rabbit struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      amqp.Queue
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func (r *Rabbit) Start(host, login, password string, port int) {
	connCredential := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		login,
		password,
		host,
		port,
	)
	conn, err := amqp.Dial(connCredential)
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	args := amqp.Table{"x-message-ttl": 5 * 1000} // 5 sec
	q, err := ch.QueueDeclare(
		"buses", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		args,    // arguments
	)
	failOnError(err, "Failed to declare a queue")
	r.connection = conn
	r.queue = q
	r.channel = ch
	fmt.Println("RabbitMQ started")
}

func (r *Rabbit) Stop() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.connection != nil {
		r.connection.Close()
	}
	fmt.Println("Stop RabbitMQ")
}

func (r *Rabbit) GetConsumeChan() <-chan amqp.Delivery {
	consumeCh, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	failOnError(err, "Failed to register a consumer")
	return consumeCh
}

func (r *Rabbit) SendData(msg []byte) {
	err := r.channel.Publish(
		"",           // exchange
		r.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [X] Sent %s", string(msg))
}
