package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/patrickbucher/meow"
)

// Config maps the identifiers to endpoints.
type Config map[string]*meow.Endpoint

// ConcurrentConfig wraps the config together with a mutex.
type ConcurrentConfig struct {
	mu     sync.RWMutex
	config Config
}

var cfg ConcurrentConfig

func main() {
	addr := flag.String("addr", "0.0.0.0", "listen to address")
	port := flag.Uint("port", 8000, "listen on port")
	file := flag.String("file", "config.csv", "CSV file to store the configuration")
	flag.Parse()

	log.SetOutput(os.Stderr)

	cfg.config = mustReadConfig(*file)

	http.HandleFunc("/endpoints/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getEndpoint(w, r)
		case http.MethodPost:
			postEndpoint(w, r, *file)
		// TODO: support http.MethodDelete to delete endpoints
		default:
			log.Printf("request from %s rejected: method %s not allowed",
				r.RemoteAddr, r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/endpoints", func(w http.ResponseWriter, r *http.Request) {
		getEndpoints(w, r)
	})

	listenTo := fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("listen to %s", listenTo)
	http.ListenAndServe(listenTo, nil)
}

func getEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET %s from %s", r.URL, r.RemoteAddr)
	identifier, err := extractEndpointIdentifier(r.URL.String())
	if err != nil {
		log.Printf("extract endpoint identifier of %s: %v", r.URL, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cfg.mu.RLock()
	endpoint, ok := cfg.config[identifier]
	cfg.mu.RUnlock()
	if ok {
		payload, err := endpoint.JSON()
		if err != nil {
			log.Printf("convert %v to JSON: %v", endpoint, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(payload)
	} else {
		log.Printf(`no such endpoint "%s"`, identifier)
		w.WriteHeader(http.StatusNotFound)
	}
}

func postEndpoint(w http.ResponseWriter, r *http.Request, file string) {
	log.Printf("POST %s from %s", r.URL, r.RemoteAddr)
	identifier, err := extractEndpointIdentifier(r.URL.String())
	if err != nil {
		log.Printf("extract endpoint identifier of %s: %v", r.URL, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	buf := bytes.NewBufferString("")
	io.Copy(buf, r.Body)
	defer r.Body.Close()
	endpoint, err := meow.EndpointFromJSON(buf.String())
	if err != nil {
		log.Printf("parse JSON body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if endpoint.Identifier != identifier {
		log.Printf("identifier mismatch: (ressource: %s, body: %s)",
			identifier, endpoint.Identifier)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cfg.mu.Lock()
	_, ok := cfg.config[identifier]
	var status int
	if ok {
		status = http.StatusNoContent
	} else {
		status = http.StatusCreated
	}
	cfg.config[identifier] = endpoint
	if err := writeConfig(cfg.config, file); err != nil {
		status = http.StatusInternalServerError
	}
	cfg.mu.Unlock()
	w.WriteHeader(status)
}

func getEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("request from %s rejected: method %s not allowed",
			r.RemoteAddr, r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	log.Printf("GET %s from %s", r.URL, r.RemoteAddr)
	payloads := make([]meow.EndpointPayload, 0)
	for _, endpoint := range cfg.config {
		payload := meow.EndpointPayload{
			Identifier:   endpoint.Identifier,
			URL:          endpoint.URL.String(),
			Method:       endpoint.Method,
			StatusOnline: endpoint.StatusOnline,
			Frequency:    endpoint.Frequency.String(),
			FailAfter:    endpoint.FailAfter,
		}
		payloads = append(payloads, payload)
	}
	data, err := json.Marshal(payloads)
	if err != nil {
		log.Printf("serialize payloads: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

const endpointIdentifierPatternRaw = "^/endpoints/([a-z][-a-z0-9]+)$"

var endpointIdentifierPattern = regexp.MustCompile(endpointIdentifierPatternRaw)

func extractEndpointIdentifier(endpoint string) (string, error) {
	matches := endpointIdentifierPattern.FindStringSubmatch(endpoint)
	if len(matches) == 0 {
		return "", fmt.Errorf(`endpoint "%s" does not match pattern "%s"`,
			endpoint, endpointIdentifierPatternRaw)
	}
	return matches[1], nil
}

func writeConfig(config Config, configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf(`open "%s" for write: %v`, configPath, err)
	}

	writer := csv.NewWriter(file)
	defer file.Close()
	for _, endpoint := range config {
		record := []string{
			endpoint.Identifier,
			endpoint.URL.String(),
			endpoint.Method,
			strconv.Itoa(int(endpoint.StatusOnline)),
			endpoint.Frequency.String(),
			strconv.Itoa(int(endpoint.FailAfter)),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf(`write endpoint "%s": %v`, endpoint, err)
		}
	}
	writer.Flush()
	return nil
}

func mustReadConfig(configPath string) Config {
	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		// just start with an empty config
		log.Printf(`the config file "%s" does not exist`, configPath)
		return Config{}
	}

	config := make(Config, 0)
	reader := csv.NewReader(file)
	defer file.Close()
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("the config file '%s' is malformed: %v", configPath, err)
	}
	for i, line := range records {
		endpoint, err := meow.EndpointFromRecord(line)
		if err != nil {
			log.Fatalf(`line %d: "%s": %v`, i, line, err)
		}
		config[endpoint.Identifier] = endpoint
	}
	return config
}
