package generatedapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/google/uuid"
)

const templateAppID = "hello-greeting"
const manifestSchemaVersion = "v2"

type ScaffoldInput struct {
	AppID       string
	Name        string
	Description string
	SessionID   string
	TablePrefix string
	Summary     string
	Highlights  []string
	CorePages   []string
	DataModels  []string
	APIs        []string
	NextSteps   []string
}

func (s *Service) CreateAppScaffold(ctx context.Context, input ScaffoldInput) (*model.GeneratedApp, error) {
	if s == nil {
		return nil, errors.New("generated app runtime is not configured")
	}

	appID := strings.TrimSpace(input.AppID)
	if appID == "" {
		appID = strings.ToLower(uuid.NewString())
	}
	if appID == "" {
		return nil, errors.New("app id is required")
	}
	appID = slugifyAppID(appID)
	if appID == "" {
		return nil, errors.New("app id is invalid")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = appID
	}
	description := strings.TrimSpace(input.Description)

	appDir := filepath.Join(s.rootDir, appID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("create app directory: %w", err)
	}

	manifest := map[string]any{
		"schemaVersion":   manifestSchemaVersion,
		"id":              appID,
		"name":            name,
		"version":         "v0.1.0",
		"description":     firstNonEmpty(description, fmt.Sprintf("功能“%s”的可操作页面。", name)),
		"export":          "handle",
		"actions":         []string{},
		"frontendFile":    "frontend.js",
		"backendSource":   "backend",
		"backendModule":   "backend.wasm",
		"sourceSessionId": input.SessionID,
		"summary":         input.Summary,
		"highlights":      input.Highlights,
		"corePages":       input.CorePages,
		"dataModels":      []map[string]any{},
		"apis":            input.APIs,
		"nextSteps":       input.NextSteps,
		"configKeys":      []string{},
	}
	if len(input.DataModels) > 0 {
		manifest["dataModelHints"] = input.DataModels
	}
	manifest["tablePrefix"] = normalizeTableName(input.TablePrefix)
	manifest["tables"] = []map[string]string{}
	if err := writeJSONFile(filepath.Join(appDir, "manifest.json"), manifest); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(appDir, "frontend.js"), []byte(frontendTemplate(appID, name, description, input)), 0o644); err != nil {
		return nil, fmt.Errorf("write frontend template: %w", err)
	}
	frontendDir := filepath.Join(appDir, "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		return nil, fmt.Errorf("create frontend module directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "dom.js"), []byte(frontendDOMTemplate()), 0o644); err != nil {
		return nil, fmt.Errorf("write frontend dom template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "state.js"), []byte(frontendStateTemplate()), 0o644); err != nil {
		return nil, fmt.Errorf("write frontend state template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "styles.js"), []byte(frontendStylesTemplate()), 0o644); err != nil {
		return nil, fmt.Errorf("write frontend styles template: %w", err)
	}
	backendDir := filepath.Join(appDir, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		return nil, fmt.Errorf("create backend directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "main.go"), []byte(backendMainTemplate(name)), 0o644); err != nil {
		return nil, fmt.Errorf("write backend template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "platform.go"), []byte(backendPlatformTemplate()), 0o644); err != nil {
		return nil, fmt.Errorf("write backend platform template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "models.go"), []byte(backendModelsTemplate()), 0o644); err != nil {
		return nil, fmt.Errorf("write backend models template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, "handlers.go"), []byte(backendHandlersTemplate()), 0o644); err != nil {
		return nil, fmt.Errorf("write backend handlers template: %w", err)
	}
	if err := ensureBackendWASM(appDir, s.rootDir); err != nil {
		return nil, err
	}
	return s.CandidateApp(ctx, appID)
}

func writeJSONFile(path string, payload any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json file %q: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write json file %q: %w", path, err)
	}
	return nil
}

func ensureBackendWASM(appDir string, rootDir string) error {
	templatePath := filepath.Join(rootDir, templateAppID, "backend.wasm")
	raw, err := os.ReadFile(templatePath)
	if err == nil {
		if err := os.WriteFile(filepath.Join(appDir, "backend.wasm"), raw, 0o644); err != nil {
			return fmt.Errorf("write app wasm: %w", err)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read template wasm %q: %w", templatePath, err)
	}
	return buildScaffoldWASM(appDir)
}

func buildScaffoldWASM(appDir string) error {
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", filepath.Join(appDir, "backend.wasm"), ".")
	cmd.Dir = filepath.Join(appDir, "backend")
	cmd.Env = append(os.Environ(),
		"GOOS=wasip1",
		"GOARCH=wasm",
		"GOCACHE=/private/tmp/ai-dandelion-go-cache",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build scaffold wasm: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

var invalidAppIDPattern = regexp.MustCompile(`[^a-z0-9-]+`)

func slugifyAppID(name string) string {
	value := strings.ToLower(strings.TrimSpace(name))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = invalidAppIDPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return ""
	}
	return value
}

var invalidTableNamePattern = regexp.MustCompile(`[^a-z0-9_]+`)

func normalizeTableName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = invalidTableNamePattern.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return ""
	}
	return value
}

func frontendTemplate(appID string, name string, description string, input ScaffoldInput) string {
	return fmt.Sprintf(`import { escapeHTML } from './frontend/dom.js'
import { injectStyles } from './frontend/styles.js'

export function render(container, context) {
  container.innerHTML = [
    '<div class="generated-function-page">',
    '  <section class="generated-function-empty" data-app-id="' + escapeHTML(context.app.id) + '">',
    '    <span class="generated-function-dot"></span>',
    '    <h3>%s</h3>',
    '    <p>%s</p>',
    '    <div class="generated-function-hint">功能页面正在生成或等待生成结果匹配，完成后这里会自动替换为可操作的业务页面。</div>',
    '  </section>',
    '</div>',
  ].join('')
  injectStyles(container)
}

export function dispose(container) {
  container.innerHTML = ''
}
`, escapeJS(name), escapeJS(firstNonEmpty(description, firstNonEmpty(input.Summary, "页面还未生成完成，请在左侧生成会话继续生成或等待自动匹配。"))))
}

func frontendDOMTemplate() string {
	return `export function escapeHTML(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}
`
}

func frontendStateTemplate() string {
	return `export const scaffoldState = {
  status: 'pending',
}

export function createInitialRows() {
  return []
}

export function nextRowId(rows) {
  return 'ROW-' + String((rows || []).length + 1).padStart(3, '0')
}
`
}

func frontendStylesTemplate() string {
	return `export function injectStyles(root) {
  if (!root || root.querySelector('.generated-function-styles')) return
  const style = document.createElement('style')
  style.className = 'generated-function-styles'
  style.textContent = [
    '.generated-function-page{display:grid;min-height:280px;color:#172033;}',
    '.generated-function-empty{display:grid;place-items:center;align-content:center;gap:12px;min-height:280px;padding:28px;border:1px dashed #cbd5e1;border-radius:8px;background:#f8fafc;text-align:center;}',
    '.generated-function-dot{width:10px;height:10px;border-radius:999px;background:#1677ff;box-shadow:0 0 0 6px rgba(22,119,255,.12);}',
    '.generated-function-empty h3{margin:0;font-size:20px;font-weight:800;letter-spacing:0;color:#111827;}',
    '.generated-function-empty p{max-width:560px;margin:0;color:#475569;line-height:1.7;}',
    '.generated-function-hint{max-width:560px;padding:10px 12px;border-radius:8px;background:white;color:#64748b;font-size:13px;line-height:1.6;box-shadow:0 1px 2px rgba(15,23,42,.04);}',
    '@media(max-width:760px){.generated-function-empty{min-height:220px;padding:22px;}}',
  ].join('')
  root.appendChild(style)
}
`
}

func backendMainTemplate(name string) string {
	return fmt.Sprintf(`//go:build wasip1

package main

import (
	"encoding/json"
	"unsafe"
)

var requestBuffer []byte

func main() {}

//go:wasmexport alloc
func alloc(size uint32) uint32 {
	requestBuffer = make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&requestBuffer[0])))
}

//go:wasmexport handle
func handle(reqPtr, reqLen uint32) uint64 {
	reqBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(reqPtr))), reqLen)
	var req InvokeRequest
	if len(reqBytes) > 0 {
		if err := json.Unmarshal(reqBytes, &req); err != nil {
			return storeResponse(InvokeResponse{Error: "JSON解析失败: " + err.Error()})
		}
	}
	return handleRequest(req)
}

// %s
// Replace the scaffold implementation above with your real business logic.
`, name)
}

func backendModelsTemplate() string {
	return `//go:build wasip1

package main

import "encoding/json"

type InvokeRequest struct {
	Action string          ` + "`json:\"action\"`" + `
	Data   json.RawMessage ` + "`json:\"data,omitempty\"`" + `
}

type InvokeResponse struct {
	Success bool   ` + "`json:\"success\"`" + `
	Error   string ` + "`json:\"error,omitempty\"`" + `
	Data    any    ` + "`json:\"data,omitempty\"`" + `
	Rows    any    ` + "`json:\"rows,omitempty\"`" + `
	Total   int    ` + "`json:\"total,omitempty\"`" + `
}
`
}

func backendHandlersTemplate() string {
	return `//go:build wasip1

package main

func handleRequest(req InvokeRequest) uint64 {
	switch req.Action {
	case "", "status":
		return storeResponse(InvokeResponse{
			Success: true,
			Data: map[string]any{
				"status": "pending_generation",
				"message": "业务页面脚手架已创建，等待生成结果接管实际逻辑。",
			},
		})
	default:
		return storeResponse(InvokeResponse{
			Success: false,
			Error:   "generated app scaffold has no business handler for action: " + req.Action,
		})
	}
}
`
}

func backendPlatformTemplate() string {
	return `//go:build wasip1

package main

import (
	"encoding/json"
	"unsafe"
)

//go:wasmimport platform result_store
func resultStore(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_list
func hostDataList(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_create
func hostDataCreate(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_update
func hostDataUpdate(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_delete
func hostDataDelete(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_join_query
func hostDataJoinQuery(reqPtr, reqLen uint32) uint64

//go:wasmimport platform data_run_query
func hostDataRunQuery(reqPtr, reqLen uint32) uint64

//go:wasmimport platform call_capability
func hostCallCapability(reqPtr, reqLen uint32) uint64

//go:wasmimport platform result_len
func resultLen(handle uint64) uint32

//go:wasmimport platform result_read
func resultRead(handle uint64, outPtr uint32) uint32

func storeResponse(data any) uint64 {
	raw, _ := json.Marshal(data)
	return resultStore(uint32(uintptr(unsafe.Pointer(&raw[0]))), uint32(len(raw)))
}

type DataListRequest struct {
	Model   string      ` + "`json:\"model\"`" + `
	Where   []DataWhere ` + "`json:\"where,omitempty\"`" + `
	OrderBy []DataOrder ` + "`json:\"orderBy,omitempty\"`" + `
	Page    DataPage    ` + "`json:\"page,omitempty\"`" + `
}

type DataWhere struct {
	Field string ` + "`json:\"field\"`" + `
	Op    string ` + "`json:\"op\"`" + `
	Value any    ` + "`json:\"value\"`" + `
}

type DataOrder struct {
	Field     string ` + "`json:\"field\"`" + `
	Direction string ` + "`json:\"direction\"`" + `
}

type DataPage struct {
	Limit int ` + "`json:\"limit,omitempty\"`" + `
}

type DataWriteRequest struct {
	Model  string         ` + "`json:\"model\"`" + `
	ID     any            ` + "`json:\"id,omitempty\"`" + `
	Record map[string]any ` + "`json:\"record,omitempty\"`" + `
	Patch  map[string]any ` + "`json:\"patch,omitempty\"`" + `
}

type DataJoinRequest struct {
	From    string      ` + "`json:\"from\"`" + `
	Joins   []DataJoin  ` + "`json:\"joins,omitempty\"`" + `
	Select  []string    ` + "`json:\"select\"`" + `
	Where   []DataWhere ` + "`json:\"where,omitempty\"`" + `
	OrderBy []DataOrder ` + "`json:\"orderBy,omitempty\"`" + `
	Limit   int         ` + "`json:\"limit,omitempty\"`" + `
}

type DataJoin struct {
	Relation string ` + "`json:\"relation\"`" + `
	Type     string ` + "`json:\"type,omitempty\"`" + `
}

type DataRunQueryRequest struct {
	Query  string         ` + "`json:\"query\"`" + `
	Params map[string]any ` + "`json:\"params,omitempty\"`" + `
}

type CapabilityCallRequest struct {
	AppID      string         ` + "`json:\"appId\"`" + `
	Capability string         ` + "`json:\"capability\"`" + `
	Params     map[string]any ` + "`json:\"params,omitempty\"`" + `
}

type DataListResult struct {
	Rows  []map[string]any ` + "`json:\"rows\"`" + `
	Total int              ` + "`json:\"total\"`" + `
	Error string           ` + "`json:\"error,omitempty\"`" + `
}

type DataWriteResult struct {
	ID           any            ` + "`json:\"id,omitempty\"`" + `
	RowsAffected int64          ` + "`json:\"rowsAffected,omitempty\"`" + `
	Record       map[string]any ` + "`json:\"record,omitempty\"`" + `
	Error        string         ` + "`json:\"error,omitempty\"`" + `
}

func dataList(req DataListRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostDataList, req, &result)
	return result
}

func dataCreate(req DataWriteRequest) DataWriteResult {
	var result DataWriteResult
	callPlatformJSON(hostDataCreate, req, &result)
	return result
}

func dataUpdate(req DataWriteRequest) DataWriteResult {
	var result DataWriteResult
	callPlatformJSON(hostDataUpdate, req, &result)
	return result
}

func dataDelete(model string, id any) DataWriteResult {
	var result DataWriteResult
	callPlatformJSON(hostDataDelete, map[string]any{"model": model, "id": id}, &result)
	return result
}

func dataJoinQuery(req DataJoinRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostDataJoinQuery, req, &result)
	return result
}

func dataRunQuery(req DataRunQueryRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostDataRunQuery, req, &result)
	return result
}

func callCapability(req CapabilityCallRequest) DataListResult {
	var result DataListResult
	callPlatformJSON(hostCallCapability, req, &result)
	return result
}

func callPlatformJSON(call func(uint32, uint32) uint64, req any, out any) {
	raw, _ := json.Marshal(req)
	handle := call(uint32(uintptr(unsafe.Pointer(&raw[0]))), uint32(len(raw)))
	length := resultLen(handle)
	if length == 0 {
		return
	}
	buf := make([]byte, length)
	resultRead(handle, uint32(uintptr(unsafe.Pointer(&buf[0]))))
	_ = json.Unmarshal(buf, out)
}
`
}

func escapeJS(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		"\n", `\n`,
		"\r", ``,
	)
	return replacer.Replace(value)
}

func renderListHTML(items []string, fallback string) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		lines = append(lines, "        '<li>' + escapeHTML('"+escapeJS(item)+"') + '</li>'")
	}
	if len(lines) == 0 {
		lines = append(lines, "        '<li>' + escapeHTML('"+escapeJS(fallback)+"') + '</li>'")
	}
	return "[\n" + strings.Join(lines, ",\n") + "\n      ].join('')"
}
