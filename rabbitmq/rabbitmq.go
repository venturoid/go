package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (conf *Config) connect(url string) (*amqp.Connection, error) {
	maxRetries := 5
	if conf.Retries > 0 {
		maxRetries = conf.Retries
	}

	initialDelay := time.Second * 2
	if conf.Delay > 0 {
		initialDelay = conf.Delay
	}

	for i := 0; i < maxRetries; i++ {
		retry := false
		if data, err := amqp.Dial(url); err == nil {
			return data, nil
		} else {
			log.Println("error on connecting to rabbitmq", err.Error())
			retry = true
		}

		if retry {
			select {
			case <-time.After(initialDelay):
				initialDelay *= 2
				continue
			}
		}
	}

	return nil, fmt.Errorf("failed to connect to rabbitmq")
}

func (conf *Config) createPool(url string) ([]*amqp.Connection, error) {
	pool := []*amqp.Connection{}

	if conf.Pool == 0 {
		conf.Pool = 1
	}

	for i := 0; i < conf.Pool; i++ {
		if data, err := conf.connect(url); err != nil {
			log.Println("error on creating pool", err.Error())
			return nil, err
		} else {
			pool = append(pool, data)
		}
	}

	return pool, nil
}

func NewRabbitMQ(config Config) (*Rabbit, error) {
	var dns string
	if data, err := buildDNS(config); err != nil || data == nil {
		log.Println("error on building rabbitmq dns", err.Error())
		return nil, err
	} else {
		dns = *data
	}

	var pool []*amqp.Connection
	if data, err := config.createPool(dns); err != nil {
		log.Println("error on creating connection", err.Error())
		return nil, err
	} else {
		pool = data
	}

	service := Rabbit{
		mx:  &sync.Mutex{},
		dns: dns,

		connection: pool,
		redis:      config.RedisConnection,
	}

	log.Println("successfully create rabbitmq service")
	return &service, nil
}

func (rmq *Rabbit) CloseConnection() error {
	if rmq.connection == nil {
		return fmt.Errorf("rabbitmq connection is empty")
	}

	for _, conn := range rmq.connection {
		if err := conn.Close(); err != nil {
			log.Println("error on close connection", err.Error())
			return fmt.Errorf("error on close connection")
		}
	}

	return nil
}

func (rmq *Rabbit) getConnection() *amqp.Connection {
	rmq.mx.Lock()
	defer rmq.mx.Unlock()

	if rmq.n == len(rmq.connection)-1 {
		rmq.n = 0
	}

	if rmq.connection[rmq.n].IsClosed() {
		log.Println("reconnect rabbitmq connection")
		if data, err := amqp.Dial(rmq.dns); err == nil {
			rmq.connection[rmq.n] = data
		} else {
			log.Println("error on connecting to rabbitmq", err.Error())
		}
	}

	rmq.n++
	return rmq.connection[rmq.n]
}

func (rmq *Rabbit) PublishMessage(queueConf Queue, body interface{}, waitResponse bool, isRedis bool) (Response, error) {
	var useRedis bool
	var response Response

	if isRedis {
		useRedis = true
		if rmq.redis == nil {
			response.Status = 500
			response.Message = "Redis connection is empty"
			return response, fmt.Errorf("redis connection is empty")
		}
	}

	// Prepare the message
	key := uuid.New().String()
	var request interface{}
	if reflect.TypeOf(body).Kind() == reflect.Struct || reflect.TypeOf(body).Kind() == reflect.Map {
		bodyByte, err := json.Marshal(body)
		if err != nil {
			response.Status = 500
			response.Message = err.Error()
			return response, fmt.Errorf("failed to marshal body")
		}
		var temp map[string]interface{}
		if err := json.Unmarshal(bodyByte, &temp); err != nil {
			response.Status = 500
			response.Message = err.Error()
			return response, fmt.Errorf("failed to unmarshal body")
		}
		temp["RedisKey"] = key
		request = temp
	} else {
		request = body
	}

	// Create channel
	channel, err := rmq.getConnection().Channel()
	if err != nil {
		response.Status = 500
		response.Message = err.Error()
		return response, fmt.Errorf("failed to create channel")
	}
	defer channel.Close()

	var queue amqp.Queue
	var corrId string
	if !useRedis {
		corrId = uuid.New().String()

		durable := true
		if !queueConf.Durable {
			durable = false
		}
		queueName := strconv.Itoa(int(time.Now().Unix())) + corrId
		data, err := channel.QueueDeclare(
			queueName,
			durable,
			true,
			false,
			false,
			nil,
		)
		if err != nil {
			response.Status = 500
			response.Message = err.Error()
			return response, fmt.Errorf("failed to declare queue")
		}
		defer channel.QueueDelete(queueName, false, false, true)
		queue = data
	}

	// Define ReplyTo
	replyTo := queue.Name
	if useRedis {
		replyTo = "redis"
	}

	// Publish the message
	ctx := context.Background()
	err = channel.PublishWithContext(
		ctx,
		"",
		queueConf.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: corrId,
			ReplyTo:       replyTo,
			Body:          []byte(JSONEncode(request)),
			Expiration:    "60000",
			MessageId:     uuid.New().String(),
		},
	)
	if err != nil {
		response.Status = 500
		response.Message = err.Error()
		return response, fmt.Errorf("failed to publish message")
	}
	log.Println("publish :", request)

	if useRedis {
		return rmq.handleRedisResponse(key)
	}

	return rmq.handleQueueResponse(queue.Name, corrId, waitResponse)
}

// handleRedisResponse handles responses from Redis
func (rmq *Rabbit) handleRedisResponse(corrId string) (Response, error) {
	var response Response
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			{
				response.Status = 500
				response.Message = "RPC timeout"
				return response, fmt.Errorf("RPC timeout")
			}
		default:
			{
				data, err := rmq.redis.Get(corrId)
				if err != nil && err.Error() != "redis: nil" {
					response.Status = 500
					response.Message = err.Error()
					return response, fmt.Errorf("failed to get data from redis")
				}
				if data != nil {
					var responseRabbit map[string]interface{}
					if err := json.Unmarshal([]byte(*data), &responseRabbit); err != nil {
						log.Println("error unmarshal response:", err.Error())
						response.Status = 500
						response.Message = err.Error()
						return response, fmt.Errorf("failed to unmarshal response")
					}

					if val, ok := responseRabbit["Status"]; ok {
						response.Status = int(val.(float64))
					}

					if val, ok := responseRabbit["StatusCode"]; ok {
						response.Status = int(val.(float64))
					}

					if val, ok := responseRabbit["StatusMessage"]; ok {
						response.Message = val.(string)
					}

					if val, ok := responseRabbit["Message"]; ok {
						response.Message = val.(string)
					}

					response.Data = responseRabbit
					return response, nil
				}
			}
		}
	}
}

// handleQueueResponse handles responses from RabbitMQ queue
func (rmq *Rabbit) handleQueueResponse(queueName, corrId string, waitResponse bool) (Response, error) {
	var response Response
	if !waitResponse {
		response.Status = 200
		response.Message = "Message sent successfully"
		return response, nil
	}

	channel, err := rmq.getConnection().Channel()
	if err != nil {
		response.Status = 500
		response.Message = err.Error()
		return response, fmt.Errorf("failed to open a channel in rabbitmq")
	}
	defer channel.Close()

	messages, err := channel.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		response.Status = 500
		response.Message = err.Error()
		return response, fmt.Errorf("failed to consume from queue")
	}

	rabbitTimeout := time.After(10 * time.Second)
	for {
		select {
		case <-rabbitTimeout:
			{
				response.Status = 500
				response.Message = "RPC timeout"
				return response, fmt.Errorf("RPC timeout")
			}
		case msg := <-messages:
			{
				if msg.CorrelationId == corrId {
					var responseRabbit map[string]interface{}
					if err := json.Unmarshal(msg.Body, &responseRabbit); err != nil {
						response.Status = 500
						response.Message = err.Error()
						return response, fmt.Errorf("failed to unmarshal response")
					}

					if val, ok := responseRabbit["Status"]; ok {
						response.Status = int(val.(float64))
					}

					if val, ok := responseRabbit["StatusCode"]; ok {
						response.Status = int(val.(float64))
					}

					if val, ok := responseRabbit["StatusMessage"]; ok {
						response.Message = val.(string)
					}

					if val, ok := responseRabbit["Message"]; ok {
						response.Message = val.(string)
					}

					response.Data = responseRabbit
					return response, nil
				}
			}
		}
	}
}
