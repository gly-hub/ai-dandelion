package generatedapp

import "testing"

func TestValidateAgentSkillContract(t *testing.T) {
	valid := &AgentSkillContract{
		Name:       "图书管理",
		ToolPrefix: "book_management",
		Operations: []AgentSkillOperation{{
			Key: "create_book", Action: "book_create", Effect: "create", AutoExecute: true,
			Fields: []AgentSkillField{{Key: "publishDate", Type: "string"}, {Key: "title", Type: "string", Required: true}},
		}},
	}
	if err := ValidateAgentSkillContract(valid, []string{"book_create"}); err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
}

func TestValidateAgentSkillContractRejectsUnsafeOrUndeclaredOperations(t *testing.T) {
	cases := []struct {
		name     string
		contract *AgentSkillContract
		actions  []string
	}{
		{
			name:     "invalid prefix",
			contract: &AgentSkillContract{Name: "x", ToolPrefix: "Book Management", Operations: []AgentSkillOperation{{Key: "read", Action: "read", Effect: "read"}}},
		},
		{
			name:     "missing write action",
			contract: &AgentSkillContract{Name: "x", ToolPrefix: "books", Operations: []AgentSkillOperation{{Key: "update", Action: "book_update", Effect: "update"}}},
		},
		{
			name:     "auto update",
			contract: &AgentSkillContract{Name: "x", ToolPrefix: "books", Operations: []AgentSkillOperation{{Key: "update", Action: "book_update", Effect: "update", AutoExecute: true}}},
			actions:  []string{"book_update"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAgentSkillContract(tc.contract, tc.actions); err == nil {
				t.Fatal("expected contract validation error")
			}
		})
	}
}
