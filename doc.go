// Package gaugo provides an idiomatic Go testing harness for AI application evaluation.
//
// Gaugo evaluates RAG pipelines and AI systems directly within Go's testing workflow,
// producing deterministic, concurrent, CI-friendly results without external orchestration.
//
// # Quick start
//
// Use [Suite] inside a standard Go test to register cases and assert metrics:
//
//	func TestRAG(t *testing.T) {
//	    suite := gaugo.New(t, gaugo.WithJudge(judge))
//	    suite.Case("basic",
//	        gaugo.Question("What is Go?"),
//	        gaugo.ContextDocs(gaugo.Doc("d1", "Go is a programming language.")),
//	    )
//	    suite.Assert(ctx, myRAG, gaugo.Faithfulness(), gaugo.AnswerRelevancy())
//	}
//
// # Programmatic usage
//
// Use [Runner] when you need structured results outside the testing framework:
//
//	runner, _ := gaugo.NewRunner(gaugo.WithJudge(judge))
//	runner.Case("example", gaugo.Question("Q?"), gaugo.ExpectedContains("answer"))
//	result, _ := runner.Run(ctx, myFunc)
//	fmt.Println(result.Summary())
//
// # Built-in metrics
//
// Built-in metrics cover RAG, safety, generation quality, structured output,
// instruction following, domain-specific checks, and deterministic contracts.
//
// RAG and answer quality:
//   - [Faithfulness], [AnswerRelevancy], [ContextRelevancy]
//   - [ContextPrecision], [ContextRecall], [AnswerCorrectness]
//
// Safety and generation quality:
//   - [Hallucination], [Toxicity], [Bias]
//   - [Coherence], [Conciseness], [Completeness]
//
// Structured output and deterministic checks:
//   - [JSONValidity], [SchemaCompliance], [ExpectedJSON]
//   - [AnswerSimilarity], [Latency], [AnswerLength], [ExpectedRegex]
//
// Instruction and domain-specific metrics:
//   - [InstructionAdherence], [GEval]
//   - [CitationAccuracy], [SummarizationQuality]
//
// All metrics accept [WithThreshold] to set a custom pass/fail score in [0,1].
// Metric interfaces, shared input/output types, and built-in constructors also
// live in the [github.com/nnull13/gaugo/metric] sub-package. The root package
// re-exports that public surface so callers can use either [Faithfulness] or
// metric.Faithfulness interchangeably.
//
// # Provider judges
//
// Metrics that require LLM evaluation use a [Judge] interface. Built-in adapters
// are provided for OpenAI, Anthropic, Gemini, xAI, NVIDIA NIM, and custom
// endpoints (Ollama, LM Studio, and other OpenAI/Anthropic-compatible servers).
// See the provider sub-packages for configuration details.
package gaugo
