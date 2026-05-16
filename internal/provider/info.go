package provider

import (
	"fmt"

	"github.com/nnull13/gaugo/internal/failure"
)

const (
	OpenAI    = "openai"
	Anthropic = "anthropic"
	Gemini    = "gemini"
	XAI       = "xai"
	NVIDIA    = "nvidia"
	Custom    = "custom"
)

const (
	OpenAIHost    = "api.openai.com"
	AnthropicHost = "api.anthropic.com"
	GeminiHost    = "generativelanguage.googleapis.com"
	XAIHost       = "api.x.ai"
	NVIDIAHost    = "integrate.api.nvidia.com"
)

const (
	OpenAIBaseURL    = "https://api.openai.com/v1"
	AnthropicBaseURL = "https://api.anthropic.com"
	GeminiBaseURL    = "https://generativelanguage.googleapis.com"
	XAIBaseURL       = "https://api.x.ai/v1"
	NVIDIABaseURL    = "https://integrate.api.nvidia.com/v1"
	CustomBaseURL    = "http://127.0.0.1:11434"
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
	NVIDIADefaultModel    = "meta/llama-3.3-70b-instruct"
	CustomDefaultModel    = "llama3.1"
)

const (
	CustomAPIKey = Custom

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
	configErrorFormat    = "invalid %s config: %s"
	providerAPIKeyFormat = "%s %s"
)

func ConfigError(provider, message string) error {
	return &failure.Error{
		Kind:     failure.KindConfig,
		Code:     configCode(message),
		Provider: provider,
		Message:  fmt.Sprintf(configErrorFormat, provider, message),
	}
}

func ConfigErrorf(provider, format string, args ...any) error {
	return ConfigError(provider, fmt.Sprintf(format, args...))
}

func ConfigWrapError(provider string, err error) error {
	return &failure.Error{
		Kind:     failure.KindConfig,
		Provider: provider,
		Message:  fmt.Sprintf("invalid %s config", provider),
		Err:      err,
	}
}

func ConfigFieldWrapError(provider, field string, err error) error {
	return &failure.Error{
		Kind:     failure.KindConfig,
		Provider: provider,
		Field:    field,
		Message:  fmt.Sprintf("invalid %s config: %s", provider, field),
		Err:      err,
	}
}

func ProviderAPIKeyRequiredError(provider string) error {
	return &failure.Error{
		Kind:     failure.KindConfig,
		Code:     failure.CodeProviderAPIKeyRequired,
		Provider: provider,
		Message:  fmt.Sprintf(providerAPIKeyFormat, provider, ConfigAPIKeyRequired),
	}
}

func configCode(message string) failure.Code {
	switch message {
	case ConfigAPIKeyRequired:
		return failure.CodeProviderAPIKeyRequired
	default:
		return failure.CodeProviderConfigInvalid
	}
}
