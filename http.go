package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func RequestJSONIntoStruct[P interface{}, T interface{}](method string, url string, headers map[string][]string, into *T, payload *P) error {
	var req *http.Request
	var err error

	if payload != nil { // If payload provided, marshal to JSON. Should be missing for GET.
		// Don't shadow outer err in next assignment
		payloadJson, marshalErr := json.Marshal(payload)
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
	err = json.Unmarshal(body, into)
	if err != nil {
		return fmt.Errorf("failed to unmarshal POST jail data body: %w", err)
	}

	return nil
}
