package logic

import (
	"testing"
)

func TestValidateUsername(t *testing.T) {
	if _, err := validateUsername("ab"); err == nil {
		t.Fatal("expected short username error")
	}
	if username, err := validateUsername(" admin "); err != nil || username != "admin" {
		t.Fatalf("unexpected result: %q %v", username, err)
	}
}

func TestValidateEmail(t *testing.T) {
	if _, err := validateEmail("invalid"); err == nil {
		t.Fatal("expected invalid email error")
	}
	if email, err := validateEmail(" user@example.com "); err != nil || email != "user@example.com" {
		t.Fatalf("unexpected result: %q %v", email, err)
	}
}

func TestNormalizeStatus(t *testing.T) {
	status, err := normalizeStatus(0, 1)
	if err != nil || status != 1 {
		t.Fatalf("expected fallback status 1, got %d %v", status, err)
	}
	if _, err := normalizeStatus(9, 1); err == nil {
		t.Fatal("expected invalid status error")
	}
}
