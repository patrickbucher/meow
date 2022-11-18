package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/patrickbucher/meow"
)

func main() {
	configURL, ok := os.LookupEnv("CONFIG_URL")
	if !ok {
		log.Fatalln("environment variable CONFIG_URL must be set")
	}
	fmt.Println(mustFetchEndpoints(configURL))
}

func mustFetchEndpoints(configURL string) []meow.Endpoint {
	endpoints := make([]meow.Endpoint, 0)
	configEndpoint := fmt.Sprintf("%s/endpoints", configURL)
	res, err := http.Get(configEndpoint)
	if err != nil {
		log.Fatalf("fetch endpoints from %s: %v", configEndpoint, err)
	}
	defer res.Body.Close()
	payloads := make([]meow.EndpointPayload, 0)
	buf := bytes.NewBufferString("")
	if _, err := io.Copy(buf, res.Body); err != nil {
		log.Fatalf("copy body from result of %s: %v", configEndpoint, err)
	}
	if err := json.Unmarshal(buf.Bytes(), &payloads); err != nil {
		log.Fatalf("unmarshal JSON payload: %v", err)
	}
	for _, payload := range payloads {
		endpoint, err := meow.EndpointFromPayload(payload)
		if err != nil {
			log.Fatalf("convert payload %v to endpoint: %v", payload, err)
		}
		endpoints = append(endpoints, *endpoint)
	}
	return endpoints
}
