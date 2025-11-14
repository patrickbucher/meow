package main

import (
	"fmt"
	"time"
)

type LogMessage struct {
	Identifier    string
	URL           string
	At            time.Time
	Method        string
	StateExpected uint16
	StateResult   uint16
	Success       bool
	TookSecs      float64
}

func (l LogMessage) String() string {
	return fmt.Sprintf("accessed endpoint %s [URL: %s] at %v with HTTP method %s; "+
		"expected status %d, got status %d (success: %v) in %.4f seconds.",
		l.Identifier, l.URL, l.At, l.Method,
		l.StateExpected, l.StateResult, l.Success, l.TookSecs)
}

func main() {
	frickelbude := LogMessage{
		Identifier:    "frickelbude",
		URL:           "https://code.frickelbude.ch/api/v1/version",
		At:            time.Now(),
		Method:        "GET",
		StateExpected: 200,
		StateResult:   404,
		Success:       false,
		TookSecs:      0.132,
	}
	amazon := LogMessage{
		Identifier:    "amazon.de",
		URL:           "https://www.amazon.de/",
		At:            time.Now(),
		Method:        "GET",
		StateExpected: 200,
		StateResult:   200,
		Success:       true,
		TookSecs:      0.0012,
	}

	fmt.Println(frickelbude)
	fmt.Println(amazon)
}
