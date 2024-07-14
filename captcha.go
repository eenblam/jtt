package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
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
	// Tempting to refactor this elsewhere to separate concerns.
	getCaptchaClientURL, err := url.JoinPath(jail.BaseURL, "jtclientweb/captcha/getnewcaptchaclient")
	if err != nil {
		return "", fmt.Errorf("failed to join URL: %w", err)
	}
	validateCaptchaURL, err := url.JoinPath(jail.BaseURL, "jtclientweb/Captcha/validatecaptcha")
	if err != nil {
		return "", fmt.Errorf("failed to join URL: %w", err)
	}

	// Get the captcha key
	challenge := &CaptchaProtocol{}
	err = GetJSON[CaptchaProtocol](getCaptchaClientURL, headers, challenge)
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
	err = PostJSON[CaptchaProtocol, CaptchaAttemptResults](validateCaptchaURL, headers, challenge, results)
	if err != nil {
		return "", fmt.Errorf("failed to submit captcha solution: %w", err)
	}
	if !results.CaptchaMatched {
		return "", errors.New("captcha did not match")
	}

	log.Printf("Solution \"%s\" matched key \"%s\"", solution, results.CaptchaKey)

	return results.CaptchaKey, nil
}
