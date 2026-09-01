package logic

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

const systemToolsMCPServerID = "system-tools"

type SystemToolRuntime struct {
	client systemproto.SystemServiceClient
}

func NewSystemToolRuntime(client systemproto.SystemServiceClient) *SystemToolRuntime {
	return &SystemToolRuntime{client: client}
}

func (r *SystemToolRuntime) Server(ctx context.Context) (claudeagentsdk.SDKMCPServerConfig, error) {
	if r == nil || r.client == nil {
		return claudeagentsdk.SDKMCPServerConfig{}, nil
	}
	callCtx := authctx.ForwardUserContext(context.WithoutCancel(ctx))
	tools := []claudeagentsdk.MCPTool{
		claudeagentsdk.NewMCPTool("upload_remote_file", "Upload a model-produced remote file URL into the platform CDN and return the CDN URL. Use this for images, audio, video, and documents.", map[string]any{"type": "object", "required": []string{"url"}, "properties": map[string]any{"url": map[string]any{"type": "string"}, "fileName": map[string]any{"type": "string"}, "contentType": map[string]any{"type": "string"}}}, func(input map[string]any) (claudeagentsdk.MCPToolResult, error) {
			resp, err := r.client.UploadRemoteFile(callCtx, &systemproto.UploadRemoteFileReq{Url: stringInput(input, "url"), FileName: stringInput(input, "fileName"), ContentType: stringInput(input, "contentType")})
			if err != nil {
				return toolJSON(map[string]any{"error": err.Error()}, true), nil
			}
			return toolJSON(map[string]any{"url": resp.GetUrl(), "fileName": resp.GetFileName(), "contentType": resp.GetContentType()}, false), nil
		}),
	}
	configResp, err := r.client.GetAgentConfig(callCtx, &systemproto.GetAgentConfigReq{})
	if err != nil {
		return claudeagentsdk.CreateSDKMCPServer(systemToolsMCPServerID, "1.0.0", tools), nil
	}
	if configResp.GetConfig().GetImageToolEnabled() {
		modelResp, modelErr := r.client.GetAgentModelRuntime(callCtx, &systemproto.GetAgentModelRuntimeReq{Id: configResp.GetConfig().GetImageModelId()})
		if modelErr == nil && modelResp.GetModel().GetType() == "image" && modelResp.GetModel().GetStatus() == 1 {
			tools = append(tools, r.imageTool(callCtx, modelResp.GetModel()))
		}
	}
	return claudeagentsdk.CreateSDKMCPServer(systemToolsMCPServerID, "1.0.0", tools), nil
}

func (r *SystemToolRuntime) imageTool(ctx context.Context, item *systemproto.AgentModel) claudeagentsdk.MCPTool {
	return claudeagentsdk.NewMCPTool("generate_image", "Generate an image from a prompt, mirror it into CDN, and return the CDN URL.", map[string]any{"type": "object", "required": []string{"prompt"}, "properties": map[string]any{"prompt": map[string]any{"type": "string"}, "size": map[string]any{"type": "string"}, "fileName": map[string]any{"type": "string"}}}, func(input map[string]any) (claudeagentsdk.MCPToolResult, error) {
		prompt := strings.TrimSpace(stringInput(input, "prompt"))
		if prompt == "" {
			return toolJSON(map[string]any{"error": "prompt is required"}, true), nil
		}
		baseURL := strings.TrimRight(strings.TrimSpace(item.GetBaseUrl()), "/")
		if baseURL == "" {
			return toolJSON(map[string]any{"error": "image model base URL is not configured"}, true), nil
		}
		payload := map[string]any{"model": item.GetModel(), "prompt": prompt, "n": 1}
		if size := stringInput(input, "size"); size != "" {
			payload["size"] = size
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/images/generations", bytes.NewReader(body))
		if err != nil {
			return toolJSON(map[string]any{"error": err.Error()}, true), nil
		}
		req.Header.Set("Content-Type", "application/json")
		if token := strings.TrimSpace(item.GetAuthToken()); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
		if err != nil {
			return toolJSON(map[string]any{"error": err.Error()}, true), nil
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return toolJSON(map[string]any{"error": fmt.Sprintf("image provider: %s", strings.TrimSpace(string(data)))}, true), nil
		}
		var result struct {
			Data []struct {
				URL string `json:"url"`
				B64 string `json:"b64_json"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return toolJSON(map[string]any{"error": err.Error()}, true), nil
		}
		if len(result.Data) == 0 {
			return toolJSON(map[string]any{"error": "image provider returned no image"}, true), nil
		}
		remoteURL := result.Data[0].URL
		if strings.HasPrefix(remoteURL, "data:") {
			parts := strings.SplitN(remoteURL, ",", 2)
			if len(parts) == 2 {
				decoded, decErr := base64.StdEncoding.DecodeString(parts[1])
				if decErr == nil {
					stored, uploadErr := r.client.UploadInlineFile(ctx, &systemproto.UploadInlineFileReq{Data: decoded, FileName: stringInput(input, "fileName"), ContentType: "image/png"})
					if uploadErr != nil {
						return toolJSON(map[string]any{"error": uploadErr.Error()}, true), nil
					}
					return toolJSON(map[string]any{"url": stored.GetUrl(), "fileName": stored.GetFileName(), "contentType": stored.GetContentType()}, false), nil
				}
			}
		}
		if remoteURL == "" && result.Data[0].B64 != "" {
			decoded, decErr := base64.StdEncoding.DecodeString(result.Data[0].B64)
			if decErr != nil {
				return toolJSON(map[string]any{"error": decErr.Error()}, true), nil
			}
			stored, uploadErr := r.client.UploadInlineFile(ctx, &systemproto.UploadInlineFileReq{Data: decoded, FileName: stringInput(input, "fileName"), ContentType: "image/png"})
			if uploadErr != nil {
				return toolJSON(map[string]any{"error": uploadErr.Error()}, true), nil
			}
			return toolJSON(map[string]any{"url": stored.GetUrl(), "fileName": stored.GetFileName(), "contentType": stored.GetContentType()}, false), nil
		}
		if remoteURL == "" {
			return toolJSON(map[string]any{"error": "image provider returned no URL"}, true), nil
		}
		stored, err := r.client.UploadRemoteFile(ctx, &systemproto.UploadRemoteFileReq{Url: remoteURL, FileName: stringInput(input, "fileName"), ContentType: "image/png"})
		if err != nil {
			return toolJSON(map[string]any{"error": err.Error()}, true), nil
		}
		return toolJSON(map[string]any{"url": stored.GetUrl(), "sourceUrl": remoteURL, "fileName": stored.GetFileName(), "contentType": stored.GetContentType()}, false), nil
	})
}

func toolJSON(value any, isError bool) claudeagentsdk.MCPToolResult {
	data, _ := json.Marshal(value)
	return claudeagentsdk.MCPToolResult{Content: []claudeagentsdk.MCPToolContent{{Type: "text", Text: string(data)}}, IsError: isError}
}
