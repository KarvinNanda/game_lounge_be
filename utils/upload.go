package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// ── Legacy upload dir (kept for backward compatibility) ───────
const UploadDir = "./uploads"

func InitUploadDir() {
	dirs := []string{
		UploadDir + "/facility_categories",
		UploadDir + "/facilities",
		UploadDir + "/room_templates",
		UploadDir + "/stores",
		UploadDir + "/staffs",
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}
}

// ── Assets dir — used by /upload endpoint ────────────────────
const AssetsDir = "./assets/img"

// Default subfolders that are pre-created on startup
var defaultAssetFolders = []string{
	"stores", "facilities", "rooms", "staffs", "facility_categories", "room_templates", "img",
}

func InitAssetsDir() {
	for _, sub := range defaultAssetFolders {
		os.MkdirAll(AssetsDir+"/"+sub, 0755)
	}
}

// allowedExtensions contains image types accepted by the upload endpoint
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".svg":  true,
}

const maxFileSize = 2 * 1024 * 1024 // 2 MB

// sniffMimeByExt memetakan ekstensi ke prefix MIME hasil sniffing yang diizinkan.
// File yang isinya tidak cocok dengan ekstensinya DITOLAK — mencegah file HTML/
// script menyamar sebagai gambar (stored XSS / drive-by download).
var sniffMimeByExt = map[string][]string{
	".jpg":  {"image/jpeg"},
	".jpeg": {"image/jpeg"},
	".png":  {"image/png"},
	".webp": {"image/webp"},
	// SVG terdeteksi sebagai text/xml atau text/plain oleh DetectContentType
	".svg": {"image/svg+xml", "text/xml", "text/plain"},
}

// svgForbidden berisi pola berbahaya yang tidak boleh ada di file SVG.
// SVG adalah XML yang bisa memuat JavaScript — vektor stored XSS klasik.
var svgForbidden = []string{"<script", "javascript:", "onload=", "onerror=", "onclick=", "<foreignobject", "<iframe", "<embed", "href=\"data:text/html"}

// validateFileContent membaca isi file dan memastikan cocok dengan ekstensinya.
// Mengembalikan byte yang sudah terbaca agar bisa ditulis ulang oleh caller.
func validateFileContent(src io.Reader, ext string) ([]byte, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("gagal membaca file: %w", err)
	}
	head = head[:n]

	detected := http.DetectContentType(head)
	allowed := false
	for _, prefix := range sniffMimeByExt[ext] {
		if strings.HasPrefix(detected, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("isi file tidak sesuai dengan tipe %s", ext)
	}

	// Inspeksi ekstra untuk SVG: baca seluruh isi (max 2MB, sudah dicek size)
	// dan tolak jika mengandung script/event handler.
	if ext == ".svg" {
		rest, err := io.ReadAll(io.LimitReader(src, maxFileSize))
		if err != nil {
			return nil, fmt.Errorf("gagal membaca file: %w", err)
		}
		full := append(head, rest...)
		lower := strings.ToLower(string(full))
		if !strings.Contains(lower, "<svg") {
			return nil, fmt.Errorf("file SVG tidak valid")
		}
		for _, bad := range svgForbidden {
			if strings.Contains(lower, bad) {
				return nil, fmt.Errorf("file SVG mengandung konten yang tidak diizinkan")
			}
		}
		return full, nil
	}

	return head, nil
}

// SaveFileToAssets saves an uploaded file to ./assets/img/{folder}/ and
// returns the public path  /assets/img/{folder}/{filename}.{ext}
func SaveFileToAssets(file *multipart.FileHeader, folder string) (string, error) {
	// ── Validate extension ───────────────────────────────────
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return "", fmt.Errorf("tipe file tidak didukung, gunakan JPG, PNG, WEBP, atau SVG")
	}

	// ── Validate size ────────────────────────────────────────
	if file.Size > maxFileSize {
		return "", fmt.Errorf("ukuran file melebihi batas 2MB")
	}

	// ── Sanitize folder — only allow simple alphanumeric names ─
	folder = sanitizeFolder(folder)
	if folder == "" {
		folder = "img"
	}

	// ── Ensure target directory exists ───────────────────────
	targetDir := AssetsDir + "/" + folder
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori: %w", err)
	}

	// ── Generate unique filename ─────────────────────────────
	filename := uuid.NewString() + ext
	savePath := filepath.Join(targetDir, filename)

	// ── Open source ──────────────────────────────────────────
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("gagal membuka file: %w", err)
	}
	defer src.Close()

	// ── Validate actual content matches extension ────────────
	head, err := validateFileContent(src, ext)
	if err != nil {
		return "", err
	}

	// ── Write to disk ─────────────────────────────────────────
	dst, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, io.MultiReader(bytes.NewReader(head), src)); err != nil {
		os.Remove(savePath) // clean up on write failure
		return "", fmt.Errorf("gagal menulis file: %w", err)
	}

	// Return public-accessible relative path
	return fmt.Sprintf("/assets/img/%s/%s", folder, filename), nil
}

// sanitizeFolder strips any path traversal characters, allowing only
// letters, numbers, hyphens, and underscores.
func sanitizeFolder(folder string) string {
	// Remove leading/trailing slashes & spaces
	folder = strings.TrimSpace(folder)
	folder = strings.Trim(folder, "/")

	// Allow only safe characters
	var safe strings.Builder
	for _, r := range folder {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			safe.WriteRune(r)
		}
	}
	return safe.String()
}

// SaveUploadedFile — legacy helper still used by old code (saves to ./uploads/)
func SaveUploadedFile(file *multipart.FileHeader, folder string) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return "", fmt.Errorf("format file tidak didukung")
	}
	if file.Size > maxFileSize {
		return "", fmt.Errorf("ukuran file melebihi 2MB")
	}

	// Sanitasi folder untuk mencegah path traversal (../../etc)
	folder = sanitizeFolder(folder)
	if folder == "" {
		folder = "img"
	}

	filename := uuid.NewString() + ext
	savePath := fmt.Sprintf("%s/%s/%s", UploadDir, folder, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	head, err := validateFileContent(src, ext)
	if err != nil {
		return "", err
	}

	dst, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, io.MultiReader(bytes.NewReader(head), src)); err != nil {
		os.Remove(savePath)
		return "", err
	}

	return fmt.Sprintf("/uploads/%s/%s", folder, filename), nil
}

// DeleteFile removes a file given its public URL path (e.g. /assets/img/stores/x.jpg)
// Hanya file di dalam /assets/ atau /uploads/ yang boleh dihapus — path dengan
// ".." atau di luar direktori tersebut ditolak (mencegah path traversal).
func DeleteFile(urlPath string) {
	if urlPath == "" {
		return
	}
	if strings.Contains(urlPath, "..") {
		return
	}
	cleaned := filepath.ToSlash(filepath.Clean("/" + urlPath))
	if !strings.HasPrefix(cleaned, "/assets/") && !strings.HasPrefix(cleaned, "/uploads/") {
		return
	}
	os.Remove("." + cleaned)
}
