package main

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
	// var snaps = mongoDatabase()
	var snaps = dynamodbDatabase()
	var globalState = MakeGlobalState(config, redisclient, snaps)
	globalState.Start()
}
