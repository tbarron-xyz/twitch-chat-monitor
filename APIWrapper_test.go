package main

import (
	"fmt"
	"testing"
)

func TestAbs(t *testing.T) {
	var config = loadConfig()
	httpService := &ApiWrapper{TwitchApiClientId: config.TwitchApiClientId}
	var emotes, _, err = httpService.GetEmotes()
	if err != nil {
		t.Fail()
	}
	if len(emotes) == 0 {
		t.Fail()
	}
	for _, emote := range emotes {
		fmt.Println(emote)
	}
}
