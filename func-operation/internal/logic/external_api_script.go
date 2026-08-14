package logic

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const externalAPIScriptTimeout = 100 * time.Millisecond

type preRequestScriptResult struct {
	body     any
	bodyText string
	rawBody  bool
}

type postResponseScriptResult struct {
	status  int
	headers http.Header
	data    any
}

func runPreRequestScript(script string, req *http.Request, body any, bodyText string) (*preRequestScriptResult, error) {
	if strings.TrimSpace(script) == "" {
		return nil, nil
	}
	vm, request, headers, query := newExternalAPIScriptVM(req, body, bodyText)
	if err := runExternalAPIScript(vm, script, "pre-request"); err != nil {
		return nil, err
	}
	applyObjectHeaders(req.Header, headers)
	applyObjectQuery(req.URL, query)
	result := &preRequestScriptResult{body: request.Get("body").Export(), bodyText: request.Get("bodyText").String()}
	result.rawBody = result.bodyText != bodyText
	return result, nil
}

func runPostResponseScript(script string, req *http.Request, requestBodyText string, resp *http.Response, raw []byte) (*postResponseScriptResult, error) {
	if strings.TrimSpace(script) == "" {
		return nil, nil
	}
	var responseBody any
	jsonBody := len(raw) > 0 && json.Unmarshal(raw, &responseBody) == nil
	if !jsonBody {
		responseBody = string(raw)
	}
	vm, request, _, _ := newExternalAPIScriptVM(req, nil, requestBodyText)
	response := vm.NewObject()
	responseHeaders := newHeadersObject(vm, resp.Header)
	_ = response.Set("status", resp.StatusCode)
	_ = response.Set("headers", responseHeaders)
	_ = response.Set("body", responseBody)
	_ = response.Set("bodyText", string(raw))
	_ = response.Set("isJSON", jsonBody)
	_ = vm.Set("response", response)
	_ = request // Request remains available to post-response scripts.
	if err := runExternalAPIScript(vm, script, "post-response"); err != nil {
		return nil, err
	}
	status := int(response.Get("status").ToInteger())
	if status < 100 || status > 599 {
		return nil, fmt.Errorf("post-response script failed: response.status must be an HTTP status code")
	}
	bodyText := response.Get("bodyText").String()
	data := response.Get("body").Export()
	if bodyText != string(raw) {
		data = decodeExternalAPIBody([]byte(bodyText))
	}
	return &postResponseScriptResult{status: status, headers: objectHeaders(responseHeaders), data: data}, nil
}

func newExternalAPIScriptVM(req *http.Request, body any, bodyText string) (*goja.Runtime, *goja.Object, *goja.Object, *goja.Object) {
	vm := goja.New()
	headers := newHeadersObject(vm, req.Header)
	query := newQueryObject(vm, req.URL.Query())
	request := vm.NewObject()
	_ = request.Set("method", req.Method)
	_ = request.Set("url", req.URL.String())
	_ = request.Set("headers", headers)
	_ = request.Set("query", query)
	_ = request.Set("body", body)
	_ = request.Set("bodyText", bodyText)
	_ = request.Set("timestamp", time.Now().UnixMilli())
	_ = vm.Set("request", request)
	_ = vm.Set("headers", map[string]any{
		"set":    func(key string, value any) { setObjectString(headers, key, fmt.Sprint(value)) },
		"get":    func(key string) string { return getObjectString(headers, key) },
		"remove": func(key string) { removeObjectKey(headers, key) },
	})
	_ = vm.Set("crypto", map[string]any{
		"md5":                 md5Hex,
		"sha256":              sha256Hex,
		"hmacSha256":          hmacSHA256Hex,
		"base64Encode":        base64Encode,
		"base64Decode":        base64Decode,
		"hexEncode":           hexEncode,
		"hexDecode":           hexDecode,
		"urlEncode":           url.QueryEscape,
		"urlDecode":           url.QueryUnescape,
		"aesCbcEncryptBase64": aesCBCEncryptBase64,
		"aesCbcDecryptBase64": aesCBCDecryptBase64,
		"aesGcmEncryptBase64": aesGCMEncryptBase64,
		"aesGcmDecryptBase64": aesGCMDecryptBase64,
	})
	return vm, request, headers, query
}

func runExternalAPIScript(vm *goja.Runtime, script, phase string) error {
	timer := time.AfterFunc(externalAPIScriptTimeout, func() { vm.Interrupt("request script timeout") })
	defer timer.Stop()
	if _, err := vm.RunString(script); err != nil {
		return fmt.Errorf("%s script failed: %w", phase, err)
	}
	return nil
}

func newHeadersObject(vm *goja.Runtime, values http.Header) *goja.Object {
	out := vm.NewObject()
	for key, entries := range values {
		out.Set(key, strings.Join(entries, ","))
	}
	return out
}

func newQueryObject(vm *goja.Runtime, values url.Values) *goja.Object {
	out := vm.NewObject()
	for key, entries := range values {
		out.Set(key, strings.Join(entries, ","))
	}
	return out
}

func objectHeaders(object *goja.Object) http.Header {
	values := make(http.Header, len(object.Keys()))
	for _, key := range object.Keys() {
		if value := strings.TrimSpace(object.Get(key).String()); value != "" {
			values.Set(key, value)
		}
	}
	return values
}

func applyObjectHeaders(target http.Header, object *goja.Object) {
	for key := range target {
		delete(target, key)
	}
	for key, values := range objectHeaders(object) {
		target[key] = values
	}
}

func applyObjectQuery(target *url.URL, object *goja.Object) {
	values := make(url.Values, len(object.Keys()))
	for _, key := range object.Keys() {
		if value := object.Get(key).String(); value != "" {
			values.Set(key, value)
		}
	}
	target.RawQuery = values.Encode()
}

func setObjectString(object *goja.Object, key, value string) {
	removeObjectKey(object, key)
	object.Set(http.CanonicalHeaderKey(strings.TrimSpace(key)), value)
}

func getObjectString(object *goja.Object, key string) string {
	for _, current := range object.Keys() {
		if strings.EqualFold(current, key) {
			return object.Get(current).String()
		}
	}
	return ""
}

func removeObjectKey(object *goja.Object, key string) {
	for _, current := range object.Keys() {
		if strings.EqualFold(current, key) {
			_ = object.Delete(current)
		}
	}
}

func decodeExternalAPIBody(raw []byte) any {
	var data any
	if len(raw) > 0 && json.Unmarshal(raw, &data) == nil {
		return data
	}
	return string(raw)
}

func md5Hex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256Hex(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func base64Encode(value string) string { return base64.StdEncoding.EncodeToString([]byte(value)) }

func base64Decode(value string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return string(decoded), err
}

func hexEncode(value string) string { return hex.EncodeToString([]byte(value)) }

func hexDecode(value string) (string, error) {
	decoded, err := hex.DecodeString(value)
	return string(decoded), err
}

func aesCBCEncryptBase64(plainText, key, iv string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("AES key must contain 16, 24, or 32 bytes: %w", err)
	}
	if len(iv) != block.BlockSize() {
		return "", fmt.Errorf("AES-CBC IV must contain %d bytes", block.BlockSize())
	}
	data := pkcs7Pad([]byte(plainText), block.BlockSize())
	cipher.NewCBCEncrypter(block, []byte(iv)).CryptBlocks(data, data)
	return base64.StdEncoding.EncodeToString(data), nil
}

func aesCBCDecryptBase64(cipherText, key, iv string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", fmt.Errorf("AES key must contain 16, 24, or 32 bytes: %w", err)
	}
	if len(iv) != block.BlockSize() {
		return "", fmt.Errorf("AES-CBC IV must contain %d bytes", block.BlockSize())
	}
	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("AES-CBC ciphertext must be base64: %w", err)
	}
	if len(data) == 0 || len(data)%block.BlockSize() != 0 {
		return "", fmt.Errorf("AES-CBC ciphertext length is invalid")
	}
	cipher.NewCBCDecrypter(block, []byte(iv)).CryptBlocks(data, data)
	data, err = pkcs7Unpad(data, block.BlockSize())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func aesGCMEncryptBase64(plainText, key, nonce string) (string, error) {
	gcm, err := newAESGCM(key, nonce)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nil, []byte(nonce), []byte(plainText), nil)), nil
}

func aesGCMDecryptBase64(cipherText, key, nonce string) (string, error) {
	gcm, err := newAESGCM(key, nonce)
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("AES-GCM ciphertext must be base64: %w", err)
	}
	plainText, err := gcm.Open(nil, []byte(nonce), data, nil)
	if err != nil {
		return "", fmt.Errorf("AES-GCM decryption failed: %w", err)
	}
	return string(plainText), nil
}

func newAESGCM(key, nonce string) (cipher.AEAD, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("AES key must contain 16, 24, or 32 bytes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("AES-GCM nonce must contain %d bytes", gcm.NonceSize())
	}
	return gcm, nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytesRepeat(byte(padding), padding)...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("AES-CBC plaintext padding is invalid")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, fmt.Errorf("AES-CBC plaintext padding is invalid")
	}
	for _, value := range data[len(data)-padding:] {
		if int(value) != padding {
			return nil, fmt.Errorf("AES-CBC plaintext padding is invalid")
		}
	}
	return data[:len(data)-padding], nil
}

func bytesRepeat(value byte, count int) []byte {
	values := make([]byte, count)
	for index := range values {
		values[index] = value
	}
	return values
}
