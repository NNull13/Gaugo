package gaugo

import (
	"errors"
	"fmt"
	"strings"
)

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

// ExpectedContains requires the output answer to contain the given substring.
func ExpectedContains(substr string) CaseOption {
	return func(c *Case) {
		c.Expected.Contains = append(c.Expected.Contains, substr)
	}
}

// ExpectedAnswer sets the reference answer for metrics that need ground truth.
func ExpectedAnswer(answer string) CaseOption {
	return func(c *Case) {
		c.Expected.Answer = strings.TrimSpace(answer)
	}
}

// ExpectedInstructions sets the reference instructions for
// instruction-following metrics.
func ExpectedInstructions(instructions string) CaseOption {
	return func(c *Case) {
		c.Expected.Instructions = strings.TrimSpace(instructions)
	}
}

func validateCase(c Case) error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("case name is required")
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
