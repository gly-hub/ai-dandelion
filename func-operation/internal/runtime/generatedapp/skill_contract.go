package generatedapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var skillIdentifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,79}$`)

// Generated app payloads commonly use camelCase field names (for example
// publishDate).  Tool prefixes and operation keys remain lowercase stable
// identifiers, while field keys may also contain ASCII capitals.
var skillFieldPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9_]{0,79}$`)

type AgentSkillContract struct {
	Name        string                `json:"name"`
	ToolPrefix  string                `json:"toolPrefix"`
	Description string                `json:"description,omitempty"`
	Operations  []AgentSkillOperation `json:"operations"`
}

type AgentSkillOperation struct {
	Key         string            `json:"key"`
	Action      string            `json:"action"`
	Effect      string            `json:"effect"`
	Description string            `json:"description,omitempty"`
	Fields      []AgentSkillField `json:"fields,omitempty"`
	AutoExecute bool              `json:"autoExecute,omitempty"`
}

type AgentSkillField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label,omitempty"`
	Type        string   `json:"type"`
	Required    bool     `json:"required,omitempty"`
	EnumValues  []string `json:"enumValues,omitempty"`
	Description string   `json:"description,omitempty"`
}

// ParseAgentSkillContract reads the optional contract from an immutable manifest snapshot.
func ParseAgentSkillContract(raw string) (*AgentSkillContract, error) {
	contract, _, err := ParseAgentSkillManifest(raw)
	return contract, err
}

// ParseAgentSkillManifest returns the optional contract and the state-changing
// action declarations from an immutable manifest snapshot.
func ParseAgentSkillManifest(raw string) (*AgentSkillContract, []string, error) {
	var envelope struct {
		Actions    []string            `json:"actions"`
		AgentSkill *AgentSkillContract `json:"agentSkill"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return nil, nil, err
	}
	if envelope.AgentSkill == nil {
		return nil, envelope.Actions, nil
	}
	return envelope.AgentSkill, envelope.Actions, nil
}

func ValidateAgentSkillContract(contract *AgentSkillContract, declaredActions []string) error {
	if contract == nil {
		return errors.New("agentSkill is required")
	}
	contract.Name = strings.TrimSpace(contract.Name)
	contract.ToolPrefix = strings.TrimSpace(contract.ToolPrefix)
	if contract.Name == "" || !skillIdentifierPattern.MatchString(contract.ToolPrefix) {
		return errors.New("agentSkill name and lowercase toolPrefix are required")
	}
	if len(contract.Operations) == 0 {
		return errors.New("agentSkill operations are required")
	}
	declared := make(map[string]struct{}, len(declaredActions))
	for _, action := range declaredActions {
		declared[strings.TrimSpace(action)] = struct{}{}
	}
	seen := make(map[string]struct{}, len(contract.Operations))
	for i := range contract.Operations {
		op := &contract.Operations[i]
		op.Key, op.Action, op.Effect = strings.TrimSpace(op.Key), strings.TrimSpace(op.Action), strings.ToLower(strings.TrimSpace(op.Effect))
		if !skillIdentifierPattern.MatchString(op.Key) || op.Action == "" {
			return fmt.Errorf("agentSkill operation %d has invalid key or action", i)
		}
		if _, exists := seen[op.Key]; exists {
			return fmt.Errorf("duplicate agentSkill operation %q", op.Key)
		}
		seen[op.Key] = struct{}{}
		switch op.Effect {
		case "read", "create", "update", "delete", "execute":
		default:
			return fmt.Errorf("operation %q has unsupported effect", op.Key)
		}
		if op.Effect != "read" {
			if _, ok := declared[op.Action]; !ok {
				return fmt.Errorf("operation %q action %q is not declared", op.Key, op.Action)
			}
		}
		if op.AutoExecute && op.Effect != "create" {
			return fmt.Errorf("operation %q may auto execute only for create", op.Key)
		}
		fieldKeys := make(map[string]struct{}, len(op.Fields))
		for j := range op.Fields {
			field := &op.Fields[j]
			field.Key, field.Type = strings.TrimSpace(field.Key), strings.ToLower(strings.TrimSpace(field.Type))
			if !skillFieldPattern.MatchString(field.Key) {
				return fmt.Errorf("operation %q field %d key is invalid", op.Key, j)
			}
			if _, exists := fieldKeys[field.Key]; exists {
				return fmt.Errorf("operation %q has duplicate field %q", op.Key, field.Key)
			}
			fieldKeys[field.Key] = struct{}{}
			switch field.Type {
			case "string", "number", "integer", "boolean", "enum":
			default:
				return fmt.Errorf("operation %q field %q has unsupported type", op.Key, field.Key)
			}
			if field.Type == "enum" && len(field.EnumValues) == 0 {
				return fmt.Errorf("operation %q enum field %q has no values", op.Key, field.Key)
			}
		}
	}
	sort.Slice(contract.Operations, func(i, j int) bool { return contract.Operations[i].Key < contract.Operations[j].Key })
	return nil
}
