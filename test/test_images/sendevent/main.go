/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Implements a simple utility for sending a JSON-encoded sample event.
//package main
//
//import (
//	"flag"
//	"fmt"
//	"github.com/knative/eventing/test"
//	"io/ioutil"
//	"net/http"
//	"os"
//	"strconv"
//	"time"
//
//	"github.com/knative/pkg/cloudevents"
//
//	"github.com/google/uuid"
//)
//
//var (
//	context cloudevents.EventContext
//	webhook string
//	data    string
//)
//
//func init() {
//	flag.StringVar(&context.EventID, "event-id", "", "Event ID to use. Defaults to a generated UUID")
//	flag.StringVar(&context.EventType, "event-type", "google.events.action.demo", "The Event Type to use.")
//	flag.StringVar(&context.Source, "source", "", "Source URI to use. Defaults to the current machine's hostname")
//	flag.StringVar(&data, "data", `{"hello": "world!"}`, "Event data")
//}
//
//func main() {
//	flag.Parse()
//
//	if len(flag.Args()) != 1 {
//		fmt.Println("Usage: sendevent [flags] <webhook>\nFor details about valid flags, run sendevent --help")
//		os.Exit(1)
//	}
//
//	fmt.Println("Sleeping for 5 seconds")
//	//TODO maybe required for istio warmup?
//	time.Sleep(5 * time.Second)
//	fmt.Println("Done sleeping")
//
//	webhook := flag.Arg(0)
//
//	//var untyped map[string]string
//	//if err := json.Unmarshal([]byte(data), &untyped); err != nil {
//	//	fmt.Println("Currently sendevent only supports JSON event data")
//	//	os.Exit(1)
//	//}
//
//	hb := &test.Heartbeat{
//		Sequence: 0,
//		Label:    "hello",
//	}
//
//	//fillEventContext(&context)
//
//	//req, err := cloudevents.NewRequest(webhook, hb, context)
//	req, err := cloudevents.Binary.NewRequest(webhook, hb, context)
//	if err != nil {
//		fmt.Printf("Failed to create request: %s", err)
//		os.Exit(1)
//	}
//	fmt.Printf("requesting: %#v", req)
//	resp, err := http.DefaultClient.Do(req)
//	defer resp.Body.Close()
//	if err != nil {
//		fmt.Printf("Failed to send event to %s: %s\n", webhook, err)
//		os.Exit(1)
//	}
//	fmt.Printf("Got response from %s\n%s\n", webhook, resp.Status)
//	if resp.Header.Get("Content-Length") != "" {
//		bytes, _ := ioutil.ReadAll(resp.Body)
//		fmt.Println(string(bytes))
//	}
//}
//
//// Creates a CloudEvent Context
//func cloudEventsContext() *cloudevents.EventContext {
//	return &cloudevents.EventContext{
//		CloudEventsVersion: cloudevents.CloudEventsVersion,
//		EventType:          "dev.knative.eventing.e2e.heartbeats",
//		EventID:            uuid.New().String(),
//		Source:             "e2etest-knative-eventing",
//		EventTime:          time.Now(),
//	}
//}

package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/knative/pkg/cloudevents"
)

type Heartbeat struct {
	Sequence int    `json:"id"`
	Data     string `json:"data"`
}

var (
	sink      string
	data      string
	eventID   string
	eventType string
	source    string
	periodStr string
)

func init() {
	flag.StringVar(&sink, "sink", "", "the host url to heartbeat to")
	flag.StringVar(&data, "data", "", "special data")
	flag.StringVar(&eventID, "event-id", "", "Event ID to use. Defaults to a generated UUID")
	flag.StringVar(&eventType, "event-type", "knative.eventing.test.e2e", "The Event Type to use.")
	flag.StringVar(&source, "source", "localhost", "Source URI to use. Defaults to the current machine's hostname")
	flag.StringVar(&periodStr, "period", "5", "the number of seconds between messages")
}

func main() {
	flag.Parse()
	var period time.Duration
	if p, err := strconv.Atoi(periodStr); err != nil {
		period = time.Duration(5) * time.Second
	} else {
		period = time.Duration(p) * time.Second
	}

	if eventID == "" {
		eventID = uuid.New().String()
	}

	if source == "" {
		source = "localhost"
	}

	hb := &Heartbeat{
		Sequence: 0,
		Data:     data,
	}
	ticker := time.NewTicker(period)
	for {
		hb.Sequence++
		postMessage(sink, hb)
		// Wait for next tick
		<-ticker.C
	}
}

// Creates a CloudEvent Context for a given heartbeat.
func cloudEventsContext() *cloudevents.EventContext {
	return &cloudevents.EventContext{
		CloudEventsVersion: cloudevents.CloudEventsVersion,
		EventType:          eventType,
		EventID:            eventID,
		Source:             source,
		EventTime:          time.Now(),
	}
}

func postMessage(target string, hb *Heartbeat) error {
	ctx := cloudEventsContext()

	log.Printf("posting to %q, %d", target, hb.Sequence)
	// Explicitly using Binary encoding so that Istio, et. al. can better inspect
	// event metadata.
	req, err := cloudevents.Binary.NewRequest(target, hb, *ctx)
	if err != nil {
		log.Printf("failed to create http request: %s", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to do POST: %v", err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("response Status: %s", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("response Body: %s", string(body))
	return nil
}
