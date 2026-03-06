// Package api 提供文件存储API接口
package api

import (
	"io"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yuhang-jieke/ai/internal/httpserver"
	"github.com/yuhang-jieke/ai/internal/storage"
)

// StorageHandler 存储处理器
type StorageHandler struct {
	storage storage.Storage
}

// NewStorageHandler 创建存储处理器
func NewStorageHandler(s storage.Storage) *StorageHandler {
	return &StorageHandler{storage: s}
}

// UploadFile 上传文件
// POST /api/storage/upload
// Content-Type: multipart/form-data
// 参数: file (文件), key (可选，指定存储路径)
func (h *StorageHandler) UploadFile(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	// 获取上传的文件
	file, header, err := ginCtx.Request.FormFile("file")
	if err != nil {
		return ctx.Error(400, "获取文件失败: "+err.Error())
	}
	defer file.Close()

	// 获取存储路径
	key := ginCtx.PostForm("key")
	if key == "" {
		key = header.Filename
	}

	// 获取内容类型
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 上传到对象存储 - 使用统一接口，不论配置是 minio/oss/rustfs
	err = h.storage.PutObject(ginCtx.Request.Context(), key, file, header.Size, contentType, nil)
	if err != nil {
		slog.Error("上传文件失败", "error", err, "key", key)
		return ctx.Error(500, "上传文件失败: "+err.Error())
	}

	slog.Info("文件上传成功", "key", key, "size", header.Size, "storage", h.storage.Type())

	return ctx.Success(map[string]interface{}{
		"key":     key,
		"size":    header.Size,
		"storage": h.storage.Type(),
		"message": "文件上传成功",
	})
}

// DownloadFile 下载文件
// GET /api/storage/download?key=xxx
func (h *StorageHandler) DownloadFile(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	key := ginCtx.Query("key")
	if key == "" {
		return ctx.Error(400, "缺少文件路径参数 key")
	}

	// 从对象存储获取文件 - 使用统一接口
	reader, info, err := h.storage.GetObject(ginCtx.Request.Context(), key)
	if err != nil {
		slog.Error("下载文件失败", "error", err, "key", key)
		return ctx.Error(404, "文件不存在或获取失败: "+err.Error())
	}
	defer reader.Close()

	// 设置响应头
	ginCtx.Header("Content-Description", "File Transfer")
	ginCtx.Header("Content-Type", info.ContentType)
	ginCtx.Header("Content-Disposition", "attachment; filename="+key)
	ginCtx.Header("Content-Transfer-Encoding", "binary")
	ginCtx.Header("Content-Length", strconv.FormatInt(info.Size, 10))

	// 写入响应体
	_, err = io.Copy(ginCtx.Writer, reader)
	if err != nil {
		slog.Error("写入响应失败", "error", err)
		return ctx.Error(500, "下载文件失败")
	}

	slog.Info("文件下载成功", "key", key, "size", info.Size, "storage", h.storage.Type())
	return nil
}

// DeleteFile 删除文件
// DELETE /api/storage/delete?key=xxx
func (h *StorageHandler) DeleteFile(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	key := ginCtx.Query("key")
	if key == "" {
		return ctx.Error(400, "缺少文件路径参数 key")
	}

	// 删除文件 - 使用统一接口
	err := h.storage.DeleteObject(ginCtx.Request.Context(), key)
	if err != nil {
		slog.Error("删除文件失败", "error", err, "key", key)
		return ctx.Error(500, "删除文件失败: "+err.Error())
	}

	slog.Info("文件删除成功", "key", key, "storage", h.storage.Type())

	return ctx.Success(map[string]string{
		"key":     key,
		"message": "文件删除成功",
		"storage": string(h.storage.Type()),
	})
}

// GetFileInfo 获取文件信息
// GET /api/storage/info?key=xxx
func (h *StorageHandler) GetFileInfo(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	key := ginCtx.Query("key")
	if key == "" {
		return ctx.Error(400, "缺少文件路径参数 key")
	}

	// 获取文件信息 - 使用统一接口
	info, err := h.storage.StatObject(ginCtx.Request.Context(), key)
	if err != nil {
		slog.Error("获取文件信息失败", "error", err, "key", key)
		return ctx.Error(404, "文件不存在: "+err.Error())
	}

	return ctx.Success(map[string]interface{}{
		"key":           info.Key,
		"size":          info.Size,
		"content_type":  info.ContentType,
		"etag":          info.ETag,
		"last_modified": info.LastModified,
		"storage":       h.storage.Type(),
	})
}

// ListFiles 列出文件
// GET /api/storage/list?prefix=xxx&limit=100
func (h *StorageHandler) ListFiles(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	prefix := ginCtx.Query("prefix")
	limit := 100
	if l := ginCtx.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}

	// 列出文件 - 使用统一接口
	result, err := h.storage.ListObjects(ginCtx.Request.Context(), prefix, "", "", limit)
	if err != nil {
		slog.Error("列出文件失败", "error", err)
		return ctx.Error(500, "列出文件失败: "+err.Error())
	}

	return ctx.Success(map[string]interface{}{
		"objects":       result.Objects,
		"prefix":        result.Prefix,
		"is_truncated":  result.IsTruncated,
		"next_marker":   result.NextMarker,
		"common_prefix": result.CommonPrefix,
		"storage":       h.storage.Type(),
	})
}

// GetPresignedURL 获取预签名URL
// GET /api/storage/presign?key=xxx&expires=3600
func (h *StorageHandler) GetPresignedURL(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	key := ginCtx.Query("key")
	if key == "" {
		return ctx.Error(400, "缺少文件路径参数 key")
	}

	expires := int64(3600) // 默认1小时
	if e := ginCtx.Query("expires"); e != "" {
		if n, err := strconv.ParseInt(e, 10, 64); err == nil {
			expires = n
		}
	}

	// 生成预签名URL - 使用统一接口
	url, err := h.storage.PresignedURL(ginCtx.Request.Context(), key, expires)
	if err != nil {
		slog.Error("生成预签名URL失败", "error", err, "key", key)
		return ctx.Error(500, "生成预签名URL失败: "+err.Error())
	}

	return ctx.Success(map[string]interface{}{
		"key":     key,
		"url":     url,
		"expires": expires,
		"storage": h.storage.Type(),
	})
}

// CopyFile 复制文件
// POST /api/storage/copy?src=xxx&dst=yyy
func (h *StorageHandler) CopyFile(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	src := ginCtx.Query("src")
	dst := ginCtx.Query("dst")
	if src == "" || dst == "" {
		return ctx.Error(400, "缺少源文件或目标文件路径")
	}

	// 复制文件 - 使用统一接口
	err := h.storage.CopyObject(ginCtx.Request.Context(), src, dst)
	if err != nil {
		slog.Error("复制文件失败", "error", err, "src", src, "dst", dst)
		return ctx.Error(500, "复制文件失败: "+err.Error())
	}

	slog.Info("文件复制成功", "src", src, "dst", dst, "storage", h.storage.Type())

	return ctx.Success(map[string]string{
		"src":     src,
		"dst":     dst,
		"message": "文件复制成功",
		"storage": string(h.storage.Type()),
	})
}

// GetStorageInfo 获取存储信息
// GET /api/storage/info
func (h *StorageHandler) GetStorageInfo(ctx httpserver.Context) error {
	return ctx.Success(map[string]interface{}{
		"type":    h.storage.Type(),
		"message": "当前使用的对象存储类型: " + string(h.storage.Type()),
	})
}

// ListBuckets 列出所有存储桶
// GET /api/storage/buckets
func (h *StorageHandler) ListBuckets(ctx httpserver.Context) error {
	ginCtx := ctx.Gin().(*gin.Context)

	// 列出存储桶 - 使用统一接口
	buckets, err := h.storage.ListBuckets(ginCtx.Request.Context())
	if err != nil {
		slog.Error("列出存储桶失败", "error", err)
		return ctx.Error(500, "列出存储桶失败: "+err.Error())
	}

	return ctx.Success(map[string]interface{}{
		"buckets": buckets,
		"storage": h.storage.Type(),
	})
}
