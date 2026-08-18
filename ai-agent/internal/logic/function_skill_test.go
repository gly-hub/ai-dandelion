package logic

import (
	"strings"
	"testing"

	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
)

func TestFunctionSkillSetupToolPolicy(t *testing.T) {
	setup := &FunctionSkillSetup{
		autoTools:      map[string]bool{"books__create_book": true, "books__list_books": true},
		protectedTools: map[string]bool{"books__delete_book": true},
	}
	if !setup.IsAutoTool("mcp__function-skills__books__create_book") {
		t.Fatal("auto tool should be recognized through MCP namespace")
	}
	if !setup.IsProtectedTool("books__delete_book") {
		t.Fatal("delete tool should require confirmation")
	}
	if setup.IsAutoTool("books__delete_book") {
		t.Fatal("protected tool must not auto execute")
	}
}

func TestFunctionSkillMarkdownUsesPublishedToolNames(t *testing.T) {
	markdown := functionSkillMarkdown(&funcoperation.FunctionSkill{
		Id: "skill-1", Name: "图书管理", ToolPrefix: "books",
		Operations: []*funcoperation.FunctionSkillOperation{{Key: "create_book", Effect: "create", AutoExecute: true}},
	})
	if markdown == "" || !contains(markdown, "books__create_book") {
		t.Fatalf("tool markdown does not describe published tool: %q", markdown)
	}
}

func contains(value, needle string) bool {
	return strings.Contains(value, needle)
}
