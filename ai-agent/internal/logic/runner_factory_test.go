package logic

import "testing"

func TestResolveProviderValueUsesSelectedModelChannel(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "https://api.deepseek.com/anthropic")

	got := resolveProviderValue("https://api.seawork.ai/llm", "ANTHROPIC_BASE_URL", true)
	if got != "https://api.seawork.ai/llm" {
		t.Fatalf("selected model channel should override process endpoint, got %q", got)
	}
}

func TestResolveProviderValueUsesEnvironmentForDefaultChannel(t *testing.T) {
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "environment-token")

	got := resolveProviderValue("configured-token", "ANTHROPIC_AUTH_TOKEN", false)
	if got != "environment-token" {
		t.Fatalf("default channel should use environment token, got %q", got)
	}
}
