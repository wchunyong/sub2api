package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/mcp"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MCPHandler struct {
	openAI               *OpenAIGatewayHandler
	imageStorageResolver service.ImageStorageResolver
}

func NewMCPHandler(openAI *OpenAIGatewayHandler) *MCPHandler {
	return &MCPHandler{openAI: openAI}
}

func NewMCPHandlerWithImageStorage(openAI *OpenAIGatewayHandler, imageStorage *service.ImageStorageSettingService) *MCPHandler {
	h := NewMCPHandler(openAI)
	if imageStorage != nil {
		h.imageStorageResolver = imageStorage.Resolver()
	}
	return h
}

func (h *MCPHandler) Serve(c *gin.Context) {
	server := mcp.NewServer(&mcpImageGateway{source: c, openAI: h.openAI, imageStorageResolver: h.imageStorageResolver})
	server.ServeHTTP(c.Writer, c.Request)
}

type mcpImageGateway struct {
	source               *gin.Context
	openAI               *OpenAIGatewayHandler
	imageStorageResolver service.ImageStorageResolver
}

func (g *mcpImageGateway) GenerateImage(ctx context.Context, input mcp.GenerateImageInput) (mcp.ImageResult, error) {
	if g == nil || g.source == nil || g.openAI == nil {
		return mcp.ImageResult{}, mcp.NewToolError(mcp.ErrInternal, "", fmt.Errorf("image gateway unavailable"))
	}
	input.Model = g.resolveImageModel(input.Model)
	if err := g.validateImageModel(input.Model); err != nil {
		return mcp.ImageResult{}, err
	}
	body, err := buildMCPImageRequest(input.Model, input.Prompt, input.Size, input.Quality, input.OutputFormat, input.N, nil)
	if err != nil {
		return mcp.ImageResult{}, err
	}
	return g.callImages(ctx, "/v1/images/generations", body, input.Model, input.N)
}

func (g *mcpImageGateway) EditImage(ctx context.Context, input mcp.EditImageInput) (mcp.ImageResult, error) {
	if g == nil || g.source == nil || g.openAI == nil {
		return mcp.ImageResult{}, mcp.NewToolError(mcp.ErrInternal, "", fmt.Errorf("image gateway unavailable"))
	}
	input.Model = g.resolveImageModel(input.Model)
	if err := g.validateImageModel(input.Model); err != nil {
		return mcp.ImageResult{}, err
	}
	body, err := buildMCPImageRequest(input.Model, input.Prompt, input.Size, input.Quality, input.OutputFormat, 1, []map[string]string{{"image_url": input.Image}})
	if err != nil {
		return mcp.ImageResult{}, err
	}
	return g.callImages(ctx, "/v1/images/edits", body, input.Model, 1)
}

func (g *mcpImageGateway) resolveImageModel(model string) string {
	model = strings.TrimSpace(model)
	if model != "" {
		return model
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(g.source)
	if !ok || apiKey == nil {
		return model
	}
	return service.DefaultAllowedOpenAIImageModel(apiKey.Group)
}

func (g *mcpImageGateway) validateImageModel(model string) error {
	apiKey, ok := middleware2.GetAPIKeyFromContext(g.source)
	if !ok || apiKey == nil || apiKey.Group == nil {
		return mcp.NewToolError(mcp.ErrUnauthorized, "", fmt.Errorf("authenticated API key context missing"))
	}
	if apiKey.Group.Platform != service.PlatformOpenAI {
		return mcp.NewToolError(mcp.ErrModelNotAllowed, "", fmt.Errorf("image MCP currently supports OpenAI platform groups"))
	}
	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		return mcp.NewToolError(mcp.ErrPermission, service.ImageGenerationPermissionMessage(), nil)
	}
	if apiKey.Group.ModelAllowlistEnabled() && !apiKey.Group.ModelAllowlist.Allows(model) {
		return mcp.NewToolError(mcp.ErrModelNotAllowed, "", fmt.Errorf("model is not allowed by this API key group"))
	}
	return nil
}

func buildMCPImageRequest(model, prompt, size, quality, outputFormat string, n int, images []map[string]string) ([]byte, error) {
	payload := map[string]any{
		"model":           model,
		"prompt":          prompt,
		"size":            size,
		"quality":         quality,
		"output_format":   outputFormat,
		"response_format": "b64_json",
		"n":               n,
	}
	if images != nil {
		payload["images"] = images
	}
	return json.Marshal(payload)
}

func (g *mcpImageGateway) callImages(ctx context.Context, path string, body []byte, model string, imageCount int) (mcp.ImageResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return mcp.ImageResult{}, err
	}
	req.Header = g.source.Request.Header.Clone()
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	rec := httptest.NewRecorder()
	imageContext, _ := gin.CreateTestContext(rec)
	imageContext.Request = req
	for key, value := range g.source.Keys {
		imageContext.Set(key, value)
	}

	g.openAI.Images(imageContext)
	if rec.Code < 200 || rec.Code >= 300 {
		return mcp.ImageResult{}, classifyMCPImageResponse(rec.Code, rec.Body.Bytes())
	}
	body, offloaded, err := rewriteMCPImageResponse(ctx, g.imageStorageResolver, newMCPImageStorageID(), rec.Body.Bytes())
	if err != nil {
		return mcp.ImageResult{}, mcp.NewToolError(mcp.ErrUpstream, "", err)
	}
	result, err := parseOpenAIImageResult(body, model, imageCount)
	if err != nil {
		return mcp.ImageResult{}, err
	}
	if !offloaded {
		result, err = materializeMCPImageResult(ctx, result)
		if err != nil {
			return mcp.ImageResult{}, mcp.NewToolError(mcp.ErrUpstream, "", err)
		}
	}
	return result, nil
}

func rewriteMCPImageResponse(ctx context.Context, resolver service.ImageStorageResolver, requestID string, body []byte) ([]byte, bool, error) {
	if resolver == nil {
		return body, false, nil
	}
	uploader, enabled := resolver()
	if !enabled || uploader == nil {
		return body, false, nil
	}
	rewritten, err := uploader.Rewrite(ctx, requestID, json.RawMessage(body))
	if err != nil {
		return nil, false, fmt.Errorf("store generated image to object storage: %w", err)
	}
	return []byte(rewritten), true, nil
}

func newMCPImageStorageID() string {
	return "mcpreq_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func classifyMCPImageResponse(status int, body []byte) error {
	var payload struct {
		Error struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)
	errType := strings.ToLower(strings.TrimSpace(payload.Error.Type))
	errCode := strings.ToLower(strings.TrimSpace(payload.Error.Code))
	msg := strings.TrimSpace(payload.Error.Message)
	switch {
	case status == http.StatusUnauthorized:
		return mcp.NewToolError(mcp.ErrUnauthorized, "", nil)
	case status == http.StatusForbidden && (errType == "permission_error" || strings.Contains(strings.ToLower(msg), "image generation is not enabled")):
		return mcp.NewToolError(mcp.ErrPermission, service.ImageGenerationPermissionMessage(), nil)
	case status == http.StatusForbidden && (errType == "billing_error" || strings.Contains(errCode, "insufficient") || strings.Contains(strings.ToLower(msg), "insufficient balance")):
		return mcp.NewToolError(mcp.ErrInsufficient, "", nil)
	case status == http.StatusTooManyRequests:
		return mcp.NewToolError(mcp.ErrRateLimited, "", nil)
	case status == http.StatusNotFound || strings.Contains(strings.ToLower(msg), "no available compatible accounts"):
		return mcp.NewToolError(mcp.ErrNoAccount, "", nil)
	case status == http.StatusBadGateway || status >= http.StatusInternalServerError:
		return mcp.NewToolError(mcp.ErrUpstream, "", nil)
	case status == http.StatusBadRequest && (strings.Contains(errCode, "model") || strings.Contains(strings.ToLower(msg), "model")):
		return mcp.NewToolError(mcp.ErrModelNotAllowed, "", nil)
	default:
		return mcp.NewToolError(mcp.ErrInternal, "", nil)
	}
}

func parseOpenAIImageResult(body []byte, model string, count int) (mcp.ImageResult, error) {
	var response struct {
		Data []struct {
			URL           string `json:"url"`
			B64JSON       string `json:"b64_json"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return mcp.ImageResult{}, err
	}
	if len(response.Data) == 0 {
		return mcp.ImageResult{}, fmt.Errorf("image response did not include data")
	}
	item := response.Data[0]
	result := mcp.ImageResult{
		URL:           strings.TrimSpace(item.URL),
		MIMEType:      "image/png",
		Model:         model,
		ImageCount:    count,
		RevisedPrompt: item.RevisedPrompt,
	}
	if result.URL == "" && item.B64JSON != "" {
		result.URL = "data:image/png;base64," + item.B64JSON
	}
	if result.URL == "" {
		return mcp.ImageResult{}, fmt.Errorf("image response did not include a URL")
	}
	return result, nil
}

const maxMCPImageBytes int64 = 20 << 20

func materializeMCPImageResult(ctx context.Context, result mcp.ImageResult) (mcp.ImageResult, error) {
	raw := strings.TrimSpace(result.URL)
	if strings.HasPrefix(strings.ToLower(raw), "data:image/") {
		return result, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return mcp.ImageResult{}, fmt.Errorf("image response URL is invalid")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return mcp.ImageResult{}, fmt.Errorf("download image result: %w", err)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return mcp.ImageResult{}, fmt.Errorf("download image result: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcp.ImageResult{}, fmt.Errorf("download image result: unexpected HTTP status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMCPImageBytes+1))
	if err != nil || int64(len(data)) > maxMCPImageBytes {
		if err == nil {
			err = fmt.Errorf("image exceeds %d bytes", maxMCPImageBytes)
		}
		return mcp.ImageResult{}, fmt.Errorf("download image result: %w", err)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if mediaType, _, parseErr := mime.ParseMediaType(contentType); parseErr == nil {
		contentType = mediaType
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return mcp.ImageResult{}, fmt.Errorf("download image result: content type is not an image")
	}
	result.MIMEType = contentType
	result.URL = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
	return result, nil
}
