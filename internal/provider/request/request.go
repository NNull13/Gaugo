package request

import (
	"strings"

	"github.com/nnull13/gaugo"
	"github.com/nnull13/gaugo/internal/prompt"
	"github.com/nnull13/gaugo/internal/provider/wire"
)

// ToEval converts the public judge request into the internal wire request.
func ToEval(req gaugo.JudgeRequest) wire.EvalRequest {
	docs := make([]prompt.Document, len(req.ContextDocs))
	for i := range req.ContextDocs {
		docs[i] = prompt.Document{
			ID:   req.ContextDocs[i].ID,
			Text: req.ContextDocs[i].Text,
		}
	}
	return wire.EvalRequest{
		Metric:       strings.TrimSpace(req.Metric),
		Instructions: strings.TrimSpace(req.Instructions),
		UserPrompt: prompt.BuildUserPromptWithExpectedAndInstructions(
			req.Question,
			req.Answer,
			req.ExpectedAnswer,
			req.ExpectedInstructions,
			docs,
		),
		Schema: req.Schema,
	}
}

// ToRetry converts the public retry config into the internal wire shape.
func ToRetry(cfg gaugo.RetryConfig) wire.RetryConfig {
	def := gaugo.DefaultRetryConfig()
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = def.MaxAttempts
	}
	if cfg.BaseDelay == 0 {
		cfg.BaseDelay = def.BaseDelay
	}
	if cfg.MaxDelay == 0 {
		cfg.MaxDelay = def.MaxDelay
	}
	return wire.RetryConfig{
		MaxAttempts: cfg.MaxAttempts,
		BaseDelay:   cfg.BaseDelay,
		MaxDelay:    cfg.MaxDelay,
	}
}
