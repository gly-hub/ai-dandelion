package uploader_minio

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/toolbox/uploader-minio/minio_ext"
	"github.com/minio/minio-go"
	miniov6 "github.com/minio/minio-go/v6"
	gouuid "github.com/satori/go.uuid"
)

type MinioUploader struct {
	minioClient    *minio.Client
	coreClient     *miniov6.Core
	minioClientExt *minio_ext.Client
	minioBucket    string
	basePath       string
	location       string
	domain         string
}

// UploadBytes stores a backend-produced artifact directly in the configured bucket.
func (u *MinioUploader) UploadBytes(data []byte, fileName, contentType string) (string, error) {
	if len(data) == 0 {
		return "", errors.New("file is empty")
	}
	if strings.TrimSpace(fileName) == "" {
		fileName = "file"
	}
	uuid, err := u.newObjectUUID(fileName)
	if err != nil {
		return "", err
	}
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = u.getContentType(path.Ext(fileName))
	}
	if _, err = u.minioClient.PutObject(u.minioBucket, objectName, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType, CacheControl: "max-age=31536000"}); err != nil {
		return "", err
	}
	if publicURL, err := u.PublicURL(uuid); err == nil && publicURL != "" {
		return publicURL, nil
	}
	return u.PresignedPreviewURL(uuid, DefaultPresignedPreviewExpireTime)
}

type ComplPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"eTag"`
}

type CompleteParts struct {
	Data []ComplPart `json:"completedParts"`
}

// UploadSession describes the client-side upload strategy selected by CreateUpload.
// For put uploads, URL is the signed PUT endpoint. For multipart uploads, callers
// use MultipartUploadUrl for each part and CompleteMultipart after all parts finish.
type UploadSession struct {
	UUID        string `json:"uuid"`
	UploadID    string `json:"uploadId,omitempty"`
	Mode        string `json:"mode"`
	URL         string `json:"url,omitempty"`
	ContentType string `json:"contentType"`
	PartSize    int64  `json:"partSize,omitempty"`
	TotalParts  int    `json:"totalParts,omitempty"`
}

// completedParts is a collection of parts sortable by their part numbers.
// used for sorting the uploaded parts before completing the multipart request.
type completedParts []miniov6.CompletePart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return a[i].PartNumber < a[j].PartNumber }

// CompleteMultipartUpload container for completing multipart upload.
type CompleteMultipartUpload struct {
	XMLName xml.Name               `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUpload" json:"-"`
	Parts   []miniov6.CompletePart `xml:"Part"`
}

func NewUploader(config MinioConfig) (uploader *MinioUploader, err error) {
	aliasedURL := config.Address
	accessKeyID := config.AccessKeyId
	secretAccessKey := config.SecretAccessKey
	secure := config.Secure

	uploader = new(MinioUploader)
	uploader.minioClient, err = minio.New(aliasedURL, accessKeyID, secretAccessKey, secure)
	if nil != err {
		return
	}

	uploader.coreClient, err = miniov6.NewCore(aliasedURL, accessKeyID, secretAccessKey, secure)
	if nil != err {
		return
	}

	uploader.minioClientExt, err = minio_ext.New(aliasedURL, accessKeyID, secretAccessKey, secure)
	if nil != err {
		return
	}

	uploader.basePath = config.BasePath
	uploader.location = config.Location
	uploader.minioBucket = config.Bucket
	uploader.domain = config.Domain

	return
}

// NewMultipart 创建上传
func (u *MinioUploader) NewMultipart(totalChunkCounts int, fileSize int64, fileName, contentType string) (uuid, uploadID string, err error) {
	if totalChunkCounts > minio_ext.MaxPartsCount || totalChunkCounts <= 0 {
		err = errors.New(IllegalTotalChunkCounts)
		return
	}

	if fileSize > minio_ext.MaxMultipartPutObjectSize || fileSize <= 0 {
		err = errors.New(IllegalSize)
		return
	}

	if fileName == "" {
		err = errors.New(IllegalFileName)
		return
	}

	uuid, err = u.newObjectUUID(fileName)
	if err != nil {
		return
	}
	contentType = u.getContentType(path.Ext(fileName))

	uploadID, err = u.newMultiPartUpload(uuid, contentType)
	if err != nil {
		return
	}

	return
}

// CreateUpload selects a simple PUT upload for files up to MultipartThreshold,
// and a multipart upload for larger files.
func (u *MinioUploader) CreateUpload(fileSize int64, fileName, contentType string, expires time.Duration) (UploadSession, error) {
	if fileSize <= 0 || fileSize > minio_ext.MaxMultipartPutObjectSize {
		return UploadSession{}, errors.New(IllegalSize)
	}
	if strings.TrimSpace(fileName) == "" {
		return UploadSession{}, errors.New(IllegalFileName)
	}
	if expires <= 0 {
		expires = DefaultPresignedUploadExpireTime
	}
	contentType = u.getContentType(path.Ext(fileName))

	uuid, err := u.newObjectUUID(fileName)
	if err != nil {
		return UploadSession{}, err
	}
	if fileSize <= minio_ext.MinPartSize {
		uploadURL, err := u.PresignedUploadURL(uuid, expires)
		if err != nil {
			return UploadSession{}, err
		}
		return UploadSession{
			UUID: uuid, Mode: UploadModePut, URL: uploadURL, ContentType: contentType,
		}, nil
	}

	totalParts := int((fileSize + minio_ext.MinPartSize - 1) / minio_ext.MinPartSize)
	uploadID, err := u.newMultiPartUpload(uuid, contentType)
	if err != nil {
		return UploadSession{}, err
	}
	return UploadSession{
		UUID: uuid, UploadID: uploadID, Mode: UploadModeMultipart, ContentType: contentType,
		PartSize: minio_ext.MinPartSize, TotalParts: totalParts,
	}, nil
}

func (u *MinioUploader) newObjectUUID(fileName string) (string, error) {
	if strings.TrimSpace(fileName) == "" {
		return "", errors.New(IllegalFileName)
	}
	return gouuid.NewV4().String() + path.Ext(fileName), nil
}

func (u *MinioUploader) getContentType(postfix string) (contentType string) {
	postfix = strings.ToLower(postfix)
	switch postfix {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".bmp":
		contentType = "image/bmp"
	case ".webp":
		contentType = "image/webp"
	case ".mp4":
		contentType = "video/mp4"
	case ".mp3":
		contentType = "audio/mp3"
	case ".ogg":
		contentType = "audio/ogg"
	case ".wav":
		contentType = "audio/wav"
	case ".flac":
		contentType = "audio/flac"
	case ".aac":
		contentType = "audio/aac"
	case ".m4a":
		contentType = "audio/m4a"
	case ".m4v":
		contentType = "video/m4v"
	case ".mov":
		contentType = "video/quicktime"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".wmv":
		contentType = "video/x-ms-wmv"
	case ".flv":
		contentType = "video/x-flv"
	case ".mkv":
		contentType = "video/x-matroska"
	case ".mts", ".m2ts":
		contentType = "video/mp2t"
	case ".3gp":
		contentType = "video/3gpp"
	case ".3g2":
		contentType = "video/3gpp2"
	case ".webm":
		contentType = "video/webm"
	case ".swf":
		contentType = "application/x-shockwave-flash"
	case ".pdf":
		contentType = "application/pdf"
	case ".doc":
		contentType = "application/msword"
	case ".docx":
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		contentType = "application/vnd.ms-excel"
	case ".xlsx":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		contentType = "application/vnd.ms-powerpoint"
	case ".pptx":
		contentType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".txt":
		contentType = "text/plain"
	case ".md", ".mdx":
		contentType = "text/markdown"
	case ".go":
		contentType = "text/x-go"
	case ".js", ".mjs", ".cjs", ".jsx":
		contentType = "text/javascript"
	case ".ts", ".tsx":
		contentType = "text/typescript"
	case ".py":
		contentType = "text/x-python"
	case ".java", ".c", ".cc", ".cpp", ".h", ".hpp", ".cs", ".rs", ".php", ".rb", ".sh", ".sql", ".css", ".vue", ".svelte":
		contentType = "text/plain"
	case ".csv":
		contentType = "text/csv"
	case ".json":
		contentType = "application/json"
	case ".yaml", ".yml":
		contentType = "application/x-yaml"
	case ".toml", ".ini", ".env":
		contentType = "text/plain"
	case ".html":
		contentType = "text/html"
	case ".xml":
		contentType = "text/xml"
	case ".zip":
		contentType = "application/zip"
	case ".rar":
		contentType = "application/x-rar-compressed"
	case ".7z":
		contentType = "application/x-7z-compressed"
	case ".gz":
		contentType = "application/gzip"
	case ".tar":
		contentType = "application/x-tar"
	case ".bz2":
		contentType = "application/x-bzip2"
	case ".apk":
		contentType = "application/vnd.android.package-archive"
	case ".ipa":
		contentType = "application/vnd.iphone"
	case ".exe":
		contentType = "application/x-msdownload"
	default:
		contentType = "application/octet-stream"
	}
	return
}

func (u *MinioUploader) newMultiPartUpload(uuid string, contentType string) (string, error) {
	bucketName := u.minioBucket
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}

	return u.coreClient.NewMultipartUpload(bucketName, objectName, miniov6.PutObjectOptions{ContentType: contentType, CacheControl: "max-age=31536000"})
}

// PresignedUploadURL returns a signed PUT URL for a previously allocated UUID.
func (u *MinioUploader) PresignedUploadURL(uuid string, expires time.Duration) (string, error) {
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}
	if expires <= 0 {
		expires = DefaultPresignedUploadExpireTime
	}
	result, err := u.minioClient.PresignedPutObject(u.minioBucket, objectName, expires)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// MultipartUploadUrl 获取上传地址
func (u *MinioUploader) MultipartUploadUrl(uuid, uploadID string, partNumber int, fileSize int64) (url string, err error) {
	if fileSize > minio_ext.MinPartSize {
		err = errors.New(IllegalSize)
		return
	}
	url, err = u.genMultiPartSignedUrl(uuid, uploadID, partNumber, fileSize)
	return
}

func (u *MinioUploader) genMultiPartSignedUrl(uuid string, uploadId string, partNumber int, partSize int64) (string, error) {
	bucketName := u.minioBucket
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}

	return u.minioClientExt.GenUploadPartSignedUrl(uploadId, bucketName, objectName, partNumber, partSize, PreassignedUploadPartUrlExpireTime, u.location)
}

// CompleteMultipart 合并上传文件
func (u *MinioUploader) CompleteMultipart(uuid, uploadID string, previewExpires ...time.Duration) (url string, err error) {
	_, err = u.completeMultiPartUpload(uuid, uploadID)
	if err != nil {
		return
	}
	expires := DefaultPresignedPreviewExpireTime
	if len(previewExpires) > 0 {
		expires = previewExpires[0]
	}
	return u.PresignedPreviewURL(uuid, expires)
}

func (u *MinioUploader) completeMultiPartUpload(uuid string, uploadID string) (string, error) {
	bucketName := u.minioBucket
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}

	partInfos, err := u.minioClientExt.ListObjectParts(bucketName, objectName, uploadID)
	if err != nil {
		return "", err
	}

	var completeMultipartUpload CompleteMultipartUpload
	for _, partInfo := range partInfos {
		completeMultipartUpload.Parts = append(completeMultipartUpload.Parts, miniov6.CompletePart{
			PartNumber: partInfo.PartNumber,
			ETag:       partInfo.ETag,
		})
	}

	// Sort all completed parts.
	sort.Sort(completedParts(completeMultipartUpload.Parts))

	return u.coreClient.CompleteMultipartUpload(bucketName, objectName, uploadID, completeMultipartUpload.Parts)
}

// IsObjectExist 校验文件是否存在
func (u *MinioUploader) IsObjectExist(uuid string) (isExist bool, url string, err error) {
	bucketName := u.minioBucket
	objectName, err := u.objectName(uuid)
	if err != nil {
		return false, "", err
	}
	doneCh := make(chan struct{})
	defer close(doneCh)

	objectCh := u.minioClient.ListObjects(bucketName, objectName, false, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			return isExist, "", object.Err
		}
		isExist = true
		url, err = u.PresignedPreviewURL(uuid, DefaultPresignedPreviewExpireTime)
		break
	}
	return
}

func (u *MinioUploader) ObjectParts(uuid, uploadID string) (partsInfo map[int]minio_ext.ObjectPart, err error) {
	objectName, err := u.objectName(uuid)
	if err != nil {
		return nil, err
	}
	return u.minioClientExt.ListObjectParts(u.minioBucket, objectName, uploadID)
}

// ReadObject returns up to maxBytes from a completed object. It is intended for
// server-side agent analysis and always rejects objects larger than the limit.
func (u *MinioUploader) ReadObject(uuid string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errors.New("maxBytes is illegal")
	}
	objectName, err := u.objectName(uuid)
	if err != nil {
		return nil, err
	}
	object, err := u.minioClient.GetObjectWithContext(context.Background(), u.minioBucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	data, err := io.ReadAll(io.LimitReader(object, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("object exceeds analysis limit of %d bytes", maxBytes)
	}
	return data, nil
}

// PublicURL returns the stable object URL when a public delivery domain is configured.
// The domain may include a scheme; otherwise HTTPS is used by default.
func (u *MinioUploader) PublicURL(uuid string) (string, error) {
	if strings.TrimSpace(u.domain) == "" {
		return "", nil
	}
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}
	domain := strings.TrimRight(strings.TrimSpace(u.domain), "/")
	if !strings.Contains(domain, "://") {
		domain = "https://" + domain
	}
	publicURL, err := url.Parse(domain)
	if err != nil || publicURL.Host == "" {
		return "", errors.New("public domain is illegal")
	}
	publicURL.Path = path.Join(publicURL.Path, u.minioBucket, objectName)
	return publicURL.String(), nil
}

// PresignedPreviewURL returns a signed inline GET URL for an object.
func (u *MinioUploader) PresignedPreviewURL(uuid string, expires time.Duration) (string, error) {
	return u.presignedGetURL(uuid, expires, "inline", "")
}

// PresignedDownloadURL returns a signed attachment GET URL for an object.
func (u *MinioUploader) PresignedDownloadURL(uuid string, expires time.Duration, fileName string) (string, error) {
	if strings.TrimSpace(fileName) == "" {
		fileName = uuid
	}
	return u.presignedGetURL(uuid, expires, "attachment", fileName)
}

func (u *MinioUploader) presignedGetURL(uuid string, expires time.Duration, disposition, fileName string) (string, error) {
	objectName, err := u.objectName(uuid)
	if err != nil {
		return "", err
	}
	if expires <= 0 {
		expires = DefaultPresignedPreviewExpireTime
	}
	params := make(url.Values)
	params.Set("response-content-disposition", disposition)
	if fileName != "" {
		fileName = strings.ReplaceAll(strings.ReplaceAll(path.Base(fileName), "\"", ""), "\r", "")
		fileName = strings.ReplaceAll(fileName, "\n", "")
		params.Set("response-content-disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, fileName))
	}
	result, err := u.minioClient.PresignedGetObject(u.minioBucket, objectName, expires, params)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func (u *MinioUploader) objectName(uuid string) (string, error) {
	if len(strings.TrimSpace(uuid)) < 2 {
		return "", errors.New("uuid is illegal")
	}
	return strings.TrimPrefix(path.Join(u.basePath, uuid[0:1], uuid[1:2], uuid), "/"), nil
}
