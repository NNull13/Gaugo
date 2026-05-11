package gaugo

import (
	"fmt"
	"strings"
)

// Document is one retrieved context record for a case.
type Document struct {
	ID   string
	Text string
}

// Input is the test input passed to a target system under evaluation.
type Input struct {
	Question string
	Context  []Document
}

// Output is the target system answer under evaluation.
type Output struct {
	Answer string
}

// EvalInput is provided to metrics after a case run completes.
type EvalInput struct {
	CaseName string
	Input    Input
	Output   Output
	Expected Expected
}

// Expected holds simple non-LLM assertions for a case.
type Expected struct {
	Contains []string
}

// Case defines one evaluation scenario.
type Case struct {
	Name     string
	Input    Input
	Expected Expected
}

// CaseOption mutates one Case definition.
type CaseOption func(*Case)

// Question sets the user question for a case.
func Question(question string) CaseOption {
	return func(c *Case) {
		c.Input.Question = strings.TrimSpace(question)
	}
}

// ContextDocs sets retrieved context documents for a case.
func ContextDocs(docs ...Document) CaseOption {
	docsCopy := append([]Document(nil), docs...)
	return func(c *Case) {
		c.Input.Context = docsCopy
	}
}

// Doc is a convenience constructor for Document.
func Doc(id, text string) Document {
	return Document{ID: id, Text: text}
}

// ExpectedContains requires the output answer to contain the given substring.
func ExpectedContains(substr string) CaseOption {
	return func(c *Case) {
		c.Expected.Contains = append(c.Expected.Contains, substr)
	}
}

func validateCase(c Case) error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("case name is required")
	}
	if strings.TrimSpace(c.Input.Question) == "" {
		return fmt.Errorf("case %q input question is required", c.Name)
	}
	for i, needle := range c.Expected.Contains {
		if strings.TrimSpace(needle) == "" {
			return fmt.Errorf("case %q ExpectedContains[%d] must be non-empty", c.Name, i)
		}
	}
	return nil
}
