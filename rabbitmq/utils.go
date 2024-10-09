package rabbitmq

import (
	"encoding/json"
	"errors"
	"log"
)

func buildDNS(config Config) (*string, error) {
	if config.RabbitMQConfig.Host == "" {
		return nil, errors.New("rabbitmq host is required")
	}

	if config.RabbitMQConfig.Port == "" {
		config.RabbitMQConfig.Port = "5672"
	}

	if config.RabbitMQConfig.Username == "" {
		return nil, errors.New("rabbitmq username is required")
	}

	if config.RabbitMQConfig.Password == "" {
		return nil, errors.New("rabbitmq password is required")
	}

	if config.RabbitMQConfig.VirtualHost == "" {
		return nil, errors.New("rabbitmq virtual host is required")
	}

	url := "amqp://" + config.RabbitMQConfig.Username + ":" + config.RabbitMQConfig.Password + "@" + config.RabbitMQConfig.Host + ":" + config.RabbitMQConfig.Port + config.RabbitMQConfig.VirtualHost

	return &url, nil
}

func JSONEncode(data interface{}) string {
	byteData, err := json.Marshal(data)
	if err != nil {
		log.Println("error on encoding data to json : ", err.Error())
		return ""
	}
	return string(byteData)
}
