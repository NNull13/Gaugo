package gaugo

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type classifiedTestError struct{}

func (classifiedTestError) Error() string {
	return "provider request failed status=429 request_id=\"req_1\" body=redacted(len=12)"
}
func (classifiedTestError) GaugoErrorKind() string { return string(ErrorKindProviderRateLimit) }
func (classifiedTestError) GaugoProvider() string  { return "anthropic" }
func (classifiedTestError) GaugoStatusCode() int   { return 429 }
func (classifiedTestError) GaugoRequestID() string { return "req_1" }
func (classifiedTestError) GaugoBodyBytes() int    { return 12 }

func TestClassifyErrorFromDuckTypedProviderError(t *testing.T) {
	t.Parallel()

	info := ClassifyError(classifiedTestError{})
	if info.Kind != ErrorKindProviderRateLimit {
		t.Fatalf("kind got=%q want=%q", info.Kind, ErrorKindProviderRateLimit)
	}
	if info.Provider != "anthropic" || info.StatusCode != 429 || info.RequestID != "req_1" || info.BodyBytes != 12 {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestClassifyErrorContext(t *testing.T) {
	t.Parallel()

	if info := ClassifyError(context.Canceled); info.Kind != ErrorKindContextCanceled || string(info.Code) != "context_canceled" {
		t.Fatalf("canceled info got=%+v", info)
	}
	if info := ClassifyError(context.DeadlineExceeded); info.Kind != ErrorKindContextDeadline || string(info.Code) != "context_deadline" {
		t.Fatalf("deadline info got=%+v", info)
	}
}

func TestMetricErrorInfo(t *testing.T) {
	t.Parallel()

	details := errorInfoDetails(classifiedTestError{})
	info, ok := MetricErrorInfo(MetricResult{Details: details})
	if !ok {
		t.Fatalf("expected metric error info")
	}
	if info.Kind != ErrorKindProviderRateLimit || info.Provider != "anthropic" {
		t.Fatalf("unexpected info: %+v", info)
	}

	if _, ok := MetricErrorInfo(MetricResult{Details: []byte(`{"score":1}`)}); ok {
		t.Fatalf("quality metric details should not parse as error info")
	}
}

func TestClassifyErrorDoesNotInferFromStrings(t *testing.T) {
	t.Parallel()

	info := ClassifyError(errors.New("answer relevancy parse failed: score must be in [0,1]"))
	if info.Kind != ErrorKindUnknown {
		t.Fatalf("kind got=%q want=%q", info.Kind, ErrorKindUnknown)
	}
}

func TestClassifyErrorFromStructuredMetricErrors(t *testing.T) {
	t.Parallel()

	// metric configuration error: calling a judge-backed metric without a Judge.
	_, configErr := Faithfulness().Evaluate(context.Background(), EvalInput{}, nil)

	// metric parse error: judge returns malformed JSON the metric cannot decode.
	parseJudge := fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{RawJSON: []byte(`{"score":2,"reason":"out of range"}`)}, nil
		},
	}
	_, parseErr := AnswerRelevancy().Evaluate(context.Background(), EvalInput{
		Input:  Input{Question: "Q"},
		Output: Output{Answer: "A"},
	}, parseJudge)

	// metric option error: an out-of-range threshold surfaces at Evaluate time.
	_, optionErr := Faithfulness(WithThreshold(2.0)).Evaluate(context.Background(), EvalInput{}, nil)

	tests := []struct {
		name string
		err  error
		want ErrorKind
	}{
		{name: "metric configuration", err: configErr, want: ErrorKindMetric},
		{name: "metric parse", err: parseErr, want: ErrorKindMetricParse},
		{name: "metric option", err: optionErr, want: ErrorKindMetric},
	}
	for _, tt := range tests {
		if tt.err == nil {
			t.Fatalf("%s setup did not produce an error", tt.name)
		}
		if !errors.Is(tt.err, sentinelForKind(tt.want)) {
			t.Fatalf("%s errors.Is did not match %q for %v", tt.name, tt.want, tt.err)
		}
		info := ClassifyError(tt.err)
		if info.Kind != tt.want {
			t.Fatalf("%s kind got=%q want=%q", tt.name, info.Kind, tt.want)
		}
	}
}

func TestTypedErrorSupportsErrorsIsAndAs(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("outer: %w", &Error{
		Kind:       ErrorKindProviderRateLimit,
		Code:       ErrorCode("provider_http_status"),
		Provider:   "openai",
		StatusCode: 429,
		Message:    "openai request failed",
	})
	if !errors.Is(err, ErrProviderRateLimit) {
		t.Fatalf("expected errors.Is provider rate limit")
	}
	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("expected errors.As *gaugo.Error")
	}
	info := typed.Info()
	if info.Kind != ErrorKindProviderRateLimit || string(info.Code) != "provider_http_status" || info.Provider != "openai" {
		t.Fatalf("unexpected typed info: %+v", info)
	}
}

func TestErrorSentinelsClassifyByKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want ErrorKind
	}{
		{ErrConfig, ErrorKindConfig},
		{ErrValidation, ErrorKindValidation},
		{ErrMetric, ErrorKindMetric},
		{ErrMetricParse, ErrorKindMetricParse},
		{ErrProviderRequest, ErrorKindProviderRequest},
		{ErrProviderAuth, ErrorKindProviderAuth},
		{ErrProviderRateLimit, ErrorKindProviderRateLimit},
		{ErrProviderUnavailable, ErrorKindProviderUnavailable},
		{ErrProviderResponse, ErrorKindProviderResponse},
		{ErrProviderRefusal, ErrorKindProviderRefusal},
		{ErrProviderTruncated, ErrorKindProviderTruncated},
		{ErrPanic, ErrorKindPanic},
	}
	for _, tt := range tests {
		info := ClassifyError(tt.err)
		if info.Kind != tt.want {
			t.Fatalf("ClassifyError(%v).Kind got=%q want=%q", tt.err, info.Kind, tt.want)
		}
	}
}

func sentinelForKind(kind ErrorKind) error {
	switch kind {
	case ErrorKindMetric:
		return ErrMetric
	case ErrorKindMetricParse:
		return ErrMetricParse
	case ErrorKindProviderRateLimit:
		return ErrProviderRateLimit
	default:
		return ErrValidation
	}
}

// statusCodeOnlyError implements gaugoStatusCodeError but not gaugoKindError,
// exercising the kindFromStatusCode path in ClassifyError.
type statusCodeOnlyError struct {
	code int
}

func (e statusCodeOnlyError) Error() string        { return "http error" }
func (e statusCodeOnlyError) GaugoStatusCode() int { return e.code }

func TestClassifyErrorFromStatusCodeWithoutKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code int
		want ErrorKind
	}{
		{429, ErrorKindProviderRateLimit},
		{401, ErrorKindProviderAuth},
		{403, ErrorKindProviderAuth},
		{400, ErrorKindProviderRequest},
		{422, ErrorKindProviderRequest},
		{500, ErrorKindProviderUnavailable},
		{503, ErrorKindProviderUnavailable},
		{200, ErrorKindUnknown},
	}
	for _, tt := range tests {
		info := ClassifyError(statusCodeOnlyError{code: tt.code})
		if info.Kind != tt.want {
			t.Fatalf("ClassifyError(status=%d).Kind got=%q want=%q", tt.code, info.Kind, tt.want)
		}
		if info.StatusCode != tt.code {
			t.Fatalf("ClassifyError(status=%d).StatusCode got=%d", tt.code, info.StatusCode)
		}
	}
}

func TestClassifyErrorWrappedStatusCodePreservesKind(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("outer: %w", statusCodeOnlyError{code: 429})
	info := ClassifyError(wrapped)
	if info.Kind != ErrorKindProviderRateLimit {
		t.Fatalf("wrapped status code kind got=%q want=%q", info.Kind, ErrorKindProviderRateLimit)
	}
}

func TestClassifyErrorJudgeEvaluationPreservesProviderKind(t *testing.T) {
	t.Parallel()

	// A Judge that fails with a 429-tagged transport error should propagate
	// through Faithfulness.Evaluate as a provider-rate-limit classification,
	// not a generic metric error.
	j := fakeJudge{
		eval: func(context.Context, JudgeRequest) (JudgeResponse, error) {
			return JudgeResponse{}, statusCodeOnlyError{code: 429}
		},
	}
	_, err := Faithfulness().Evaluate(context.Background(), EvalInput{
		Input:  Input{Question: "Q"},
		Output: Output{Answer: "A"},
	}, j)
	if err == nil {
		t.Fatal("expected judge error to propagate")
	}
	info := ClassifyError(err)
	if info.Kind != ErrorKindProviderRateLimit {
		t.Fatalf("judge provider kind got=%q want=%q", info.Kind, ErrorKindProviderRateLimit)
	}
}
