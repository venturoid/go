package rabbitmq

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/venturoid/go/redis"
)

// Config is a struct to store rabbitmq configuration
type Config struct {
	RabbitMQConfig  *RabbitMQConfig
	RedisConnection *redis.Redis
	Retries         int
	Delay           time.Duration
	Pool            int
}

// RabbitMQConfig is a struct to store rabbitmq configuration
type RabbitMQConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	VirtualHost string
}

// Rabbit is a struct to store rabbitmq connection
type Rabbit struct {
	mx  *sync.Mutex
	n   int
	dns string

	connection []*amqp.Connection
	redis      *redis.Redis
}

// Response is a struct to store message data
type Response struct {
	Status  int
	Message string
	Data    interface{}
}

type Queue struct {
	Name    string
	Durable bool
}
