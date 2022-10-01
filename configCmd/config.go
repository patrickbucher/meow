package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/patrickbucher/meow"
)

type Config []*meow.Endpoint

func main() {
	addr := flag.String("addr", "0.0.0.0", "listen to address")
	port := flag.Uint("port", 8000, "listen on port")
	file := flag.String("file", "config.csv", "CSV file to store the configuration")
	flag.Parse()

	log.SetOutput(os.Stderr)

	config := mustReadConfig(*file)
	fmt.Println(config)

	listenTo := fmt.Sprintf("%s:%d", *addr, *port)
	log.Printf("listen to %s", listenTo)
	http.ListenAndServe(listenTo, nil)
}

func mustReadConfig(configPath string) Config {
	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		// just start with an empty config
		log.Printf(`the config file "%s" does not exist`, configPath)
		return Config{}
	}

	var config Config
	reader := csv.NewReader(file)
	defer file.Close()
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("the config file '%s' is malformed: %v", configPath, err)
	}
	for i, line := range records {
		endpoint, err := meow.EndpointFrom(line)
		if err != nil {
			log.Fatalf(`line %d: "%s": %v`, i, line, err)
		}
		config = append(config, endpoint)
	}
	return config
}
