package main

import (
	"fmt"
)

const OpenAICompletionsURL = "https://api.openai.com/v1/chat/completions"

// RequestPayload represents the entire request payload
type RequestPayload struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

// Message represents an individual message in the payload
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

// Content represents the content of a message
type Content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`      // Text field for system role content
	ImageURL *ImageURL `json:"image_url,omitempty"` // ImageURL field for user role content
}

// ImageURL represents the URL of the image
type ImageURL struct {
	URL string `json:"url"`
}

/* Example response content we're parsing
{
  "id": "chatcmpl-aBCd...",
  "object": "chat.completion",
  "created": 1719504619,
  "model": "gpt-4o-2024-05-13",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "ETaX"
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 296,
    "completion_tokens": 3,
    "total_tokens": 299
  },
  "system_fingerprint": "fp_4008e3b719"
}
*/

type CompletionResponse struct {
	Choices []ResponseChoice `json:"choices"`
}

// ResponseChoice represents a Choice in the response, with a simplified Content
type ResponseChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// Get the payload to request a captcha solve from GPT-4o
func getCaptchaPayload(inlineImage string) RequestPayload {
	return RequestPayload{
		Model: "gpt-4o",
		Messages: []Message{
			{
				Role: "system",
				Content: []Content{
					{
						Type: "text",
						Text: "The user will send you images containing a single word of obfuscated text. Reply only with the text in the image, with no spaces or quotes.",
					},
				},
			},
			{
				Role: "user",
				Content: []Content{
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: inlineImage,
						},
					},
				},
			},
		},
		MaxTokens: 300,
	}
}

// Solve a captcha image, returning the pictured text or an error.
func solveCaptchaOpenAI(inline_image string) (string, error) {
	payload := getCaptchaPayload(inline_image)
	completion := &CompletionResponse{}
	headers := map[string][]string{
		"Authorization": {"Bearer " + OpenAIAPIKey},
	}
	err := RequestJSONIntoStruct[RequestPayload, CompletionResponse]("POST", OpenAICompletionsURL, headers, completion, &payload)
	if err != nil {
		return "", fmt.Errorf("error from OpenAI: %w", err)

	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI completion %v", completion)
	}
	return completion.Choices[0].Message.Content, nil
}
