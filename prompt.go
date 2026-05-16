// Gaugo's built-in metrics. Import this when implementing a custom [gaugo.Judge]
// so you can reuse the same prompts without managing API keys through Gaugo.
package gaugo

import (
	"encoding/json"

	internalprompt "github.com/nnull13/gaugo/internal/prompt"
)

// FaithfulnessInstructions returns the system instruction for the Faithfulness metric.
func FaithfulnessInstructions() string { return internalprompt.FaithfulnessInstructions() }

// FaithfulnessSchema returns the JSON schema for Faithfulness metric output.
func FaithfulnessSchema() json.RawMessage { return internalprompt.FaithfulnessSchema() }

// AnswerRelevancyInstructions returns the system instruction for the AnswerRelevancy metric.
func AnswerRelevancyInstructions() string { return internalprompt.AnswerRelevancyInstructions() }

// AnswerRelevancySchema returns the JSON schema for AnswerRelevancy metric output.
func AnswerRelevancySchema() json.RawMessage { return internalprompt.AnswerRelevancySchema() }

// ContextRelevancyInstructions returns the system instruction for the ContextRelevancy metric.
func ContextRelevancyInstructions() string { return internalprompt.ContextRelevancyInstructions() }

// ContextRelevancySchema returns the JSON schema for ContextRelevancy metric output.
func ContextRelevancySchema() json.RawMessage { return internalprompt.ContextRelevancySchema() }

// ContextPrecisionInstructions returns the system instruction for the ContextPrecision metric.
func ContextPrecisionInstructions() string { return internalprompt.ContextPrecisionInstructions() }

// ContextPrecisionSchema returns the JSON schema for ContextPrecision metric output.
func ContextPrecisionSchema() json.RawMessage { return internalprompt.ContextPrecisionSchema() }

// ContextRecallInstructions returns the system instruction for the ContextRecall metric.
func ContextRecallInstructions() string { return internalprompt.ContextRecallInstructions() }

// ContextRecallSchema returns the JSON schema for ContextRecall metric output.
func ContextRecallSchema() json.RawMessage { return internalprompt.ContextRecallSchema() }

// AnswerCorrectnessInstructions returns the system instruction for the AnswerCorrectness metric.
func AnswerCorrectnessInstructions() string { return internalprompt.AnswerCorrectnessInstructions() }

// AnswerCorrectnessSchema returns the JSON schema for AnswerCorrectness metric output.
func AnswerCorrectnessSchema() json.RawMessage { return internalprompt.AnswerCorrectnessSchema() }

// HallucinationInstructions returns the system instruction for the Hallucination metric.
func HallucinationInstructions() string { return internalprompt.HallucinationInstructions() }

// HallucinationSchema returns the JSON schema for Hallucination metric output.
func HallucinationSchema() json.RawMessage { return internalprompt.HallucinationSchema() }

// ToxicityInstructions returns the system instruction for the Toxicity metric.
func ToxicityInstructions() string { return internalprompt.ToxicityInstructions() }

// ToxicitySchema returns the JSON schema for Toxicity metric output.
func ToxicitySchema() json.RawMessage { return internalprompt.ToxicitySchema() }

// BiasInstructions returns the system instruction for the Bias metric.
func BiasInstructions() string { return internalprompt.BiasInstructions() }

// BiasSchema returns the JSON schema for Bias metric output.
func BiasSchema() json.RawMessage { return internalprompt.BiasSchema() }

// CoherenceInstructions returns the system instruction for the Coherence metric.
func CoherenceInstructions() string { return internalprompt.CoherenceInstructions() }

// CoherenceSchema returns the JSON schema for Coherence metric output.
func CoherenceSchema() json.RawMessage { return internalprompt.CoherenceSchema() }

// ConcisenessInstructions returns the system instruction for the Conciseness metric.
func ConcisenessInstructions() string { return internalprompt.ConcisenessInstructions() }

// ConcisenessSchema returns the JSON schema for Conciseness metric output.
func ConcisenessSchema() json.RawMessage { return internalprompt.ConcisenessSchema() }

// CompletenessInstructions returns the system instruction for the Completeness metric.
func CompletenessInstructions() string { return internalprompt.CompletenessInstructions() }

// CompletenessSchema returns the JSON schema for Completeness metric output.
func CompletenessSchema() json.RawMessage { return internalprompt.CompletenessSchema() }

// InstructionAdherenceInstructions returns the system instruction for the InstructionAdherence metric.
func InstructionAdherenceInstructions() string {
	return internalprompt.InstructionAdherenceInstructions()
}

// InstructionAdherenceSchema returns the JSON schema for InstructionAdherence metric output.
func InstructionAdherenceSchema() json.RawMessage {
	return internalprompt.InstructionAdherenceSchema()
}

// GEvalInstructions returns the system instruction for a GEval metric with the given criteria.
func GEvalInstructions(criteria string) string { return internalprompt.GEvalInstructions(criteria) }

// CitationAccuracyInstructions returns the system instruction for the CitationAccuracy metric.
func CitationAccuracyInstructions() string { return internalprompt.CitationAccuracyInstructions() }

// CitationAccuracySchema returns the JSON schema for CitationAccuracy metric output.
func CitationAccuracySchema() json.RawMessage { return internalprompt.CitationAccuracySchema() }

// SummarizationQualityInstructions returns the system instruction for the SummarizationQuality metric.
func SummarizationQualityInstructions() string {
	return internalprompt.SummarizationQualityInstructions()
}

// SummarizationQualitySchema returns the JSON schema for SummarizationQuality metric output.
func SummarizationQualitySchema() json.RawMessage {
	return internalprompt.SummarizationQualitySchema()
}
