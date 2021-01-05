package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	irc "github.com/tbarron-xyz/go-ircevent"
)

type GlobalState struct {
	emotes                []string
	emotesContainingKappa []string // emotes containing Kappa as a substring, not counting "Kappa" itself
	channels              []string
	redisclient           *RedisWrapper
	ircclient             *irc.Connection
	snapsCollection       DatabaseWrapper
	config                *Config
	httpService           *ApiWrapper
	MessagesLast5Minutes  int
}

func MakeGlobalState(config *Config, redis *RedisWrapper, snaps DatabaseWrapper) *GlobalState {
	httpService := &ApiWrapper{TwitchApiClientId: config.TwitchApiClientId}
	state := &GlobalState{config: config, httpService: httpService, snapsCollection: snaps, redisclient: redis}
	return state
}

func (state *GlobalState) Start() {
	state.updateEmotes()
	go state.updateUsersLoop()
	state.cloneCacheToStorageLoop()
	state.setServerStartupStats()
	state.ircSetupAndLoop(state.config.TwitchNick, state.config.TwitchOauthPassword)
}

func (state *GlobalState) setServerStartupStats() {
	state.redisclient.Do("SET", "serverStartTime", time.Now().Unix())
	state.redisclient.Do("SET", "goVersion", runtime.Version())
}

//
// func (state *GlobalState) serverStatsLoop() {
//     state.redisclient.Do("SET", "miscstats", map[string]string{
//         ""
//     })
// }

func (state *GlobalState) ircSetupAndLoop(twitchNick, startIrcLoop string) {
	state.ircclient = irc.IRC(twitchNick, twitchNick)
	state.ircclient.Password = startIrcLoop
	state.ircclient.ConnectCallback = state.ircConnectCallback // triggers on successful connection
	state.ircclient.AddCallback("PRIVMSG", state.ircPrivmsgCallback)
	state.ircclient.Connect("irc.twitch.tv:6667") //Connect to server
	state.ircclient.Loop()
}

func (state *GlobalState) ircPrivmsgCallback(ircevent *irc.Event) {
	event := *ircevent
	channel := event.Arguments[0]
	msg := event.Message()

	var emoteCounts = map[string]int{}

	for _, e := range state.emotes {
		ecount := strings.Count(msg, e)
		if e == "Kappa" { // Kappa gets overcounted due to being a substring of some other emotes
			for _, k := range state.emotesContainingKappa {
				ecount -= strings.Count(msg, k)
			}
		}
		emoteCounts[e] = ecount
	}

	state.incrementCache(channel, emoteCounts)
}

func (state *GlobalState) ircConnectCallback() {
	state.channels = []string{}
	state.updateChannelsLoop()
}

func (state *GlobalState) updateChannelsLoop() {
	defer time.AfterFunc(1*time.Minute, state.updateChannelsLoop)
	var newChannels, err = state.httpService.GetChannels()
	if err != nil {
		fmt.Println("Failed to get channels at " + time.Now().String())
		return
	}

	var actualChannelList = state.joinAndPartChannels(newChannels)
	actualChannelListJson, err := json.Marshal(actualChannelList)
	EH(err)

	state.redisclient.Do("SET", "channels", actualChannelListJson)
}

func (state *GlobalState) cloneCacheToStorageLoop() {
	fmt.Println("Cloning cache to storage.")
	defer time.AfterFunc(5*time.Minute, state.cloneCacheToStorageLoop)
	res, err := redis.IntMap(state.redisclient.Do("HGETALL", "curEmoteCountOverall")) // res is a map[string]int
	if err != nil {
		fmt.Println("Failed to HGETALL curEmoteCountOverall at " + time.Now().String())
		return
	}
	snap := snapshot{int64(time.Now().Unix()), res}
	err = state.snapsCollection.Insert(snap)
	if err != nil {
		fmt.Println("Failed to insert snapshot to database at "+time.Now().String(), err)
		return
	}
}

func (state *GlobalState) joinAndPartChannels(top25Channels []string) []string {
	var ratelimit = 50

	var tojoin []string
	oldchannels := state.channels
	channelsmap := mapfromarray(oldchannels)
	for _, c := range top25Channels {
		_, contains := channelsmap[c]
		if !contains {
			tojoin = append(tojoin, c)
		}
	}
	var topart []string
	topsmap := mapfromarray(top25Channels)
	for _, c := range oldchannels {
		_, contains := topsmap[c]
		if !contains {
			topart = append(topart, c)
		}
	}

	max := len(tojoin)
	if max > ratelimit {
		max = ratelimit
	}
	tojoin = tojoin[:max] // twitch ratelimits channel joins

	for _, c := range tojoin {
		state.ircclient.Join(c)
		state.channels = append(state.channels, c)
	}
	for _, c := range topart {
		state.ircclient.Part(c)
		state.channels = remove(state.channels, c)
	}
	return state.channels
}

func (state *GlobalState) updateUsersLoop() {
	defer time.AfterFunc(time.Minute, state.updateUsersLoop) // 1 minute
	for _, channelName := range state.channels {
		count := state.httpService.GetUserCount(channelName)
		state.redisclient.Do("SET", "curUsers:"+channelName, count)
		state.redisclient.Do("EXPIRE", "curUsers:"+channelName, 600) // 10 minutes
	}
}

func (state *GlobalState) updateEmotes() {
	var emotes, emotesContainingKappa, _ = state.httpService.GetEmotes()
	fmt.Println("Emotes:", emotes)
	fmt.Println("Emotes containing Kappa: ", emotesContainingKappa)
	state.emotes = emotes
	state.emotesContainingKappa = emotesContainingKappa
	for e, _ := range emotes {
		state.redisclient.Do("HSET", "curEmoteCountOverall", e, 0)
	}
	emotesJson, _ := json.Marshal(emotes)
	state.redisclient.Do("SET", "emotes", emotesJson)
	state.redisclient.Do("SET", "emoticons", emotesJson)
	state.redisclient.Do("SET", "kappaEmotes", emotesContainingKappa)
}

func (state *GlobalState) incrementCache(channel string, emoteCounts map[string]int) {
	for emote, count := range emoteCounts {

		if count > 0 {
			state.redisclient.Do("HINCRBY", "curEmoteCountOverall", emote, count)
			state.redisclient.PublishAfterFunc(5, "HINCRBY", "curEmoteCountOverall", emote, -1*count)

			state.redisclient.Do("HINCRBY", "curEmoteCountByChannel:"+emote, channel, count)
			state.redisclient.PublishAfterFunc(5, "HINCRBY", "curEmoteCountByChannel:"+emote, channel, -1*count)
		}
	}

	state.MessagesLast5Minutes += 1
	var intervalBetweenCacheUpdates = 10
	var m = state.MessagesLast5Minutes
	if m%intervalBetweenCacheUpdates == 0 {
		state.redisclient.Do("INCRBY", "totalMessagesSinceStart", m)
		state.redisclient.Do("INCRBY", "messagesLast5Minutes", m)
		state.redisclient.PublishAfterFunc(5, "INCRBY", "messagesLast5Minutes", -1*m)
		state.MessagesLast5Minutes = 0
	}
}
