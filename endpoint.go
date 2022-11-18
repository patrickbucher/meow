package meow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// Endpoint is something to monitor with according rules.
type Endpoint struct {
	// Identifier identifies the endpoint.
	Identifier string

	// URL is the URL to be requested.
	URL *url.URL

	// Method is the HTTP method to be used for the request.
	Method string

	// StatusOnline is the status indicating that the endpoint is online.
	StatusOnline uint16

	// Frequency is how often the endpoint is being tried.
	Frequency time.Duration

	// FailAfter is the number of failed requests after which the endpoint is
	// considered to be offline.
	FailAfter uint8
}

// EndpointPayload contains the same fields as Endpoint, but only as
// serializable primitives with JSON tags.
type EndpointPayload struct {
	Identifier   string `json:"identifier"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	StatusOnline uint16 `json:"status_online"`
	Frequency    string `json:"frequency"`
	FailAfter    uint8  `json:"fail_after"`
}

const idPatternRaw = "^[a-z][-a-z0-9]+$"

var idPattern = regexp.MustCompile(idPatternRaw)

// NewDefaultEndpoint creates a new Endpoint from rawURL, which is parsed. An
// endpoint is returned, if the rawURL is valid, and an error (indicating the
// parse error) otherwise.
func NewDefaultEndpoint(id string, rawURL string) (*Endpoint, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &Endpoint{
		Identifier:   id,
		URL:          parsedURL,
		Method:       http.MethodGet,
		StatusOnline: http.StatusOK,
		Frequency:    5 * time.Minute,
		FailAfter:    3,
	}, nil
}

// String returns the Endpoint's fields separated by a space.
func (e Endpoint) String() string {
	return fmt.Sprintf("%s %s %s %d %v %d", e.Identifier,
		e.URL, e.Method, e.StatusOnline, e.Frequency, e.FailAfter)
}

// JSON returns the Endpoint's fields as a JSON data, or an error, if it cannot
// be serialized.
func (e Endpoint) JSON() ([]byte, error) {
	payload := EndpointPayload{
		e.Identifier,
		e.URL.String(),
		e.Method,
		e.StatusOnline,
		e.Frequency.String(),
		e.FailAfter,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal endpoint %v as JSON: %v", e, err)
	}
	return data, nil
}

var methodsAllowed = map[string]bool{
	http.MethodGet:  true,
	http.MethodHead: true,
}

// EndpointFromJSON creates a new endpoint from a given JSON structure.
func EndpointFromJSON(rawJSON string) (*Endpoint, error) {
	var payload EndpointPayload
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, fmt.Errorf(`unmarshal raw json "%s": %v`, rawJSON, err)
	}
	return EndpointFromPayload(payload)
}

// EndpointFromPayload creates an endpoint from the given payload.
func EndpointFromPayload(payload EndpointPayload) (*Endpoint, error) {
	if !idPattern.MatchString(payload.Identifier) {
		return nil, fmt.Errorf(`identifier "%s" does not match pattern "%s"`,
			payload.Identifier, idPatternRaw)
	}
	parsedURL, err := url.Parse(payload.URL)
	if err != nil {
		return nil, fmt.Errorf(`parse URL "%s": %v`, payload.URL, err)
	}
	if allowed, ok := methodsAllowed[payload.Method]; !allowed || !ok {
		return nil, fmt.Errorf(`"%s" is not an allowed method`, payload.Method)
	}
	if payload.StatusOnline < 100 || payload.StatusOnline > 999 {
		return nil, fmt.Errorf(`"%d" is not a valid status code`, payload.StatusOnline)
	}
	frequency, err := time.ParseDuration(payload.Frequency)
	if err != nil {
		return nil, fmt.Errorf(`"%s" is not a valid duration`, payload.Frequency)
	}
	return &Endpoint{
		payload.Identifier,
		parsedURL,
		payload.Method,
		payload.StatusOnline,
		frequency,
		payload.FailAfter,
	}, nil
}

// EndpointFromRecord creates a new Endpoint from the given record, which must
// provide the fields in the following order: 1) Identifier, 2) URL, 3) Method,
// 4) StatusOnline, 5) Frequency, 6) FailAfter
func EndpointFromRecord(record []string) (*Endpoint, error) {
	const nFields = 6
	if len(record) < nFields {
		return nil, fmt.Errorf(`malformed record "%s" (needs %d fields)`, record, nFields)
	}
	id := record[0]
	if !idPattern.MatchString(id) {
		return nil, fmt.Errorf(`id "%s" does not match pattern %s`, id, idPatternRaw)
	}
	parsedURL, err := url.Parse(record[1])
	if err != nil {
		return nil, fmt.Errorf(`parse URL "%s": %v`, record[1], err)
	}
	method := record[2]
	if allowed, ok := methodsAllowed[method]; !allowed || !ok {
		return nil, fmt.Errorf(`"%s" is not an allowed method`, method)
	}
	statusOnline, err := strconv.Atoi(record[3])
	if err != nil || statusOnline < 100 || statusOnline > 999 {
		return nil, fmt.Errorf(`"%s" is not a valid status code`, record[3])
	}
	frequency, err := time.ParseDuration(record[4])
	if err != nil {
		return nil, fmt.Errorf(`"%s" is not a valid duration`, record[4])
	}
	failAfter, err := strconv.Atoi(record[5])
	if err != nil {
		return nil, fmt.Errorf(`"%s" is not a number`, record[5])
	}
	return &Endpoint{
		Identifier:   id,
		URL:          parsedURL,
		Method:       method,
		StatusOnline: uint16(statusOnline),
		Frequency:    frequency,
		FailAfter:    uint8(failAfter),
	}, nil
}
