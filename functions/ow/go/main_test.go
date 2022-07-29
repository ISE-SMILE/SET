package main

import (
	factc "github.com/faas-facts/fact-go-client"
	"math/rand"
	"os"
	"ow/bencher"
	"testing"
)

func TestPrime_OpenWhisk(t *testing.T) {
	os.Setenv("__OW_ACTION_NAME", "Foo")
	client := &factc.FactClient{}

	platformHint := "OW"

	client.Boot(factc.FactClientConfig{
		SendOnUpdate:       true,
		IncludeEnvironment: false,
		IOArgs:             map[string]string{},
		Platform:           &platformHint,
	})

	p := uint32((rand.Int63()*rand.Int63() + 1) + (rand.Int63()*rand.Int63() + 1))
	//TODO: convert input to job
	job := bencher.Job{
		Prime: &p,
	}

	trace := bencher.Handle(client, job, nil)
	t.Log(trace)

}
