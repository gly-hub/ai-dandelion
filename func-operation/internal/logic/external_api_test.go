package logic

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateExternalAPIClient(t *testing.T) {
	key, baseURL, headers, err := validateClient("game-server", "https://game.example.com/", `{"X-Client":"func-operation"}`)
	if err != nil || key != "game-server" || baseURL != "https://game.example.com" || headers != `{"X-Client":"func-operation"}` {
		t.Fatalf("validateClient() = %q, %q, %q, %v", key, baseURL, headers, err)
	}
	if _, _, _, err := validateClient("game-server", "ftp://game.example.com", "{}"); err == nil {
		t.Fatal("validateClient() accepted a non-HTTP URL")
	}
}

func TestValidateExternalAPI(t *testing.T) {
	method, path, _, _, _, err := validateAPI("post", "/v1/orders", "{}", "{}", "{}")
	if err != nil || method != "POST" || path != "/v1/orders" {
		t.Fatalf("validateAPI() = %q, %q, %v", method, path, err)
	}
	if _, _, _, _, _, err := validateAPI("POST", "https://unexpected.example.com", "{}", "{}", "{}"); err == nil {
		t.Fatal("validateAPI() accepted an absolute URL")
	}
}

func TestCallExternalAPIRejectsPrivateNetworkProxy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalAPIClient{}, &model.ExternalAPI{}); err != nil {
		t.Fatal(err)
	}
	store := dao.NewExternalAPI(db)
	if err := store.CreateClient(context.Background(), &model.ExternalAPIClient{UUID: "client-id", ClientKey: "game-server", Name: "Game", BaseURL: "http://127.0.0.1:1", DefaultHeadersJSON: `{"X-Client":"platform"}`, Status: "enabled"}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAPI(context.Background(), &model.ExternalAPI{UUID: "api-id", ClientKey: "game-server", APIKey: "game.order.submit", Name: "Submit", Method: "POST", Path: "/v1/orders", HeadersJSON: `{"X-API":"submit"}`, RequestSchemaJSON: "{}", ResponseSchemaJSON: "{}", Status: "enabled"}); err != nil {
		t.Fatal(err)
	}
	logic := NewExternalAPILogic(store, nil)
	if _, err := logic.CallExternalAPI(context.Background(), "app-id", generatedapp.ExternalAPICallRequest{APIKey: "game.order.submit", Body: map[string]any{"orderId": "O-1"}}); err == nil {
		t.Fatal("generated function external API call unexpectedly reached a private address")
	}
}

func TestExecuteExternalAPIReplacesPathAndProtectsPlatformHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/orders/O 1" || req.URL.Query().Get("limit") != "10" {
			t.Fatalf("unexpected URL: %s", req.URL.String())
		}
		if req.Header.Get("X-Caller") != "editor" || req.Header.Get("X-Platform") != "fixed" {
			t.Fatalf("unexpected headers: %#v", req.Header)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	endpoint := &model.ExternalAPI{Method: "GET", Path: "/v1/orders/{orderId}", Status: "enabled"}
	client := &model.ExternalAPIClient{BaseURL: server.URL, DefaultHeadersJSON: `{"X-Platform":"fixed"}`, Status: "enabled"}
	result, err := executeExternalAPI(context.Background(), endpoint, client, map[string]any{"orderId": "O 1", "limit": 10}, map[string]any{"X-Caller": "editor", "X-Platform": "override"}, nil)
	if err != nil || result.StatusCode != http.StatusOK {
		t.Fatalf("executeExternalAPI() = %#v, %v", result, err)
	}
}

func TestRunPreRequestScriptSetsSignatureHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://game.example.com/v1/orders?source=console", nil)
	req.Header.Set("X-Source", "script-test")
	body := map[string]any{"orderId": "O-1"}
	encoded, _ := json.Marshal(body)
	result, err := runPreRequestScript(`
headers.set('sign', crypto.md5(request.bodyText + 'shared-secret'));
headers.set('timestamp', String(request.timestamp));
headers.set('source', headers.get('X-Source'));

request.query.traceId = 'trace-1';
request.body.playerId = 'P-1';
request.headers['X-Script-Mode'] = 'direct-object';
`, req, body, string(encoded))
	if err != nil {
		t.Fatal(err)
	}
	expected := md5.Sum(append(encoded, []byte("shared-secret")...))
	if req.Header.Get("sign") != fmtHex(expected[:]) {
		t.Fatalf("unexpected sign header: %q", req.Header.Get("sign"))
	}
	if req.Header.Get("timestamp") == "" || req.Header.Get("source") != "script-test" || req.Header.Get("X-Script-Mode") != "direct-object" {
		t.Fatalf("unexpected script headers: %#v", req.Header)
	}
	if req.URL.Query().Get("traceId") != "trace-1" || result.body.(map[string]any)["playerId"] != "P-1" {
		t.Fatalf("unexpected script request mutation: query=%s body=%#v", req.URL.RawQuery, result.body)
	}
}

func TestRunPreRequestScriptReportsErrorsAndTimeouts(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://game.example.com/v1/orders", nil)
	if _, err := runPreRequestScript(`headers.set(`, req, nil, ""); err == nil || !strings.Contains(err.Error(), "pre-request script failed") {
		t.Fatalf("syntax error = %v", err)
	}
	started := time.Now()
	_, err := runPreRequestScript(`while (true) {}`, req, nil, "")
	if err == nil || !strings.Contains(err.Error(), "request script timeout") {
		t.Fatalf("timeout error = %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatalf("script timeout took too long: %s", time.Since(started))
	}
}

func TestExecuteExternalAPIPreRequestScriptOverridesStaticHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		encoded, _ := json.Marshal(map[string]any{"orderId": "O-1"})
		expected := md5.Sum(append(encoded, []byte("shared-secret")...))
		if req.Header.Get("sign") != fmtHex(expected[:]) {
			t.Fatalf("script signature was not sent: %q", req.Header.Get("sign"))
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content type was not available on request: %q", req.Header.Get("Content-Type"))
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	endpoint := &model.ExternalAPI{Method: http.MethodPost, Path: "/v1/orders", HeadersJSON: `{"sign":"endpoint-static"}`, Status: "enabled"}
	client := &model.ExternalAPIClient{BaseURL: server.URL, DefaultHeadersJSON: `{"sign":"client-static"}`, PreRequestScript: `headers.set('sign', crypto.md5(request.bodyText + 'shared-secret'));`, Status: "enabled"}
	result, err := executeExternalAPI(context.Background(), endpoint, client, nil, nil, map[string]any{"orderId": "O-1"})
	if err != nil || result.StatusCode != http.StatusOK {
		t.Fatalf("executeExternalAPI() = %#v, %v", result, err)
	}
}

func TestExecuteExternalAPIPostResponseScriptDecryptsBody(t *testing.T) {
	key := "0123456789abcdef"
	iv := "abcdef0123456789"
	plainText := `{"code":0,"deliveryId":"D-1"}`
	cipherText, err := aesCBCEncryptBase64(plainText, key, iv)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(cipherText))
	}))
	defer server.Close()
	endpoint := &model.ExternalAPI{Method: http.MethodGet, Path: "/v1/orders", HeadersJSON: `{}`, Status: "enabled"}
	client := &model.ExternalAPIClient{BaseURL: server.URL, DefaultHeadersJSON: `{}`, PostResponseScript: `response.bodyText = crypto.aesCbcDecryptBase64(response.bodyText, '0123456789abcdef', 'abcdef0123456789');`, Status: "enabled"}
	result, err := executeExternalAPI(context.Background(), endpoint, client, nil, nil, nil)
	if err != nil || result.StatusCode != http.StatusOK {
		t.Fatalf("executeExternalAPI() = %#v, %v", result, err)
	}
	data, ok := result.Data.(map[string]any)
	if !ok || data["deliveryId"] != "D-1" {
		t.Fatalf("post-response script did not decode response: %#v", result.Data)
	}
}

func TestExecuteExternalAPIPreRequestScriptEncryptsRawBody(t *testing.T) {
	key := "0123456789abcdef"
	iv := "abcdef0123456789"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		plainText, err := aesCBCDecryptBase64(string(raw), key, iv)
		if err != nil || plainText != `{"orderId":"O-1"}` {
			t.Fatalf("request body was not encrypted: %q, %v", plainText, err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	endpoint := &model.ExternalAPI{Method: http.MethodPost, Path: "/v1/orders", HeadersJSON: `{}`, Status: "enabled"}
	client := &model.ExternalAPIClient{BaseURL: server.URL, DefaultHeadersJSON: `{}`, PreRequestScript: `request.bodyText = crypto.aesCbcEncryptBase64(request.bodyText, '0123456789abcdef', 'abcdef0123456789');`, Status: "enabled"}
	result, err := executeExternalAPI(context.Background(), endpoint, client, nil, nil, map[string]any{"orderId": "O-1"})
	if err != nil || result.StatusCode != http.StatusOK {
		t.Fatalf("executeExternalAPI() = %#v, %v", result, err)
	}
}

func TestExternalAPIScriptCryptoAESGCMRoundTrip(t *testing.T) {
	cipherText, err := aesGCMEncryptBase64("payload", "0123456789abcdef", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	plainText, err := aesGCMDecryptBase64(cipherText, "0123456789abcdef", "123456789012")
	if err != nil || plainText != "payload" {
		t.Fatalf("AES-GCM round trip = %q, %v", plainText, err)
	}
}

func TestClientToProtoIncludesLifecycleScripts(t *testing.T) {
	client := clientToProto(&model.ExternalAPIClient{PreRequestScript: "request.headers.sign = 'pre'", PostResponseScript: "response.body = { ok: true }"})
	if client.PreRequestScript == "" || client.PostResponseScript == "" {
		t.Fatalf("scripts were not returned to the client: %#v", client)
	}
}

func fmtHex(value []byte) string { return fmt.Sprintf("%x", value) }

func TestParseExternalAPIDocumentOpenAPI(t *testing.T) {
	operations, err := parseExternalAPIDocument(`{
  "openapi":"3.0.3",
  "paths":{"/v1/orders/{orderId}":{"get":{"tags":["订单"],"summary":"查询订单","parameters":[{"name":"orderId","in":"path","required":true,"schema":{"type":"string"}},{"name":"traceId","in":"header","example":"trace-example","schema":{"type":"string"}}],"responses":{"200":{"description":"ok","content":{"application/json":{"example":{"id":"O-1"}}}}}}}}
}`)
	if err != nil || len(operations) != 1 {
		t.Fatalf("parseExternalAPIDocument() = %#v, %v", operations, err)
	}
	operation := operations[0]
	if operation.group != "订单" || operation.method != "GET" || operation.name != "查询订单" {
		t.Fatalf("unexpected operation: %#v", operation)
	}
	var request map[string]any
	if err := json.Unmarshal([]byte(operation.requestJSON), &request); err != nil {
		t.Fatal(err)
	}
	if len(request["parameters"].([]any)) != 2 || operation.headersJSON != `{"traceId":"trace-example"}` {
		t.Fatalf("request contract = %#v, headers = %s", request, operation.headersJSON)
	}
}

func TestParseExternalAPIDocumentSwaggerBody(t *testing.T) {
	operations, err := parseExternalAPIDocument(`{
  "swagger":"2.0",
  "parameters":{"traceId":{"name":"traceId","in":"header","type":"string","x-example":"trace-1"}},
  "definitions":{
    "CreatePlayerRequest":{"type":"object","required":["name"],"properties":{"name":{"type":"string","description":"玩家名称"},"level":{"type":"integer"}}},
    "CreatePlayerResponse":{"type":"object","properties":{"code":{"type":"integer"},"playerId":{"type":"string","description":"玩家 ID"}}}
  },
  "paths":{"/players":{"post":{"operationId":"createPlayer","parameters":[{"$ref":"#/parameters/traceId"},{"in":"body","name":"body","schema":{"$ref":"#/definitions/CreatePlayerRequest"}}],"responses":{"201":{"description":"created","schema":{"$ref":"#/definitions/CreatePlayerResponse"}}}}}}
}`)
	if err != nil || len(operations) != 1 {
		t.Fatalf("parseExternalAPIDocument() = %#v, %v", operations, err)
	}
	var request map[string]any
	if err := json.Unmarshal([]byte(operations[0].requestJSON), &request); err != nil {
		t.Fatal(err)
	}
	body := request["requestBody"].(map[string]any)
	content := body["content"].(map[string]any)
	if _, ok := content["application/json"]; !ok {
		t.Fatalf("Swagger body was not normalized: %#v", request)
	}
	bodySchema := content["application/json"].(map[string]any)["schema"].(map[string]any)
	if _, ok := bodySchema["properties"].(map[string]any)["name"]; !ok {
		t.Fatalf("Swagger request definition was not expanded: %#v", bodySchema)
	}
	parameters := request["parameters"].([]any)
	if len(parameters) != 2 || parameters[0].(map[string]any)["schema"].(map[string]any)["type"] != "string" {
		t.Fatalf("Swagger global parameter was not normalized: %#v", parameters)
	}
	var response map[string]any
	if err := json.Unmarshal([]byte(operations[0].responseJSON), &response); err != nil {
		t.Fatal(err)
	}
	responseSchema := response["responses"].(map[string]any)["201"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if _, ok := responseSchema["properties"].(map[string]any)["playerId"]; !ok {
		t.Fatalf("Swagger response definition was not expanded: %#v", responseSchema)
	}
}

func TestImportExternalAPIDocumentUpsertsSameRoute(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalAPIGroup{}, &model.ExternalAPI{}); err != nil {
		t.Fatal(err)
	}
	store := dao.NewExternalAPI(db)
	ctx := context.Background()
	group := model.ExternalAPIGroup{UUID: "group-orders", ClientKey: "game-server", Name: "订单", CreatedBy: "user", UpdatedBy: "user"}
	first := model.ExternalAPI{UUID: "api-orders", ClientKey: "game-server", GroupID: "订单", APIKey: "game-server.post.orders", Name: "创建订单", Method: "POST", Path: "/orders", HeadersJSON: `{}`, RequestSchemaJSON: `{"version":1}`, ResponseSchemaJSON: `{}`, Description: "旧说明", Status: "enabled", CreatedBy: "user", UpdatedBy: "user"}
	created, updated, err := store.ImportDocument(ctx, []model.ExternalAPIGroup{group}, []model.ExternalAPI{first})
	if err != nil || created != 1 || updated != 0 {
		t.Fatalf("first import = created %d, updated %d, err %v", created, updated, err)
	}
	second := first
	second.UUID = "api-orders-new"
	second.Name = "创建订单（已更新）"
	second.RequestSchemaJSON = `{"version":2}`
	second.Description = "新说明"
	created, updated, err = store.ImportDocument(ctx, []model.ExternalAPIGroup{group}, []model.ExternalAPI{second})
	if err != nil || created != 0 || updated != 1 {
		t.Fatalf("repeat import = created %d, updated %d, err %v", created, updated, err)
	}
	rows, err := store.ListAPIs(ctx, "game-server")
	if err != nil || len(rows) != 1 {
		t.Fatalf("imported APIs = %#v, %v", rows, err)
	}
	if rows[0].Name != second.Name || rows[0].RequestSchemaJSON != second.RequestSchemaJSON || rows[0].Description != second.Description {
		t.Fatalf("same route was not overwritten: %#v", rows[0])
	}
	created, updated, err = store.ImportDocument(ctx, []model.ExternalAPIGroup{group}, []model.ExternalAPI{second})
	if err != nil || created != 0 || updated != 0 {
		t.Fatalf("unchanged repeat import = created %d, updated %d, err %v", created, updated, err)
	}
}

func TestUploadExternalAPIDocumentAuthenticatesAPIKeyAndSkipsUnchanged(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ExternalAPIClient{}, &model.ExternalAPIGroup{}, &model.ExternalAPI{}); err != nil {
		t.Fatal(err)
	}
	store := dao.NewExternalAPI(db)
	key, keyHash, err := newSwaggerImportKey()
	if err != nil {
		t.Fatal(err)
	}
	client := &model.ExternalAPIClient{UUID: "client-id", ClientKey: "game-server", Name: "游戏服务", BaseURL: "https://game.example.com", DefaultHeadersJSON: `{}`, Status: "enabled", SwaggerImportKeyHash: keyHash}
	if err := store.CreateClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	logic := NewExternalAPILogic(store, nil)
	document := `{"swagger":"2.0","paths":{"/players":{"post":{"summary":"创建玩家","responses":{"200":{"description":"ok"}}}}}}`
	if _, _, err := logic.UploadDocument(context.Background(), &funcoperation.UploadExternalAPIDocumentReq{ClientKey: "game-server", ApiKey: "wrong", DocumentJson: document}); err == nil {
		t.Fatal("upload accepted an invalid API key")
	}
	created, updated, err := logic.UploadDocument(context.Background(), &funcoperation.UploadExternalAPIDocumentReq{ClientKey: "game-server", ApiKey: key, DocumentJson: document})
	if err != nil || created != 1 || updated != 0 {
		t.Fatalf("first upload = created %d, updated %d, err %v", created, updated, err)
	}
	created, updated, err = logic.UploadDocument(context.Background(), &funcoperation.UploadExternalAPIDocumentReq{ClientKey: "game-server", ApiKey: key, DocumentJson: document})
	if err != nil || created != 0 || updated != 0 {
		t.Fatalf("unchanged upload = created %d, updated %d, err %v", created, updated, err)
	}
}
