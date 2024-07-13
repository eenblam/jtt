package main

import (
	"errors"
	"fmt"
	"log"
)

const (
	MaxCaptchaAttempts = 5
)

// Response from GET GET_CAPTCHA_CLIENT_URL and request to POST VALIDATE_CAPTCHA_URL
// Format: {"captchaKey":"BASE64","captchaImage":"data:image/gif;base64,...BASE64...","userCode":null}
type CaptchaProtocol struct {
	// This is set in the GET response, and should also be sent in the POST
	CaptchaKey string `json:"captchaKey"`
	// This doesn't need to be set in the POST
	CaptchaImage string `json:"captchaImage"`
	// UserCode is null in the GET response, set for POST
	UserCode string `json:"userCode"`
}

// Success: {"captchaMatched": True, "captchaKey": "NEW\BASE64=="}
type CaptchaAttemptResults struct {
	CaptchaMatched bool   `json:"captchaMatched"`
	CaptchaKey     string `json:"captchaKey"`
}

// ProcessCaptcha retrieves and solves the captcha for the given jail, returning the captchaKey.
func ProcessCaptcha(jail *Jail) (string, error) {
	// Referer should be the jail's URL; used for redirection in web client.
	// May not affect us, but matches "normal" traffic.
	headers := map[string][]string{
		"Referer": {jail.getJailURL()},
	}

	// Yes, "captcha" and "Captcha", as seen in the application traffic
	getCaptchaClientURL := fmt.Sprintf("https://%s/jtclientweb/captcha/getnewcaptchaclient", jail.DomainName)
	validateCaptchaURL := fmt.Sprintf("https://%s/jtclientweb/Captcha/validatecaptcha", jail.DomainName)

	// Get the captcha key
	challenge := &CaptchaProtocol{}
	err := RequestJSONIntoStruct[interface{}, CaptchaProtocol]("GET", getCaptchaClientURL, headers, challenge, nil)
	if err != nil {
		return "", fmt.Errorf("failed to GET captcha key: %w", err)
	}

	// Solve captcha
	solution, err := solveCaptchaOpenAI(challenge.CaptchaImage)
	if err != nil {
		return "", fmt.Errorf("failed to get captcha solution: %w", err)
	}
	challenge.UserCode = solution
	log.Printf("Received solution: %s", solution)

	// Submit response
	results := &CaptchaAttemptResults{}
	err = RequestJSONIntoStruct[CaptchaProtocol, CaptchaAttemptResults]("POST", validateCaptchaURL, headers, results, challenge)
	if err != nil {
		return "", fmt.Errorf("failed captcha solution: %w", err)
	}
	if !results.CaptchaMatched {
		return "", errors.New("captcha did not match")
	}

	log.Printf("Solution \"%s\" matched key \"%s\"", solution, results.CaptchaKey)

	return results.CaptchaKey, nil
}
