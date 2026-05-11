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
// Three metrics cover the RAG evaluation triad:
//   - [Faithfulness] — is the answer supported by the provided context?
//   - [AnswerRelevancy] — does the answer address the question?
//   - [ContextRelevancy] — are the retrieved documents useful for the question?
//
// All metrics accept [WithThreshold] to set a custom pass/fail score in [0,1].
//
// # Provider judges
//
// Metrics that require LLM evaluation use a [Judge] interface. Built-in adapters
// are provided for OpenAI, Anthropic, Gemini, xAI, and local models (Ollama).
// See the provider sub-packages for configuration details.
package gaugo
