package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ApiWrapper struct {
	TwitchApiClientId string
}

func (this *ApiWrapper) getJsonWithClientId(url string, target interface{}) error { // target should be a pointer &
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Client-ID", this.TwitchApiClientId)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(target)
}

func (this *ApiWrapper) getJson(url string, target interface{}) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(target)
}

type EmotesResponse struct {
	Emotes map[string]interface{}
}

func (this *ApiWrapper) GetEmotes() (emotes, emotesContainingKappa []string, e error) {
	var emotesresponse map[string]interface{}
	url := "http://twitchemotes.com/api_cache/v3/global.json"
	var err = this.getJsonWithClientId(url, &emotesresponse)
	if err != nil {
		return nil, nil, err
	}
	emotes = []string{}
	for key, _ := range emotesresponse {
		emotes = append(emotes, key)
		if strings.Contains(key, "Kappa") && key != "Kappa" {
			emotesContainingKappa = append(emotesContainingKappa, key)
		}
	}

	return
}

type StreamsResponse struct {
	Streams []struct {
		Channel struct {
			Name string
		}
	}
}

func (this *ApiWrapper) GetChannels() ([]string, error) {
	url := "https://api.twitch.tv/kraken/streams"

	var streamsresponse StreamsResponse
	err := this.getJsonWithClientId(url, &streamsresponse)
	if err != nil {
		return nil, err
	}
	var top25Channels []string
	for _, stream := range streamsresponse.Streams {
		top25Channels = append(top25Channels, "#"+stream.Channel.Name)
	}

	return top25Channels, nil
}

type UsersResponse struct {
	Chatter_count int
}

func (this *ApiWrapper) GetUserCount(channelName string) int {
	var usersresponse UsersResponse
	if len(channelName) == 0 {
		pr("Empty channel: ", channelName)
		return 0
	} else {
		url := "http://tmi.twitch.tv/group/user/" + channelName[1:] + "/chatters"
		err := this.getJsonWithClientId(url, &usersresponse)
		if err != nil {
			fmt.Println("Failed to get user count for " + channelName)
		}
		return usersresponse.Chatter_count
	}
}
