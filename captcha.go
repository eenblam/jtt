package main

import (
	"errors"
	"fmt"
	"log"
)

const (
	MAX_CAPTCHA_ATTEMPTS = 5
	// Yes, "Requred". This key (like others) is typo'd. Using a variable here so it's easy to update if they fix the typo.
	CAPTCHA_REQUIRED_KEY = "captchaRequred"
	OMS_URL              = "https://omsweb.public-safety-cloud.com"
)

var (
	// Yes, "captcha" and "Captcha", as seen in the application traffic
	GET_CAPTCHA_CLIENT_URL = fmt.Sprintf("%s/jtclientweb/captcha/getnewcaptchaclient", OMS_URL)
	VALIDATE_CAPTCHA_URL   = fmt.Sprintf("%s/jtclientweb/Captcha/validatecaptcha", OMS_URL)
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

// ProcessCaptcha retrieves and solves the captcha for the given jail name, returning the captchaKey.
func ProcessCaptcha(name string) (string, error) {
	// Referer should be the jail's URL; used for redirection in web client.
	// May not affect us, but matches "normal" traffic.
	headers := map[string][]string{
		"Referer": {getJailURL(name)},
	}

	// Get the captcha key
	challenge := &CaptchaProtocol{}
	err := RequestJSONIntoStruct[interface{}, CaptchaProtocol]("GET", GET_CAPTCHA_CLIENT_URL, headers, challenge, nil)
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
	err = RequestJSONIntoStruct[CaptchaProtocol, CaptchaAttemptResults]("POST", VALIDATE_CAPTCHA_URL, headers, results, challenge)
	if err != nil {
		return "", fmt.Errorf("failed captcha solution: %w", err)
	}
	if !results.CaptchaMatched {
		return "", errors.New("captcha did not match")
	}

	log.Printf("Solution \"%s\" matched key \"%s\"", solution, results.CaptchaKey)

	return results.CaptchaKey, nil
}
