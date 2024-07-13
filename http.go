package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GetJSON makes a GET request to url, then unmarshals the response body from JSON.
// Additional headers can be passed as a map.
func GetJSON[Res interface{}](url string, headers map[string][]string, responseBody *Res) error {
	return RequestJSON[interface{}, Res]("GET", url, headers, nil, responseBody)
}

// PostJSON makes a POST request to url, marshaling the request body to JSON and unmarshaling the response body from JSON.
// Method is set by argument, and additional headers can be passed as a map.
func PostJSON[Req interface{}, Res interface{}](url string, headers map[string][]string, requestBody *Req, responseBody *Res) error {
	return RequestJSON[Req, Res]("POST", url, headers, requestBody, responseBody)
}

// RequestJSON makes an HTTP request, marshaling the request body to JSON and unmarshaling the response body from JSON.
// Method is set by argument, and additional headers can be passed as a map.
// For GET requests, Req should be interface{} and requestBody should be nil.
func RequestJSON[Req interface{}, Res interface{}](method string, url string, headers map[string][]string, requestBody *Req, responseBody *Res) error {
	var req *http.Request
	var err error

	if requestBody != nil { // If payload provided, marshal to JSON. Should be missing for GET.
		// Don't shadow outer err in next assignment
		payloadJson, marshalErr := json.Marshal(requestBody)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal payload to JSON: %w", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(payloadJson))
	} else { // Probably a GET
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	// Set any extra headers
	for k, v := range headers {
		req.Header[k] = v
	}

	// Make request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("got non-200 status for jail data: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST jail data body: %w", err)
	}
	err = json.Unmarshal(body, responseBody)
	if err != nil {
		return fmt.Errorf("failed to unmarshal POST jail data body: %w", err)
	}

	return nil
}
