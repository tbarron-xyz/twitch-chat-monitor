package main

import (
	"encoding/json"
	"sync"

	"github.com/garyburd/redigo/redis"
)

type RedisWrapper struct {
	redisConn redis.Conn
	mutex     *sync.Mutex
}

func MakeRedisWrapper() *RedisWrapper {
	var redisclient_, err = redis.Dial("tcp", ":6379") // default settings - note the underscore
	if err != nil {
		panic("Could not connect to redis")
	}
	var redisclient = &RedisWrapper{redisclient_, &sync.Mutex{}}
	return redisclient
}

func (r *RedisWrapper) Do(cmd string, args ...interface{}) (interface{}, error) { // wrapper for syncing
	r.mutex.Lock()
	response, err := r.redisConn.Do(cmd, args...)
	r.mutex.Unlock()
	return response, err
}

type AfterFuncData struct {
	Minutes int
	Cmd     string
	Args    []interface{}
	//	ID int
}

func (redisclient *RedisWrapper) PublishAfterFunc(minutes int, cmd string, args ...interface{}) { // we're using a separate process, so that our AfterFunc's don't die when we restart this process
	data := AfterFuncData{minutes, cmd, args}
	j, err := json.Marshal(data)
	EH(err)
	redisclient.Do("RPUSH", "kappaafterfunc", j)
}
