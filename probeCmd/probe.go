package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/patrickbucher/meow"
)

func main() {
	configURL, ok := os.LookupEnv("CONFIG_URL")
	if !ok {
		fmt.Fprintln(os.Stderr, "environment variable CONFIG_URL must be set")
		os.Exit(1)
	}
	endpoints := mustFetchEndpoints(configURL)

	logFileName := fmt.Sprintf("meow-%v.log", time.Now().Format("2006-01-02T15-04-05"))
	logFilePath := strings.Join([]string{os.TempDir(), logFileName}, string(os.PathSeparator))
	logFile, err := meow.NewLogFile(logFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open log file %s: %v\n\n", logFilePath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "started logging to %s\n", logFilePath)

	go monitor(endpoints, logFile)

	done := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-signals
		fmt.Fprintf(os.Stderr, "signal %v received\n", s)
		logFile.Close()
		// TODO: now it would be a good time to archive logFilePath to S3
		done <- struct{}{}
	}()

	<-done
}

func monitor(endpoints []meow.Endpoint, logger *meow.LogFile) {
	probe := func(e meow.Endpoint, messages chan string) {
		messages <- fmt.Sprintf("started probing %s every %v", e.Identifier, e.Frequency)
		freq := time.NewTicker(e.Frequency)
		errorCount := 0
		lastStateOK := false
		firstTry := true
		alerted := false
		for {
			start := time.Now()
			status, err := requestForStatus(e)
			if err != nil {
				// TODO: adjust log format
				messages <- fmt.Sprintf("%c request failed: %v", meow.CrossMark, err)
			}
			end := time.Now()
			duration := end.Sub(start)
			stateOK := status == int(e.StatusOnline)
			if stateOK {
				if lastStateOK || firstTry {
					// TODO: adjust log format
					messages <- fmt.Sprintf("%c %s is online (took %v)",
						meow.CatAvailable, e.Identifier, duration)
				} else {
					// TODO: adjust log format
					messages <- fmt.Sprintf("%c %s is online again (took %v)",
						meow.CatAvailableAgain, e.Identifier, duration)
				}
				lastStateOK = true
				errorCount = 0
				alerted = false
			} else {
				errorCount++
				// TODO: adjust log format
				messages <- fmt.Sprintf("%c %s is not online (%d times)",
					meow.CatUnavailable, e.Identifier, errorCount)
				if errorCount >= int(e.FailAfter) && !alerted {
					// TODO: adjust log format
					messages <- fmt.Sprintf("%c ALERT: %s is offline (%d failed attempts)",
						meow.CatAlert, e.Identifier, e.FailAfter)
					alerted = true
				}
				lastStateOK = false
			}
			firstTry = false
			<-freq.C
		}
	}
	messages := make(chan string)
	for _, endpoint := range endpoints {
		go probe(endpoint, messages)
	}
	for logMessage := range messages {
		fmt.Fprintln(os.Stderr, logMessage)
		logger.WriteLine(logMessage)
	}
}

func requestForStatus(e meow.Endpoint) (int, error) {
	req, err := http.NewRequest(e.Method, e.URL.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("prepare request: %s %s %s: %v", e.Identifier, e.Method, e.URL, err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("perform request %s %s %s: %v", e.Identifier, e.Method, e.URL, err)
	}
	defer res.Body.Close()
	return res.StatusCode, nil
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
