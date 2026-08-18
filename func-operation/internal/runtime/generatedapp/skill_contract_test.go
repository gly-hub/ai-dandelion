package generatedapp

import "testing"

func TestValidateAgentSkillContract(t *testing.T) {
	valid := &AgentSkillContract{
		Name:       "图书管理",
		ToolPrefix: "book_management",
		Operations: []AgentSkillOperation{
			{Key: "list_books", Action: "book_list", Effect: "read", Fields: []AgentSkillField{{Key: "keyword", Type: "string"}}},
			{Key: "create_book", Action: "book_create", Effect: "create", AutoExecute: true,
				Fields: []AgentSkillField{{Key: "publishDate", Type: "string"}, {Key: "title", Type: "string", Required: true}}},
		},
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
			contract: &AgentSkillContract{Name: "x", ToolPrefix: "books", Operations: []AgentSkillOperation{{Key: "list", Action: "book_list", Effect: "read"}, {Key: "update", Action: "book_update", Effect: "update", AutoExecute: true}}},
			actions:  []string{"book_update"},
		},
		{
			name:     "write only",
			contract: &AgentSkillContract{Name: "x", ToolPrefix: "books", Operations: []AgentSkillOperation{{Key: "create", Action: "book_create", Effect: "create"}}},
			actions:  []string{"book_create"},
		},
		{
			name:     "read action declared as write",
			contract: &AgentSkillContract{Name: "x", ToolPrefix: "books", Operations: []AgentSkillOperation{{Key: "list", Action: "book_list", Effect: "read"}}},
			actions:  []string{"book_list"},
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

func TestParseAgentSkillSnapshotAcceptsStoredContract(t *testing.T) {
	contract, err := ParseAgentSkillSnapshot(`{"name":"图书管理","toolPrefix":"books","operations":[{"key":"create_book","action":"book_create","effect":"create"}]}`)
	if err != nil {
		t.Fatalf("ParseAgentSkillSnapshot() error = %v", err)
	}
	if contract == nil || contract.ToolPrefix != "books" || len(contract.Operations) != 1 {
		t.Fatalf("unexpected contract: %#v", contract)
	}
}
