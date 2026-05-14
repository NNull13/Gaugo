# Gaugo Documentation

Gaugo is a Go-native evaluation library for AI applications. It is designed for
teams that want AI quality checks to live next to their Go tests, CI pipelines,
and internal evaluation tooling.

This documentation is organized by what you are trying to do.

## Start here

| Goal | Page |
| --- | --- |
| Install Gaugo and write the first test | [Getting started](getting-started.md) |
| Understand the core model | [Concepts](concepts.md) |
| Run evaluations through `go test` | [Testing with Suite](guides/testing-with-suite.md) |
| Run evaluations programmatically | [Programmatic Runner](guides/programmatic-runner.md) |
| Pick an LLM judge provider | [Provider](provider/index.md) |

## Guides

| Guide | Use it when |
| --- | --- |
| [Testing with Suite](guides/testing-with-suite.md) | You want evaluations to fail normal Go tests. |
| [Programmatic Runner](guides/programmatic-runner.md) | You need structured results for a CLI, service, dashboard, or pipeline. |
| [CI integration](guides/ci-integration.md) | You want stable GitHub Actions or CI jobs. |
| [Production usage](guides/production.md) | You need cost, rate-limit, timeout, and reliability guidance. |

## Metrics

Detailed documentation for every built-in metric, organized by category.
Start with the [metrics overview](metrics/index.md) for the full catalog.

| Category | Page |
| --- | --- |
| Overview and catalog | [metrics/index.md](metrics/index.md) |
| RAG evaluation | [metrics/rag.md](metrics/rag.md) |
| Answer quality | [metrics/answer.md](metrics/answer.md) |
| Safety | [metrics/safety.md](metrics/safety.md) |
| Generation quality | [metrics/generation-quality.md](metrics/generation-quality.md) |
| Structured output | [metrics/structured-output.md](metrics/structured-output.md) |
| Specialized | [metrics/specialized.md](metrics/specialized.md) |
| Deterministic contracts | [metrics/deterministic.md](metrics/deterministic.md) |

## Reference

| Reference | Covers |
| --- | --- |
| [Cases and inputs](reference/cases-and-inputs.md) | `Case`, `Question`, `ContextDocs`, `ExpectedContains`, `Input`, and `Output`. |
| [Metrics quick reference](reference/metrics.md) | Metric catalog, threshold defaults, and option summary. |
| [Configuration](reference/configuration.md) | Suite options, metric options, provider options, retries, and URL behavior. |
| [Results and reporting](reference/results-and-reporting.md) | `RunResult`, `CaseResult`, `MetricResult`, `Assert`, and `Reporter`. |
| [Errors and retries](reference/errors-and-retries.md) | Validation errors, run errors, judge errors, HTTP failures, and retry behavior. |

## Providers

| Provider | Page |
| --- | --- |
| OpenAI | [provider/openai.md](provider/openai.md) |
| Anthropic | [provider/anthropic.md](provider/anthropic.md) |
| Gemini | [provider/gemini.md](provider/gemini.md) |
| xAI | [provider/xai.md](provider/xai.md) |
| Local | [provider/local.md](provider/local.md) |

Start with the [provider overview](provider/index.md) if you are not sure
which adapter to use.

## Extending Gaugo

| Extension point | Page |
| --- | --- |
| Custom judge | [extending/custom-judges.md](extending/custom-judges.md) |
| Custom metric | [extending/custom-metrics.md](extending/custom-metrics.md) |
| Custom reporter | [extending/custom-reporters.md](extending/custom-reporters.md) |

## Examples

| Example | Page |
| --- | --- |
| No-provider guardrail checks | [examples/deterministic-checks.md](examples/deterministic-checks.md) |
| RAG quality evaluation | [examples/rag-evaluation.md](examples/rag-evaluation.md) |
| Dashboards and pipelines | [examples/dashboards-and-pipelines.md](examples/dashboards-and-pipelines.md) |

## Troubleshooting

If something fails, start with [Troubleshooting](troubleshooting.md). It covers
missing judges, invalid provider config, rate limits, empty outputs, and broken
test setup.
