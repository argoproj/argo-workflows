package rediscl

import (
	"applatix.io/axerror"
	"encoding/json"
	"gopkg.in/redis.v5"
	"time"
)

type RedisClient struct {
	address     string
	password    string
	database    int
	redisClient *redis.Client
}

func NewRedisClient(address, password string, database int) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password, // no password set
		DB:       database, // use default DB
	})
	return &RedisClient{address, password, database, client}
}

func (c *RedisClient) GetString(key string) (string, *axerror.AXError) {
	result := c.redisClient.Get(key)
	if result == nil {
		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := result.Err()
	if err != nil {
		if err.Error() == "redis: nil" {
			return "", nil
		}

		return "", axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return result.Val(), nil
}

func (c *RedisClient) Set(key string, value string) *axerror.AXError {
	status := c.redisClient.Set(key, value, time.Duration(0))
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) SetWithTTL(key string, value interface{}, expiration time.Duration) *axerror.AXError {
	status := c.redisClient.Set(key, value, expiration)
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) Del(keys ...string) *axerror.AXError {
	status := c.redisClient.Del(keys...)
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) RPush(key string, values ...interface{}) *axerror.AXError {
	status := c.redisClient.RPush(key, values...)
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) RPushWithTTL(key string, values interface{}, expiration time.Duration) *axerror.AXError {
	if axErr := c.RPush(key, values); axErr != nil {
		return axErr
	}

	status := c.redisClient.Expire(key, expiration)
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) BRPopWithTTL(timeout time.Duration, keys ...string) ([]string, *axerror.AXError) {
	status := c.redisClient.BRPop(timeout, keys...)
	if status == nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := status.Err()
	if err != nil {
		return nil, axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}
	return status.Val(), nil
}

func (c *RedisClient) SetObj(key string, value interface{}) *axerror.AXError {

	objBytes, err := json.Marshal(value)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	status := c.redisClient.Set(key, string(objBytes), time.Duration(0))
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err = status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) SetObjWithTTL(key string, value interface{}, expiration time.Duration) *axerror.AXError {

	objBytes, err := json.Marshal(value)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	status := c.redisClient.Set(key, string(objBytes), expiration)
	if status == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err = status.Err()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	return nil
}

func (c *RedisClient) GetObj(key string, value interface{}) *axerror.AXError {
	result := c.redisClient.Get(key)
	if result == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	err := result.Err()
	if err != nil {
		if err.Error() == "redis: nil" {
			return axerror.ERR_API_RESOURCE_NOT_FOUND.New()
		}
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	objBytes, err := result.Bytes()
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}

	err = json.Unmarshal(objBytes, value)
	if err != nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage(err.Error())
	}
	return nil
}

func (c *RedisClient) FlushDB() *axerror.AXError {
	result := c.redisClient.FlushDb()
	if result == nil {
		return axerror.ERR_API_INTERNAL_ERROR.NewWithMessage("Cannot connect with redis.")
	}

	return nil
}
