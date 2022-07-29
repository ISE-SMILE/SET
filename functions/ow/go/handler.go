package main

import (
	"encoding/json"
	"ow/bencher"

	factc "github.com/faas-facts/fact-go-client"
)

var client *factc.FactClient

func init() {
	client = &factc.FactClient{}

	platformHint := "OW"

	client.Boot(factc.FactClientConfig{
		SendOnUpdate:       true,
		IncludeEnvironment: false,
		IOArgs:             map[string]string{},
		Platform:           &platformHint,
	})
}

// Main forwading to Hello
func Main(args map[string]interface{}) map[string]interface{} {
	delay := 10
	//TODO: convert input to job
	job := bencher.Job{
		Idle: &delay,
	}

	trace := bencher.Handle(client, job, nil)

	data, err := json.Marshal(&trace)
	if err != nil {
		panic(err)
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)

	if err != nil {
		panic(err)
	}

	return result
}
