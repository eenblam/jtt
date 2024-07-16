package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProcessCaptcha(t *testing.T) {
	/* Currently, POST validateCaptcha will always 200 for a valid jail!
	Instead, it uses the .captchaMatched boolean field to indicate a success.
	.captchaKey is unchanged in the response,
	but you have to click to get a new captcha (GET getnewcaptchaclient) and retry.

	So, to test, we need a mock captcha prompt and a mock solution,
	but the only thing we really have to control is the contents of the validation response.
	(We could later cover additional HTTP failure modes.)

	We test for:
	  - Errors if and only if expected
	  - Correct captcha key returned on success
	  - Solver solution is sent for validation
	*/

	// Mock captcha server
	captchaKey := "TEST_KEY"
	getCaptchaResponse := []byte(`{"captchaKey":"TEST_KEY","captchaImage":"TEST_IMAGE","userCode":null}`)
	// Solution server expects
	captchaSolution := "a1B2"
	// Solution returned by solver
	solverResponseTemplate := `{"choices":[{"message":{"content":"%s"}}]}`
	solverResponse := []byte(fmt.Sprintf(solverResponseTemplate, captchaSolution))
	// Simulate validation success
	var validateCaptchaSuccess bool

	mux := http.NewServeMux()
	mux.Handle("/jtclientweb/captcha/getnewcaptchaclient", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(getCaptchaResponse)
	}))
	mux.Handle("/jtclientweb/Captcha/validatecaptcha", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Process body and confirm expected solution was sent
		// If the solution is unexpected, we error here instead of continuing test, since that indicates misbehavior.
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		cp := &CaptchaProtocol{}
		err = json.Unmarshal(data, cp)
		if err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if cp.UserCode != captchaSolution {
			t.Fatalf("unexpected user code. Got %s, want %s", cp.UserCode, captchaSolution)
		}

		w.Header().Set("Content-Type", "application/json")
		// Always 200 for a valid jail
		w.WriteHeader(http.StatusOK)
		// Our actual response is a test control
		response := &CaptchaAttemptResults{
			CaptchaMatched: validateCaptchaSuccess,
			// JT resends the same key, but we always choose the latest just in case.
			CaptchaKey: captchaKey,
		}
		responseJson, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal response: %v", err)
		}
		w.Write(responseJson)
	}))
	mockJTServer := httptest.NewServer(mux)
	defer mockJTServer.Close()

	// Mock OpenAI
	mockOpenAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(solverResponse)
	}))
	defer mockOpenAIServer.Close()
	// Point solver at our mock
	OpenAICompletionsURL = mockOpenAIServer.URL

	jail := &Jail{
		BaseURL: mockJTServer.URL,
		Name:    "test",
	}

	cases := []struct {
		Label string
		// Solution the solver provides
		Solution string
		// What the captcha server expects
		ExpectedSolution string
		// Mock responses
		ValidateCaptchaSuccess bool
		WantErr                bool
	}{
		{
			Label:                  "test happy path",
			Solution:               "a1B2",
			ExpectedSolution:       "a1B2",
			ValidateCaptchaSuccess: true,
			WantErr:                false,
		},
		{
			Label:    "test captcha validation failure",
			Solution: "a1B2",
			// Keeping these the same; ValidateCaptchaSuccess is what actually determines server response
			ExpectedSolution:       "a1B2",
			ValidateCaptchaSuccess: false,
			WantErr:                true,
		},
		{
			Label:            "error on malformed solution",
			Solution:         "this breaks the regex",
			ExpectedSolution: "this breaks the regex",
			// Ensure that we get an error, even though captcha would be validated
			ValidateCaptchaSuccess: true,
			WantErr:                true,
		},
		// Can extend cases to test other failure modes:
		//   Test failure to get captcha key
		//   Test failure to solve captcha
		//   Test failure to submit solution
		//   Test failure to match captcha
		// Jail.updateCaptcha should be tested as well for retry behavior.
	}
	for _, c := range cases {
		t.Run(c.Label, func(t *testing.T) {
			// Set up the mock responses
			validateCaptchaSuccess = c.ValidateCaptchaSuccess
			solverResponse = []byte(fmt.Sprintf(solverResponseTemplate, c.Solution))
			// Reset the expected solution
			captchaSolution = c.ExpectedSolution

			got, err := ProcessCaptcha(jail)
			if c.WantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != captchaKey {
				t.Fatalf("unexpected captcha key. Got %s, want %s", got, captchaKey)
			}
		})
	}
}

func TestProcessCaptchaBadURL(t *testing.T) {
	// Tedious coverage farming. Just confirming we fail on a bad URL.
	j := &Jail{
		BaseURL: "Bad URL with spaces",
		Name:    "Doesn't matter",
	}
	_, err := ProcessCaptcha(j)
	if err == nil {
		t.Fatal("expected error, got nil for bad URL")
	}
}

func TestSolutionFormatIsValid(t *testing.T) {
	if !solutionFormatIsValid("a1B2") { // Happy path
		t.Fatal("expected solution format to be valid")
	}
	invalidCases := []string{
		"123",   // Too short
		"12345", // Too long
		" 123",  // White space
		"BVλd",  // Non-ASCII
	}
	for _, c := range invalidCases {
		if solutionFormatIsValid(c) {
			t.Fatalf(`expected solution "%s" to have invalid format`, c)
		}
	}
}
