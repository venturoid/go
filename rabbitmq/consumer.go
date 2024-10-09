package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/venturoid/go/redis"
)

// IRqAutoConsumer is interface defining method of rabbit mq auto connect for consumer
type IRqAutoConsumer interface {
	IRqAutoConnect
	GetMessageChanel(queue string) <-chan amqp.Delivery
	SetReadQueue(queue ...string)
	ConsumeMessage() (err error)
}

type rMqAutoConsumer struct {
	rMqAutoConnect
	deliveryCh     map[string]<-chan amqp.Delivery
	msgCh          map[string]chan amqp.Delivery
	ctxConsumeMsg  map[string]context.Context
	stopConsumeMsg map[string]context.CancelFunc
	isBroken       bool
	readQueue      []string
}

func (r *rMqAutoConsumer) Stop() {
	log.Printf("prepare to stop consumer %v\n", r.readQueue)

	for _, stopConsumeMsg := range r.stopConsumeMsg {
		stopConsumeMsg()
	}
	r.stop()
	log.Printf("stop consumer %v complete\n", r.readQueue)
}

func (r *rMqAutoConsumer) SetReadQueue(queue ...string) {
	r.readQueue = queue
}

func (r *rMqAutoConsumer) StartConnection(username, password, host, port, vhost string) (c *amqp.Connection, err error) {
	err = r.startConnection(username, password, host, port, vhost)
	if err != nil {
		log.Panicln(err.Error())
	}
	c = r.conn
	return
}

func (r *rMqAutoConsumer) listenQueueOnChannel() (err error) {
	// make sure that only one message at one time
	err = r.GetRqChannel().Qos(
		1,     //prefetch count
		0,     //prefetch size
		false, //global
	)
	if err != nil {
		r.ch.Close()
		r.conn.Close()
		log.Panicln(err.Error())
	}

	for _, readQueue := range r.readQueue {
		log.Printf("listen to queue %s\n", readQueue)
		r.deliveryCh[readQueue], err = r.ch.Consume(
			readQueue, //queue
			//config.RqNotifQueue(), //name
			"",    //consumer
			false, //auto ack
			false, //exclusive
			false, //no local
			false, //no wait
			nil,   //args
		)
		if err != nil {
			log.Panicln(err.Error())
		}
	}
	r.isBroken = false

	return
}

func (r *rMqAutoConsumer) ConsumeMessage() (err error) {
	r.deliveryCh = map[string]<-chan amqp.Delivery{}
	r.msgCh = map[string]chan amqp.Delivery{}
	r.ctxConsumeMsg = map[string]context.Context{}
	r.stopConsumeMsg = map[string]context.CancelFunc{}
	r.listenQueueOnChannel()
	for _, readQueue := range r.readQueue {
		log.Printf("create context with cancel, delivery chanel, message chanel on queue : %s\n", readQueue)
		// prepare context
		r.ctxConsumeMsg[readQueue], r.stopConsumeMsg[readQueue] = context.WithCancel(context.Background())
		// prepare chanel
		r.msgCh[readQueue] = make(chan amqp.Delivery)
		go func(readQueue string) {
			//defer close(r.msgCh[readQueue])
			for {
				if r.isBroken {
					<-time.After(time.Duration(1) * time.Second)
					continue
				}
				select {
				case <-r.ctxConsumeMsg[readQueue].Done():
					close(r.msgCh[readQueue])
					log.Printf("stop consuming message on %s\n", readQueue)
					return
				case <-time.After(time.Duration(1) * time.Second):
				case delivery := <-r.deliveryCh[readQueue]:
					log.Printf("receive new data \"%s\" on %s\n", delivery.Body, readQueue)
					if delivery.Body == nil {
						log.Println("queue may be not exist or server down")
						<-time.After(time.Duration(1) * time.Second)
					}
					r.msgCh[readQueue] <- delivery
				}
			}
		}(readQueue)
	}
	return
}

func (r *rMqAutoConsumer) GetMessageChanel(queue string) <-chan amqp.Delivery {
	return r.msgCh[queue]
}

func (r *rMqAutoConsumer) beforeReconnect() { // implement template pattern
	r.isBroken = true
}

func (r *rMqAutoConsumer) afterReconnect() { // implement template pattern
	r.listenQueueOnChannel()
}

// CreateRqConsumer is function to create rabbit mq auto connect for consumer
func CreateRqConsumer() (r IRqAutoConsumer) {
	rmq := new(rMqAutoConsumer)
	rmq.rq = rmq
	return rmq
}

// funciton auto connect rabbitmq queue
type Connection struct {
	Username string
	Password string
	Host     string
	Port     string
	Vhost    string
	Queue    string

	Redis *redis.Redis
}

func Consumer(conn Connection, handler func([]byte) string) error {
	log.Println("connect to rabbit mq")

	rbMq := CreateRqPubConsumer()
	rbMq.SetReadQueue(conn.Queue)
	if _, err := rbMq.StartConnection(conn.Username, conn.Password, conn.Host, conn.Port, conn.Vhost); err != nil {
		log.Println("error on connecting to rabbitmq", err.Error())
		return err
	}

	defer rbMq.Stop()

	if err := createQueue(rbMq, conn.Queue); err != nil {
		log.Printf("failed to declare a queue %s in rabbitmq: %v", conn.Queue, err.Error())
	}

	rbMq.ConsumeMessage()
	message := rbMq.GetMessageChanel(conn.Queue)
	forever := make(chan bool)
	defer close(forever)

	deliveredMsg := make(chan amqp.Delivery)
	defer close(deliveredMsg)

	for i := 0; i < 5; i++ {
		go func() {
			for data := range deliveredMsg {
				process := process{
					rbMq:    rbMq,
					data:    data,
					handler: handler,
					redis:   conn.Redis,
				}

				go processData(process)
			}
		}()
	}

	go func() {
		for data := range message {
			data.Ack(true)
			deliveredMsg <- data
		}
	}()

	log.Println("module is ready now")
	fmt.Println("")
	<-forever

	return nil
}

type process struct {
	rbMq    IRqAutoConnect
	data    amqp.Delivery
	handler func([]byte) string
	redis   *redis.Redis
}

func processData(config process) {
	log.Println("data.ReplyTo : ", config.data.ReplyTo)
	defer func() { recover() }()

	response := config.handler(config.data.Body)

	if config.data.ReplyTo == "redis" {
		key := config.data.Headers["redis-key"].(string)
		duration := 60 * time.Second
		config.redis.Set(key, response, &duration)

		log.Println("Success set response to redis with key:", key)
		log.Println("")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := config.rbMq.GetRqChannel().PublishWithContext(
		ctx,
		"",
		config.data.ReplyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: config.data.CorrelationId,
			Body:          []byte(response),
			Expiration:    "60000",
		},
	)
	if err != nil {
		log.Println("failed to publish a message to rabbitmq", err.Error())
	}

	log.Println("Success publish message reply")
	log.Println("")
}

func createQueue(rbMq IRqAutoPubConsumer, queueName string) (err error) {
	_, err = rbMq.GetRqChannel().QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	return
}
