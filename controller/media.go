package controller

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const (
	mediaUploadField       = "file"
	multipartOverheadBytes = int64(1024 * 1024)

	maxReferenceImageBytes = int64(10 * 1024 * 1024)
	maxReferenceVideoBytes = int64(50 * 1024 * 1024)
	maxReferenceAudioBytes = int64(15 * 1024 * 1024)
)

var referenceMediaExtensions = map[string]map[string]bool{
	"image": {
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".bmp": true, ".webp": true,
	},
	"video": {
		".mp4": true, ".mov": true, ".avi": true, ".mkv": true,
		".webm": true, ".flv": true,
	},
	"audio": {
		".mp3": true, ".wav": true,
	},
}

func UploadMedia(c *gin.Context) {
	kind := strings.ToLower(strings.TrimSpace(c.Query("kind")))
	maxBytes, ok := referenceMediaMaxBytes(kind)
	if !ok {
		common.ApiErrorMsg(c, "unsupported media type")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+multipartOverheadBytes)

	fileHeader, err := c.FormFile(mediaUploadField)
	if err != nil {
		common.ApiErrorMsg(c, "missing upload file")
		return
	}
	if fileHeader.Size > maxBytes {
		common.ApiErrorMsg(c, fmt.Sprintf("file size exceeds %s", common.Bytes2Size(maxBytes)))
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !referenceMediaExtensions[kind][ext] {
		common.ApiErrorMsg(c, "unsupported file extension")
		return
	}
	if !referenceMediaContentTypeAllowed(kind, fileHeader) {
		common.ApiErrorMsg(c, "unsupported file content type")
		return
	}

	source, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer source.Close()

	name, err := randomMediaFilename(ext)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userID := c.GetInt("id")
	relativeDir := filepath.Join(kind, fmt.Sprintf("%d", userID))
	targetDir := filepath.Join(*common.MediaUploadDir, relativeDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		common.ApiError(c, err)
		return
	}

	targetPath := filepath.Join(targetDir, name)
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	written, copyErr := io.Copy(target, io.LimitReader(source, maxBytes+1))
	closeErr := target.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		common.ApiError(c, copyErr)
		return
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		common.ApiError(c, closeErr)
		return
	}
	if written > maxBytes {
		_ = os.Remove(targetPath)
		common.ApiErrorMsg(c, fmt.Sprintf("file size exceeds %s", common.Bytes2Size(maxBytes)))
		return
	}

	publicPath := "/uploads/media/" + strings.ReplaceAll(filepath.ToSlash(filepath.Join(relativeDir, name)), "\\", "/")
	common.ApiSuccess(c, gin.H{
		"url":  requestBaseURL(c) + publicPath,
		"path": publicPath,
	})
}

func referenceMediaMaxBytes(kind string) (int64, bool) {
	switch kind {
	case "image":
		return maxReferenceImageBytes, true
	case "video":
		return maxReferenceVideoBytes, true
	case "audio":
		return maxReferenceAudioBytes, true
	default:
		return 0, false
	}
}

func referenceMediaContentTypeAllowed(kind string, fileHeader *multipart.FileHeader) bool {
	contentType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if contentType == "" || contentType == "application/octet-stream" {
		return true
	}
	if strings.HasPrefix(contentType, kind+"/") {
		return true
	}
	return kind == "video" && contentType == "application/x-matroska"
}

func randomMediaFilename(ext string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]) + ext, nil
}

func requestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := c.Request.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = strings.ToLower(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]))
	}
	host := c.Request.Host
	if forwardedHost := c.Request.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}
	return scheme + "://" + host
}
