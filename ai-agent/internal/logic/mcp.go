package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
)

const (
	mcpRegistryDir  = "mcp_servers"
	mcpRegistryFile = "manifest.json"
)

type MCPLogic struct {
	store mcpRegistryStore
}

type mcpRegistryStore interface {
	ReadRegistry(userID string) (mcpRegistry, error)
	WriteRegistry(userID string, registry mcpRegistry) error
	WalkRegistries(visit func(mcpServerManifest) error) error
}

type fileMCPRegistryStore struct {
	storageDir string
}

type mcpRegistry struct {
	Servers []mcpServerManifest `json:"servers"`
}

type mcpServerManifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Enabled     bool              `json:"enabled"`
	ConfigJSON  string            `json:"configJson,omitempty"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	UpdatedAt   int64             `json:"updatedAt"`
}

func NewMCPLogic(storageDir string) *MCPLogic {
	return &MCPLogic{store: newFileMCPRegistryStore(storageDir)}
}

func newFileMCPRegistryStore(storageDir string) *fileMCPRegistryStore {
	return &fileMCPRegistryStore{storageDir: storageDir}
}

func (m *MCPLogic) ListMCPServers(ctx context.Context, req *aiagent.ListMCPServersReq) ([]*aiagent.AgentMCPServer, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	registry, err := m.store.ReadRegistry(userID)
	if err != nil {
		return nil, err
	}
	servers := make([]*aiagent.AgentMCPServer, 0, len(registry.Servers))
	for _, server := range registry.Servers {
		servers = append(servers, mcpManifestToProto(server))
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].GetUpdatedAt() > servers[j].GetUpdatedAt()
	})
	return servers, nil
}

func (m *MCPLogic) CreateMCPServer(ctx context.Context, req *aiagent.SaveMCPServerReq) (*aiagent.AgentMCPServer, error) {
	return m.saveMCPServer(ctx, req, false)
}

func (m *MCPLogic) UpdateMCPServer(ctx context.Context, req *aiagent.SaveMCPServerReq) (*aiagent.AgentMCPServer, error) {
	return m.saveMCPServer(ctx, req, true)
}

func (m *MCPLogic) DeleteMCPServer(ctx context.Context, req *aiagent.DeleteMCPServerReq) (string, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return "", err
	}
	id := normalizeMCPID(req.GetId())
	if id == "" {
		return "", errors.New("mcp server id is required")
	}
	registry, err := m.store.ReadRegistry(userID)
	if err != nil {
		return "", err
	}
	next := registry.Servers[:0]
	for _, server := range registry.Servers {
		if server.ID != id {
			next = append(next, server)
		}
	}
	registry.Servers = next
	if err := m.store.WriteRegistry(userID, registry); err != nil {
		return "", err
	}
	return id, nil
}

func (m *MCPLogic) ResolveMCPServers(ctx context.Context, userID string, ids []string) (map[string]agent.MCPServerConfig, error) {
	userID, err := userIDFromContextOrRequest(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids = uniqueNormalizedMCPIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	registry, err := m.store.ReadRegistry(userID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]mcpServerManifest, len(registry.Servers))
	for _, server := range registry.Servers {
		if server.Enabled {
			byID[server.ID] = server
		}
	}
	resolved := make(map[string]agent.MCPServerConfig, len(ids))
	for _, id := range ids {
		server, ok := byID[id]
		if !ok {
			continue
		}
		config, err := mcpManifestToAgentConfig(server)
		if err != nil {
			return nil, fmt.Errorf("resolve mcp server %s: %w", id, err)
		}
		resolved[id] = config
	}
	if len(resolved) == 0 {
		return nil, nil
	}
	return resolved, nil
}

func (m *MCPLogic) ResolveLinkedMCPServers(ids []string) (map[string]agent.MCPServerConfig, error) {
	ids = uniqueNormalizedMCPIDs(ids)
	if m == nil || len(ids) == 0 {
		return nil, nil
	}
	needed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		needed[id] = struct{}{}
	}
	resolved := make(map[string]agent.MCPServerConfig, len(ids))
	if err := m.store.WalkRegistries(func(server mcpServerManifest) error {
		if _, ok := needed[server.ID]; !ok || !server.Enabled {
			return nil
		}
		if _, exists := resolved[server.ID]; exists {
			return nil
		}
		config, err := mcpManifestToAgentConfig(server)
		if err != nil {
			return fmt.Errorf("resolve mcp server %s: %w", server.ID, err)
		}
		resolved[server.ID] = config
		return nil
	}); err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, nil
	}
	return resolved, nil
}

func (m *MCPLogic) saveMCPServer(ctx context.Context, req *aiagent.SaveMCPServerReq, requireExisting bool) (*aiagent.AgentMCPServer, error) {
	userID, err := userIDFromContextOrRequest(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	server, err := normalizeMCPServer(req.GetServer())
	if err != nil {
		return nil, err
	}
	registry, err := m.store.ReadRegistry(userID)
	if err != nil {
		return nil, err
	}
	found := false
	for i := range registry.Servers {
		if registry.Servers[i].ID == server.ID {
			registry.Servers[i] = server
			found = true
			break
		}
	}
	if requireExisting && !found {
		return nil, errors.New("mcp server not found")
	}
	if !found {
		registry.Servers = append(registry.Servers, server)
	}
	if err := m.store.WriteRegistry(userID, registry); err != nil {
		return nil, err
	}
	return mcpManifestToProto(server), nil
}

func normalizeMCPServer(server *aiagent.AgentMCPServer) (mcpServerManifest, error) {
	if server == nil {
		return mcpServerManifest{}, errors.New("mcp server is required")
	}
	manifest := mcpServerManifest{
		ID:          normalizeMCPID(server.GetId()),
		Name:        strings.TrimSpace(server.GetName()),
		Description: strings.TrimSpace(server.GetDescription()),
		Type:        normalizeMCPType(server.GetType()),
		Enabled:     server.GetEnabled(),
		ConfigJSON:  strings.TrimSpace(server.GetConfigJson()),
		Command:     strings.TrimSpace(server.GetCommand()),
		Args:        cleanStringList(server.GetArgs()),
		Env:         kvListToMap(server.GetEnv()),
		URL:         strings.TrimSpace(server.GetUrl()),
		Headers:     kvListToMap(server.GetHeaders()),
		UpdatedAt:   time.Now().UnixMilli(),
	}
	if manifest.ConfigJSON != "" {
		config, err := parseMCPConfigJSON(manifest.ConfigJSON)
		if err != nil {
			return mcpServerManifest{}, err
		}
		manifest.ConfigJSON = config.raw
		manifest.Type = config.Type
		manifest.Command = config.Command
		manifest.Args = config.Args
		manifest.Env = config.Env
		manifest.URL = config.URL
		manifest.Headers = config.Headers
	}
	if manifest.ID == "" {
		manifest.ID = normalizeMCPID(manifest.Name)
	}
	if manifest.ID == "" {
		return mcpServerManifest{}, errors.New("mcp server id is required")
	}
	if manifest.Name == "" {
		manifest.Name = manifest.ID
	}
	switch manifest.Type {
	case "stdio":
		if manifest.Command == "" {
			return mcpServerManifest{}, errors.New("mcp stdio command is required")
		}
	case "http", "sse":
		if manifest.URL == "" {
			return mcpServerManifest{}, errors.New("mcp server url is required")
		}
	default:
		return mcpServerManifest{}, errors.New("mcp server type must be stdio, http, or sse")
	}
	return manifest, nil
}

func mcpManifestToProto(server mcpServerManifest) *aiagent.AgentMCPServer {
	return &aiagent.AgentMCPServer{
		Id:          server.ID,
		Name:        server.Name,
		Description: server.Description,
		Type:        server.Type,
		Enabled:     server.Enabled,
		ConfigJson:  server.ConfigJSON,
		Command:     server.Command,
		Args:        append([]string(nil), server.Args...),
		Env:         mapToKVList(server.Env),
		Url:         server.URL,
		Headers:     mapToKVList(server.Headers),
		UpdatedAt:   server.UpdatedAt,
	}
}

func mcpManifestToAgentConfig(server mcpServerManifest) (agent.MCPServerConfig, error) {
	switch server.Type {
	case "stdio":
		return agent.MCPServerConfig{
			Type:    "stdio",
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     copyStringMap(server.Env),
		}, nil
	case "http":
		return agent.MCPServerConfig{
			Type:    "http",
			URL:     server.URL,
			Headers: copyStringMap(server.Headers),
		}, nil
	case "sse":
		return agent.MCPServerConfig{
			Type:    "sse",
			URL:     server.URL,
			Headers: copyStringMap(server.Headers),
		}, nil
	default:
		return agent.MCPServerConfig{}, errors.New("unsupported mcp server type")
	}
}

type parsedMCPConfig struct {
	Type    string
	Command string
	Args    []string
	Env     map[string]string
	URL     string
	Headers map[string]string
	raw     string
}

func parseMCPConfigJSON(value string) (parsedMCPConfig, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return parsedMCPConfig{}, errors.New("mcp config json is invalid")
	}
	config := parsedMCPConfig{
		Type:    normalizeMCPType(stringFromAny(raw["type"])),
		Command: strings.TrimSpace(stringFromAny(raw["command"])),
		Args:    stringSliceFromAny(raw["args"]),
		Env:     stringMapFromAny(raw["env"]),
		URL:     strings.TrimSpace(stringFromAny(raw["url"])),
		Headers: stringMapFromAny(raw["headers"]),
	}
	switch config.Type {
	case "stdio":
		if config.Command == "" {
			return parsedMCPConfig{}, errors.New("mcp config command is required")
		}
	case "http", "sse":
		if config.URL == "" {
			return parsedMCPConfig{}, errors.New("mcp config url is required")
		}
	default:
		return parsedMCPConfig{}, errors.New("mcp config type must be stdio, http, or sse")
	}
	normalized := map[string]any{"type": config.Type}
	switch config.Type {
	case "stdio":
		normalized["command"] = config.Command
		if len(config.Args) > 0 {
			normalized["args"] = config.Args
		}
		if len(config.Env) > 0 {
			normalized["env"] = config.Env
		}
	case "http", "sse":
		normalized["url"] = config.URL
		if len(config.Headers) > 0 {
			normalized["headers"] = config.Headers
		}
	}
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return parsedMCPConfig{}, err
	}
	config.raw = string(data)
	return config, nil
}

func (s *fileMCPRegistryStore) ReadRegistry(userID string) (mcpRegistry, error) {
	data, err := os.ReadFile(s.registryPath(userID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return mcpRegistry{}, nil
		}
		return mcpRegistry{}, err
	}
	var registry mcpRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return mcpRegistry{}, err
	}
	for i := range registry.Servers {
		registry.Servers[i].ID = normalizeMCPID(registry.Servers[i].ID)
		registry.Servers[i].Type = normalizeMCPType(registry.Servers[i].Type)
	}
	return registry, nil
}

func (s *fileMCPRegistryStore) WalkRegistries(visit func(mcpServerManifest) error) error {
	root := s.storageRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		registry, err := s.ReadRegistry(entry.Name())
		if err != nil {
			return err
		}
		for _, server := range registry.Servers {
			if err := visit(server); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *fileMCPRegistryStore) WriteRegistry(userID string, registry mcpRegistry) error {
	path := s.registryPath(userID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sort.Slice(registry.Servers, func(i, j int) bool {
		return registry.Servers[i].UpdatedAt > registry.Servers[j].UpdatedAt
	})
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *fileMCPRegistryStore) registryPath(userID string) string {
	return filepath.Join(s.storageRoot(), normalizePathSegment(userID), mcpRegistryDir, mcpRegistryFile)
}

func (s *fileMCPRegistryStore) storageRoot() string {
	if strings.TrimSpace(s.storageDir) != "" {
		return filepath.Clean(s.storageDir)
	}
	return filepath.Join("data", "skills")
}

func normalizeMCPID(value string) string {
	return normalizeSkillID(value)
}

func normalizeMCPType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "stdio"
	}
	return value
}

func uniqueNormalizedMCPIDs(values []string) []string {
	next := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		id := normalizeMCPID(value)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		next = append(next, id)
	}
	return next
}

func cleanStringList(values []string) []string {
	next := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			next = append(next, value)
		}
	}
	return next
}

func kvListToMap(items []*aiagent.MCPKV) map[string]string {
	out := make(map[string]string)
	for _, item := range items {
		if item == nil {
			continue
		}
		key := strings.TrimSpace(item.GetKey())
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(item.GetValue())
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mapToKVList(values map[string]string) []*aiagent.MCPKV {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]*aiagent.MCPKV, 0, len(keys))
	for _, key := range keys {
		items = append(items, &aiagent.MCPKV{Key: key, Value: values[key]})
	}
	return items
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	next := make(map[string]string, len(values))
	for key, value := range values {
		next[key] = value
	}
	return next
}

func stringFromAny(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func stringSliceFromAny(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	next := make([]string, 0, len(items))
	for _, item := range items {
		text := stringFromAny(item)
		if text != "" {
			next = append(next, text)
		}
	}
	return next
}

func stringMapFromAny(value any) map[string]string {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	next := make(map[string]string, len(raw))
	for key, value := range raw {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		next[key] = stringFromAny(value)
	}
	if len(next) == 0 {
		return nil
	}
	return next
}
