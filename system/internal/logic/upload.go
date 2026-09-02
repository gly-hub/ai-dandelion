package logic

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	uploader_minio "github.com/gly-hub/ai-dandelion/toolbox/uploader-minio"
	"github.com/gly-hub/ai-dandelion/toolbox/uploader-minio/minio_ext"
)

const maxPresignedURLExpire = 7 * 24 * time.Hour
const maxRemoteFileBytes int64 = 64 * 1024 * 1024
const maxInlineFileBytes int = 16 * 1024 * 1024

type UploadLogic struct {
	dao      *dao.Upload
	uploader *uploader_minio.MinioUploader
}

func NewUploadLogic(dao *dao.Upload, uploader *uploader_minio.MinioUploader) *UploadLogic {
	return &UploadLogic{dao: dao, uploader: uploader}
}
func (l *UploadLogic) ready() error {
	if l.dao == nil || l.uploader == nil {
		return errors.New("file uploader is not configured")
	}
	return nil
}

// UploadRemoteFile mirrors a model-produced URL into MinIO and returns its CDN URL.
func (l *UploadLogic) UploadRemoteFile(ctx context.Context, req *systemproto.UploadRemoteFileReq) (string, string, string, error) {
	if err := l.ready(); err != nil {
		return "", "", "", err
	}
	if req == nil {
		return "", "", "", errors.New("url is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(req.GetUrl()))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", "", "", errors.New("remote url is invalid")
	}
	name := path.Base(parsed.Path)
	if strings.TrimSpace(req.GetFileName()) != "" {
		name = path.Base(strings.TrimSpace(req.GetFileName()))
	}
	if name == "." || name == "/" || name == "" {
		name = "artifact"
	}
	client := &http.Client{Timeout: 60 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", "", "", err
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", "", fmt.Errorf("download remote file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", fmt.Errorf("download remote file: %s", resp.Status)
	}
	if resp.ContentLength > maxRemoteFileBytes {
		return "", "", "", errors.New("remote file exceeds limit")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteFileBytes+1))
	if err != nil {
		return "", "", "", err
	}
	if int64(len(data)) > maxRemoteFileBytes {
		return "", "", "", errors.New("remote file exceeds limit")
	}
	contentType := strings.TrimSpace(req.GetContentType())
	if contentType == "" {
		contentType = strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	}
	cdnURL, err := l.uploader.UploadBytes(data, name, contentType)
	if err != nil {
		return "", "", "", fmt.Errorf("upload remote file: %w", err)
	}
	return cdnURL, name, contentType, nil
}

func (l *UploadLogic) UploadInlineFile(_ context.Context, req *systemproto.UploadInlineFileReq) (string, string, string, error) {
	if err := l.ready(); err != nil {
		return "", "", "", err
	}
	if req == nil || len(req.GetData()) == 0 || len(req.GetData()) > maxInlineFileBytes {
		return "", "", "", errors.New("inline file is empty or too large")
	}
	name := path.Base(strings.TrimSpace(req.GetFileName()))
	if name == "" || name == "." || name == "/" {
		name = "artifact"
	}
	contentType := strings.TrimSpace(req.GetContentType())
	url, err := l.uploader.UploadBytes(req.GetData(), name, contentType)
	return url, name, contentType, err
}
func uploadExpire(seconds int64, fallback time.Duration) (time.Duration, error) {
	if seconds == 0 {
		return fallback, nil
	}
	if seconds < 1 || seconds > int64(maxPresignedURLExpire/time.Second) {
		return 0, errors.New("expires_seconds must be between 1 second and 7 days")
	}
	return time.Duration(seconds) * time.Second, nil
}
func validMD5(value string) bool {
	if len(value) != 32 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func uploadSession(item *model.Upload, reused bool, url string) *systemproto.UploadSession {
	return &systemproto.UploadSession{Uuid: item.UUID, UploadId: item.UploadID, Mode: item.Mode, Url: url, ContentType: item.ContentType, PartSize: item.PartSize, TotalParts: int32(item.TotalParts), Md5: item.MD5, Status: item.Status, Reused: reused}
}
func (l *UploadLogic) Create(ctx context.Context, req *systemproto.CreateUploadReq) (*systemproto.UploadSession, error) {
	if err := l.ready(); err != nil {
		return nil, err
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.FileSize <= 0 || req.FileSize > minio_ext.MaxMultipartPutObjectSize {
		return nil, errors.New("file_size is illegal")
	}
	if !validMD5(strings.TrimSpace(req.Md5)) {
		return nil, errors.New("md5 is illegal")
	}
	if strings.TrimSpace(req.FileName) == "" {
		return nil, errors.New("file_name is illegal")
	}
	expires, err := uploadExpire(req.ExpiresSeconds, uploader_minio.DefaultPresignedUploadExpireTime)
	if err != nil {
		return nil, err
	}
	md5 := strings.ToLower(strings.TrimSpace(req.Md5))
	if item, findErr := l.dao.FindReusable(ctx, userID, md5); findErr == nil {
		session := uploadSession(item, item.Status == model.UploadStatusCompleted, "")
		if item.Status == model.UploadStatusCompleted {
			session.Url, err = l.previewURL(item.UUID, expires)
		} else if item.Mode == uploader_minio.UploadModePut {
			session.Url, err = l.uploader.PresignedUploadURL(item.UUID, expires)
		} else if parts, partsErr := l.uploader.ObjectParts(item.UUID, item.UploadID); partsErr == nil {
			for n := range parts {
				session.CompletedParts = append(session.CompletedParts, int32(n))
			}
			sort.Slice(session.CompletedParts, func(i, j int) bool { return session.CompletedParts[i] < session.CompletedParts[j] })
		} else {
			err = partsErr
		}
		return session, err
	}
	created, err := l.uploader.CreateUpload(req.FileSize, req.FileName, req.ContentType, expires)
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	item := &model.Upload{UUID: created.UUID, UserID: userID, UploadID: created.UploadID, MD5: md5, FileName: req.FileName, ContentType: created.ContentType, FileSize: req.FileSize, Mode: created.Mode, PartSize: created.PartSize, TotalParts: created.TotalParts, Status: model.UploadStatusPending, CreatedAt: now, UpdatedAt: now}
	if err := l.dao.Create(ctx, item); err != nil {
		return nil, err
	}
	return uploadSession(item, false, created.URL), nil
}
func (l *UploadLogic) PartURL(ctx context.Context, req *systemproto.GetUploadPartURLReq) (string, error) {
	if err := l.ready(); err != nil {
		return "", err
	}
	if req == nil || strings.TrimSpace(req.Uuid) == "" || strings.TrimSpace(req.UploadId) == "" || req.PartNumber < 1 || req.PartNumber > minio_ext.MaxPartsCount || req.FileSize <= 0 || req.FileSize > minio_ext.MinPartSize {
		return "", errors.New("multipart upload parameters are illegal")
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return "", err
	}
	item, err := l.dao.GetOwned(ctx, userID, req.Uuid)
	if err != nil || item.Status != model.UploadStatusPending || item.UploadID != req.UploadId || req.PartNumber > int32(item.TotalParts) {
		return "", errors.New("upload session is invalid")
	}
	return l.uploader.MultipartUploadUrl(req.Uuid, req.UploadId, int(req.PartNumber), req.FileSize)
}
func (l *UploadLogic) Complete(ctx context.Context, req *systemproto.CompleteUploadReq) (string, error) {
	if err := l.ready(); err != nil {
		return "", err
	}
	if req == nil || strings.TrimSpace(req.Uuid) == "" {
		return "", errors.New("upload session is invalid")
	}
	expires, err := uploadExpire(req.PreviewExpiresSeconds, uploader_minio.DefaultPresignedPreviewExpireTime)
	if err != nil {
		return "", err
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return "", err
	}
	item, err := l.dao.GetOwned(ctx, userID, req.Uuid)
	if err != nil || item.Status != model.UploadStatusPending {
		return "", errors.New("upload session is invalid")
	}
	var url string
	if item.Mode == uploader_minio.UploadModePut {
		exists, _, e := l.uploader.IsObjectExist(item.UUID)
		if e != nil {
			return "", e
		}
		if !exists {
			return "", errors.New("uploaded object does not exist")
		}
		url, err = l.previewURL(item.UUID, expires)
	} else {
		if item.UploadID != req.UploadId {
			return "", errors.New("upload session is invalid")
		}
		if _, err = l.uploader.CompleteMultipart(item.UUID, item.UploadID); err == nil {
			url, err = l.previewURL(item.UUID, expires)
		}
	}
	if err != nil {
		return "", err
	}
	return url, l.dao.MarkCompleted(ctx, item.UUID, nowUnixMicro())
}
func (l *UploadLogic) PreviewURL(ctx context.Context, req *systemproto.GetUploadURLReq) (string, error) {
	return l.objectURL(ctx, req, false)
}
func (l *UploadLogic) DownloadURL(ctx context.Context, req *systemproto.GetUploadURLReq) (string, error) {
	return l.objectURL(ctx, req, true)
}
func (l *UploadLogic) objectURL(ctx context.Context, req *systemproto.GetUploadURLReq, download bool) (string, error) {
	if err := l.ready(); err != nil {
		return "", err
	}
	if req == nil || strings.TrimSpace(req.Uuid) == "" {
		return "", errors.New("uuid is illegal")
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return "", err
	}
	item, err := l.dao.GetOwned(ctx, userID, req.Uuid)
	if err != nil || item.Status != model.UploadStatusCompleted {
		return "", errors.New("uploaded file is not completed")
	}
	fallback := uploader_minio.DefaultPresignedPreviewExpireTime
	if download {
		fallback = uploader_minio.DefaultPresignedDownloadExpireTime
	}
	expires, err := uploadExpire(req.ExpiresSeconds, fallback)
	if err != nil {
		return "", err
	}
	if download {
		return l.uploader.PresignedDownloadURL(item.UUID, expires, req.FileName)
	}
	return l.previewURL(item.UUID, expires)
}

func (l *UploadLogic) ResolveForAgent(ctx context.Context, req *systemproto.ResolveUploadForAgentReq) (*systemproto.AgentUpload, error) {
	if err := l.ready(); err != nil {
		return nil, err
	}
	if req == nil || strings.TrimSpace(req.GetUuid()) == "" {
		return nil, errors.New("uuid is illegal")
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.dao.GetOwned(ctx, userID, req.GetUuid())
	if err != nil || item.Status != model.UploadStatusCompleted {
		return nil, errors.New("uploaded file is not available")
	}
	url, err := l.previewURL(item.UUID, uploader_minio.DefaultPresignedPreviewExpireTime)
	if err != nil {
		return nil, err
	}
	return &systemproto.AgentUpload{Uuid: item.UUID, FileName: item.FileName, ContentType: item.ContentType, FileSize: item.FileSize, Url: url}, nil
}

func (l *UploadLogic) previewURL(uuid string, expires time.Duration) (string, error) {
	publicURL, err := l.uploader.PublicURL(uuid)
	if err != nil || publicURL != "" {
		return publicURL, err
	}
	return l.uploader.PresignedPreviewURL(uuid, expires)
}
