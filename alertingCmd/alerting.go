package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

const (
	webhookPrefix = "https://hooks.slack.com/services"
)

type Message struct {
	Text string `json:"text"`
}

func main() {
	// TODO: read from env variable of flag
	webhookSuffix := "[webhook]/[suffix]"
	url := webhookPrefix + webhookSuffix
	message := Message{Text: "Hi there. There's something to alert you about."}
	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}
	buf := bytes.NewBufferString("")
	buf.Write(payload)
	res, err := http.Post(url, "application/json", buf)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode == http.StatusOK {
		log.Println("OK")
	}
}
