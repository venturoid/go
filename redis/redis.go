package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client          *redis.Client
	logger          bool
	defaultDuration *time.Duration
	ctx             context.Context
}

// CreateConnection is a function to create connection to redis
// it will return redis client and error if any
func CreateConnection(options Options, ctx context.Context) (*Redis, error) {
	/* validate request */
	if err := options.Validate(); err != nil {
		return nil, err
	}

	if ctx == nil {
		return nil, errors.New("context is required")
	}

	/* create connection */
	opts := options.RedisOptions()
	client := redis.NewClient(opts)

	/* check connection */
	if client == nil {
		log.Println("client connection is nil")
		return nil, errors.New("error create connection")
	}

	if data, err := client.Ping(ctx).Result(); err != nil {
		log.Println("error connect to redis :", err.Error())
		return nil, errors.New("error connect to redis")
	} else {
		log.Println("successfully connect to redis :", data)
	}

	return &Redis{
		client:          client,
		ctx:             ctx,
		logger:          options.Logger,
		defaultDuration: options.DefaultDuration,
	}, nil
}

// Close is a function to close connection to redis
func (r *Redis) Close() error {
	if r.client == nil {
		return errors.New("client connection is empty")
	}

	if err := r.client.Close(); err != nil {
		log.Println("error close connection :", err.Error())
		return errors.New("error close connection")
	}

	log.Println("successfully close connection")
	return nil
}

// Get is a function to get data from redis
func (r *Redis) Get(key string) (result *string, err error) {
	if res := r.client.Get(r.ctx, key); res.Err() != nil {
		if res.Err().Error() != "redis: nil" {
			log.Println("error fetch data from redis : ", res.Err().Error())
			return nil, errors.New("error fetch data from redis")
		}

		return nil, res.Err()
	} else {
		if data, err := res.Result(); err != nil {
			if r.logger {
				log.Println("error get result redis ; ", err.Error())
			}
			return nil, errors.New("error get redis result")
		} else {
			if r.logger {
				log.Println("successfully fetch data from redis : ", data)
			}
			return &data, nil
		}
	}
}

func (r *Redis) GetWithBind(key string, result interface{}) (err error) {
	/* get redis data */
	var redisdata string
	if res := r.client.Get(r.ctx, key); res.Err() != nil {
		if res.Err().Error() != "redis: nil" {
			log.Println("error fetch data from redis : ", res.Err().Error())
		}
		return res.Err()
	} else {
		if data, err := res.Result(); err != nil {
			if r.logger {
				log.Println("error get result redis ; ", err.Error())
			}
			return errors.New("error get redis result")
		} else {
			if r.logger {
				log.Println("successfully fetch data from redis : ", data)
			}
			redisdata = data
		}
	}

	/* check redis data */
	if redisdata == "" {
		log.Println("data is empty")
		return errors.New("data is empty")
	}

	/* bind data */
	if err := json.Unmarshal([]byte(redisdata), &result); err != nil {
		log.Println("error unmarshal data : ", err.Error())
		return errors.New("error unmarshal data")
	}

	return nil
}

// Set is a function to set data to redis within specific duration
// if duration is nil, it will set data to 24 hours
func (r *Redis) Set(key string, value interface{}, duration *time.Duration) error {
	data := value
	if value != nil {
		if reflect.TypeOf(value).Kind() == reflect.Array || reflect.TypeOf(value).Kind() == reflect.Slice || reflect.TypeOf(value).Kind() == reflect.Map || reflect.TypeOf(value).Kind() == reflect.Struct {
			if res, err := json.Marshal(value); err != nil {
				log.Println("error marshal data : ", err.Error())
				return errors.New("error marshal data")
			} else {
				data = string(res)
			}
		}
	}

	/* if duration is nil, set data to default duration */
	if duration == nil || *duration == 0 {
		duration = r.defaultDuration

		if r.defaultDuration == nil {
			duration = new(time.Duration)
		}
	}

	/* set data to redis */
	if res := r.client.Set(r.ctx, key, data, *duration); res.Err() != nil {
		log.Println("error set data to redis : ", res.Err().Error())
		return res.Err()
	}

	if r.logger {
		log.Println("successfully set data " + key + " to redis")
	}
	return nil
}

// Delete is a function to delete data from redis
func (r *Redis) Delete(key string) error {
	if res := r.client.Del(r.ctx, key); res.Err() != nil {
		log.Println("error delete data from redis : ", res.Err().Error())
		return errors.New("error delete data from redis")
	}

	if r.logger {
		log.Println("successfully delete data " + key + " from redis")
	}
	return nil
}

// CheckKey is a function to check whether all keys is exist in redis
func (r *Redis) CheckKey(keys []string) (bool, error) {
	log.Println("check key redis : ", keys)
	if res := r.client.Exists(r.ctx, keys...); res.Err() != nil {
		log.Println("error check key exists on redis : ", res.Err().Error())
		return false, errors.New("error check key exists on redis")
	} else {
		if r.logger {
			log.Println("successfully check key exsist from redis : ", res.Val())
		}

		return len(keys) == int(res.Val()), nil
	}
}

// Delete multiple is a function to delete multiple data from redis
func (r *Redis) DeleteMultiple(keys []string) (int, error) {
	log.Println("delete multiple key redis : ", keys)
	if res := r.client.Del(r.ctx, keys...); res.Err() != nil {
		log.Println("error delete data from redis : ", res.Err().Error())
		return 0, errors.New("error delete data from redis")
	} else {
		if r.logger {
			log.Printf("successfully delete data (%+v) from redis", keys)
		}

		return int(res.Val()), nil
	}
}
