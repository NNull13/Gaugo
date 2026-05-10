package provider

import (
	"errors"
	"fmt"
)

const (
	OpenAI    = "openai"
	Anthropic = "anthropic"
	Gemini    = "gemini"
	XAI       = "xai"
	Ollama    = "ollama"
)

const (
	OpenAIHost    = "api.openai.com"
	AnthropicHost = "api.anthropic.com"
	GeminiHost    = "generativelanguage.googleapis.com"
	XAIHost       = "api.x.ai"
)

const (
	OpenAIBaseURL      = "https://api.openai.com/v1"
	AnthropicBaseURL   = "https://api.anthropic.com"
	GeminiBaseURL      = "https://generativelanguage.googleapis.com"
	XAIBaseURL         = "https://api.x.ai/v1"
	OllamaLocalBaseURL = "http://127.0.0.1:11434"
)

const (
	OpenAIChatCompletionsPath = "/chat/completions"
	ResponsesPath             = "/responses"
	AnthropicMessagesPath     = "/v1/messages"
	OllamaNativeChatPath      = "/api/chat"
	OpenAIV1Path              = "/v1"
)

const (
	OpenAIDefaultModel    = "gpt-4.1-mini"
	AnthropicDefaultModel = "claude-sonnet-4-5"
	GeminiDefaultModel    = "gemini-2.5-flash"
	XAIDefaultModel       = "grok-4.3"
	OllamaDefaultModel    = "llama3.1"
)

const (
	OllamaLocalAPIKey = Ollama
)

const (
	ResponsesWireName = "responses"
)

const (
	FieldEndpointURL = "endpoint URL"

	ConfigAPIKeyRequired              = "api key is required"
	ConfigMaxResponseBodyNonNegative  = "max response body must be non-negative"
	ConfigOpenAIEndpointRequiresMode  = "openai endpoint requires mode=openai"
	ConfigUnsupportedOpenAIEndpoint   = "unsupported openai endpoint %q"
	ConfigUnsupportedProviderWireMode = "unsupported mode %q"
)

func ConfigError(provider, message string) error {
	return fmt.Errorf("invalid %s config: %s", provider, message)
}

func ConfigErrorf(provider, format string, args ...any) error {
	return ConfigError(provider, fmt.Sprintf(format, args...))
}

func ConfigWrapError(provider string, err error) error {
	return fmt.Errorf("invalid %s config: %w", provider, err)
}

func ConfigFieldWrapError(provider, field string, err error) error {
	return fmt.Errorf("invalid %s config: %s: %w", provider, field, err)
}

func APIKeyRequiredError() error {
	return errors.New(ConfigAPIKeyRequired)
}

func ProviderAPIKeyRequiredError(provider string) error {
	return fmt.Errorf("%s %s", provider, ConfigAPIKeyRequired)
}
