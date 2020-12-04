package main

import (
	"net/http"
	_ "net/http/pprof"
)

// "time"
// "sync"
// "runtime"
// "net/http"
// "encoding/json"
//"github.com/thoj/go-ircevent"
// "github.com/garyburd/redigo/redis"

func main() {
	var config = loadConfig()
	var redisclient = MakeRedisWrapper()
	// var db = mongoDatabase()
	var db = dynamodbDatabase()
	var globalState = MakeGlobalState(config, redisclient, db)
	go http.ListenAndServe("localhost:6060", nil)
	globalState.Start()
}
