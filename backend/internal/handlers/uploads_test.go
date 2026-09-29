package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUploadProductImageAcceptsPNG(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	handler := NewImageUploadHandler(uploadDir)

	var imageBuffer bytes.Buffer
	testImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	testImage.Set(0, 0, color.RGBA{R: 30, G: 100, B: 200, A: 255})
	if err := png.Encode(&imageBuffer, testImage); err != nil {
		t.Fatalf("encode test image: %v", err)
	}

	request := multipartImageRequest(t, "thumbnail.png", imageBuffer.Bytes())
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request

	handler.UploadProductImage(context)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var body struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.URL == "" {
		t.Fatal("expected uploaded image URL")
	}

	files, err := os.ReadDir(filepath.Join(uploadDir, "products"))
	if err != nil {
		t.Fatalf("read product upload directory: %v", err)
	}
	if len(files) != 1 || filepath.Ext(files[0].Name()) != ".png" {
		t.Fatalf("expected one generated PNG file, got %v", files)
	}
}

func TestUploadProductImageRejectsNonImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewImageUploadHandler(t.TempDir())
	request := multipartImageRequest(t, "not-an-image.png", []byte("plain text disguised as an image"))
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request

	handler.UploadProductImage(context)

	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnsupportedMediaType, response.Code, response.Body.String())
	}
}

func TestUploadSupplierDocumentAcceptsPDF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	handler := NewImageUploadHandler(uploadDir)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("document", "invoice.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads/supplier-documents", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request

	handler.UploadSupplierDocument(context)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	files, err := os.ReadDir(filepath.Join(uploadDir, "supplier-documents"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || filepath.Ext(files[0].Name()) != ".pdf" {
		t.Fatalf("expected one generated PDF file, got %v", files)
	}
}

func multipartImageRequest(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads/images", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
