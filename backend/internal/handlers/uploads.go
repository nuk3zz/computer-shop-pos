package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxImageUploadBytes int64 = 5 << 20

type ImageUploadHandler struct {
	uploadDir string
}

func NewImageUploadHandler(uploadDir string) *ImageUploadHandler {
	return &ImageUploadHandler{uploadDir: uploadDir}
}

func (h *ImageUploadHandler) UploadProductImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageUploadBytes+(1<<20))

	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Choose an image to upload",
			"error":   "image_required",
		})
		return
	}

	if fileHeader.Size > maxImageUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"success": false,
			"message": "Image must be 5 MB or smaller",
			"error":   "image_too_large",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Could not read the image"})
		return
	}
	defer file.Close()

	header := make([]byte, 512)
	bytesRead, err := file.Read(header)
	if err != nil && bytesRead == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Could not inspect the image"})
		return
	}

	contentType := http.DetectContentType(header[:bytesRead])
	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
	extension, allowed := extensions[contentType]
	if !allowed {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"success": false,
			"message": "Use a JPG, PNG, WebP, or GIF image",
			"error":   "unsupported_image_type",
		})
		return
	}

	productImageDir := filepath.Join(h.uploadDir, "products")
	if err := os.MkdirAll(productImageDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Could not prepare image storage"})
		return
	}

	filename := fmt.Sprintf("%s%s", uuid.NewString(), extension)
	destination := filepath.Join(productImageDir, filename)
	if err := c.SaveUploadedFile(fileHeader, destination); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Could not save the image"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Image uploaded successfully",
		"data": gin.H{
			"url":          "/uploads/products/" + filename,
			"content_type": contentType,
			"size":         fileHeader.Size,
		},
	})
}
