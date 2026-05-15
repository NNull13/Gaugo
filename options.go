package gaugo

import (
	"fmt"
	"runtime"
	"time"

	"github.com/nnull13/gaugo/internal/failure"
)

const defaultMetricDetailsLimit = 8 * 1024

type config struct {
	judge       Judge
	parallelism int
	caseTimeout time.Duration
	detailsMax  int
	reporter    Reporter
}

// Option configures a Runner or Suite.
type Option func(*config) error

// WithJudge configures the LLM judge used by metrics.
func WithJudge(j Judge) Option {
	return func(c *config) error {
		c.judge = j
		return nil
	}
}

// WithParallelism configures the maximum number of concurrent case executions.
func WithParallelism(n int) Option {
	return func(c *config) error {
		if n <= 0 {
			return failure.Config(failure.CodeConfigInvalid, "WithParallelism", fmt.Sprintf("parallelism must be positive, got %d", n), nil)
		}
		c.parallelism = n
		return nil
	}
}

// WithCaseTimeout applies a per-case timeout to run and metric evaluation.
func WithCaseTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return failure.Config(failure.CodeConfigInvalid, "WithCaseTimeout", fmt.Sprintf("case timeout must be non-negative, got %s", d), nil)
		}
		c.caseTimeout = d
		return nil
	}
}

// WithReporter overrides the default testing reporter.
func WithReporter(r Reporter) Option {
	return func(c *config) error {
		if r == nil {
			return failure.Config(failure.CodeConfigInvalid, "WithReporter", "reporter cannot be nil", nil)
		}
		c.reporter = r
		return nil
	}
}

// WithMetricDetailsLimit caps stored metric detail bytes per metric result.
// Use 0 to disable details entirely.
func WithMetricDetailsLimit(bytes int) Option {
	return func(c *config) error {
		if bytes < 0 {
			return failure.Config(failure.CodeConfigInvalid, "WithMetricDetailsLimit", fmt.Sprintf("metric details limit must be non-negative, got %d", bytes), nil)
		}
		c.detailsMax = bytes
		return nil
	}
}

func defaultConfig() config {
	p := runtime.GOMAXPROCS(0)
	if p <= 0 {
		p = 1
	}
	return config{
		parallelism: p,
		detailsMax:  defaultMetricDetailsLimit,
	}
}

func applyOptions(opts []Option) (config, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		err := opt(&cfg)
		if err != nil {
			return config{}, err
		}
	}
	return cfg, nil
}
