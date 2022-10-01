package meow

import (
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

// String returns the Endpoint's  fields separated by a space.
func (e Endpoint) String() string {
	return fmt.Sprintf("%s %s %s %d %v %d", e.Identifier,
		e.URL, e.Method, e.StatusOnline, e.Frequency, e.FailAfter)
}

var methodsAllowed = map[string]bool{
	http.MethodGet:  true,
	http.MethodHead: true,
}

// EndpointFrom creates a new Endpoint from the given record, which must provide
// the fields in the following order: 1) Identifier, 2) URL, 3) Method, 4)
// StatusOnline, 5) Frequency, 6) FailAfter
func EndpointFrom(record []string) (*Endpoint, error) {
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
