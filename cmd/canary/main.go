package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	bind := flag.String("bind", "0.0.0.0", "bind to")
	port := flag.Int("port", 9000, "port number")
	flag.Parse()
	http.HandleFunc("/canary", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(os.Stderr, "request from %s\n", r.RemoteAddr)
		w.Write([]byte("OK\n"))
	})
	listenTo := fmt.Sprintf("%s:%d", *bind, *port)
	fmt.Fprintf(os.Stderr, "listen to %s\n", listenTo)
	http.ListenAndServe(listenTo, nil)
}
