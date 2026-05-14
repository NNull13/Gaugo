package provider

import (
	"fmt"
)

const (
	OpenAI    = "openai"
	Anthropic = "anthropic"
	Gemini    = "gemini"
	XAI       = "xai"
	Local     = "local"
)

const (
	OpenAIHost    = "api.openai.com"
	AnthropicHost = "api.anthropic.com"
	GeminiHost    = "generativelanguage.googleapis.com"
	XAIHost       = "api.x.ai"
)

const (
	OpenAIBaseURL    = "https://api.openai.com/v1"
	AnthropicBaseURL = "https://api.anthropic.com"
	GeminiBaseURL    = "https://generativelanguage.googleapis.com"
	XAIBaseURL       = "https://api.x.ai/v1"
	LocalBaseURL     = "http://127.0.0.1:11434"
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
	LocalDefaultModel     = "llama3.1"
)

const (
	LocalAPIKey = Local

	AnthropicDefaultAPIVersion = "2023-06-01"
	AnthropicDefaultMaxTokens  = 1024
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

const (
	configErrorFormat          = "invalid %s config: %s"
	configWrapErrorFormat      = "invalid %s config: %w"
	configFieldWrapErrorFormat = "invalid %s config: %s: %w"
	providerAPIKeyFormat       = "%s %s"
)

func ConfigError(provider, message string) error {
	return fmt.Errorf(configErrorFormat, provider, message)
}

func ConfigErrorf(provider, format string, args ...any) error {
	return ConfigError(provider, fmt.Sprintf(format, args...))
}

func ConfigWrapError(provider string, err error) error {
	return fmt.Errorf(configWrapErrorFormat, provider, err)
}

func ConfigFieldWrapError(provider, field string, err error) error {
	return fmt.Errorf(configFieldWrapErrorFormat, provider, field, err)
}

func ProviderAPIKeyRequiredError(provider string) error {
	return fmt.Errorf(providerAPIKeyFormat, provider, ConfigAPIKeyRequired)
}
