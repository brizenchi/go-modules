package operations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/foundation/ossx"
	storageS3 "github.com/brizenchi/go-modules/foundation/ossx/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const MaxImageBytes int64 = 5 << 20

func (m *Module) listImages(c *gin.Context) {
	owner := userID(c)
	if owner == "" {
		return
	}
	p, ok := pagination(c)
	if !ok {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	q := db.Model(&Upload{}).Where("user_id = ?", owner)
	var total int64
	if queryFailed(c, q.Count(&total).Error) {
		return
	}
	items := []Upload{}
	if queryFailed(c, q.Order("created_at DESC,id DESC").Offset((p.page-1)*p.limit).Limit(p.limit).Find(&items).Error) {
		return
	}
	for i := range items {
		items[i].URL = "/api/v1/uploads/images/" + items[i].ID
	}
	pageResponse(c, p, items, total)
}

func sniffImage(body []byte) (string, string, error) {
	kind := http.DetectContentType(body)
	ext := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "image/gif": ".gif"}[kind]
	if ext == "" {
		return "", "", errors.New("only PNG, JPEG, WebP and GIF images are accepted")
	}
	if kind == "image/webp" {
		// Validate the RIFF container and image chunk; Content-Type supplied by
		// the client is never trusted. WebP stays a raster-only response.
		if len(body) < 20 || int64(binary.LittleEndian.Uint32(body[4:8]))+8 != int64(len(body)) {
			return "", "", errors.New("invalid WebP image")
		}
		switch string(body[12:16]) {
		case "VP8 ", "VP8L", "VP8X":
		default:
			return "", "", errors.New("invalid WebP image")
		}
		if int64(binary.LittleEndian.Uint32(body[16:20])) > int64(len(body)-20) {
			return "", "", errors.New("invalid WebP image")
		}
	} else {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
		if err != nil || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 40000000 {
			return "", "", errors.New("invalid image or image exceeds 40 megapixels")
		}
	}
	return kind, ext, nil
}

func (m *Module) storageProvider() string {
	provider := strings.TrimSpace(m.deps.Config.Uploads.Provider)
	if provider == "" {
		provider = "local"
	}
	return provider
}
func (m *Module) prepareStorage() error {
	m.storageOnce.Do(func() {
		cfg := m.deps.Config.Uploads
		if !cfg.Enabled {
			m.storageErr = errors.New("image uploads are not enabled")
			return
		}
		switch m.storageProvider() {
		case "local":
			if strings.TrimSpace(cfg.Directory) == "" {
				m.storageErr = errors.New("private upload directory is not configured")
				return
			}
			m.storageErr = os.MkdirAll(cfg.Directory, 0700)
		case "s3":
			if (strings.TrimSpace(cfg.AccessKeyID) == "") != (strings.TrimSpace(cfg.SecretAccessKey) == "") {
				m.storageErr = errors.New("both storage access key and secret key are required when using explicit credentials")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			m.bucket, m.storageErr = storageS3.New(ctx, storageS3.Config{Bucket: cfg.Bucket, Region: cfg.Region, Endpoint: cfg.Endpoint, UsePathStyle: cfg.UsePathStyle, AccessKeyID: cfg.AccessKeyID, SecretAccessKey: cfg.SecretAccessKey})
		default:
			m.storageErr = errors.New("unsupported upload storage provider")
		}
	})
	return m.storageErr
}
func (m *Module) putImage(ctx context.Context, key string, body []byte, kind string) error {
	if m.bucket != nil {
		return m.bucket.Put(ctx, key, bytes.NewReader(body), int64(len(body)), ossx.PutOptions{ContentType: kind, CacheControl: "private, no-store", ACL: ossx.ACLPrivate})
	}
	root, err := os.OpenRoot(m.deps.Config.Uploads.Directory)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(key), 0700); err != nil {
		return err
	}
	f, err := root.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(body)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		_ = root.Remove(key)
		return errors.Join(writeErr, closeErr)
	}
	return nil
}
func (m *Module) readImage(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.bucket != nil {
		return m.bucket.Get(ctx, key)
	}
	root, err := os.OpenRoot(m.deps.Config.Uploads.Directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.Open(key)
}
func (m *Module) removeImage(ctx context.Context, key string) {
	if m.bucket != nil {
		_ = m.bucket.Delete(ctx, key)
		return
	}
	root, err := os.OpenRoot(m.deps.Config.Uploads.Directory)
	if err != nil {
		return
	}
	defer root.Close()
	_ = root.Remove(key)
}

func (m *Module) uploadImage(c *gin.Context) {
	owner := userID(c)
	if owner == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	if err := m.prepareStorage(); err != nil {
		httpresp.Custom(c, 503, 503, "image storage is not configured", nil)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxImageBytes+(64<<10))
	if err := c.Request.ParseMultipartForm(MaxImageBytes + (64 << 10)); err != nil {
		httpresp.BadRequest(c, "invalid upload; maximum image size is 5 MB")
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) != 1 || len(c.Request.MultipartForm.File) != 1 {
		httpresp.BadRequest(c, "upload exactly one image using the file field")
		return
	}
	file, err := files[0].Open()
	if err != nil {
		httpresp.BadRequest(c, "unable to read upload")
		return
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, MaxImageBytes+1))
	if err != nil || len(body) == 0 || int64(len(body)) > MaxImageBytes {
		httpresp.BadRequest(c, "image must be between 1 byte and 5 MB")
		return
	}
	kind, ext, err := sniffImage(body)
	if err != nil {
		httpresp.BadRequest(c, err.Error())
		return
	}
	id := uuid.NewString()
	ownerHash := sha256.Sum256([]byte(owner))
	key := hex.EncodeToString(ownerHash[:]) + "/" + id + ext
	if err := m.putImage(c.Request.Context(), key, body, kind); err != nil {
		httpresp.InternalError(c, "image could not be stored")
		return
	}
	row := Upload{ID: id, UserID: owner, StorageKey: key, Provider: m.storageProvider(), ContentType: kind, Size: int64(len(body)), Filename: id + ext, URL: "/api/v1/uploads/images/" + id}
	if err := db.Create(&row).Error; err != nil {
		m.removeImage(c.Request.Context(), key)
		httpresp.InternalError(c, "image could not be saved")
		return
	}
	httpresp.OK(c, row)
}

func (m *Module) getImage(c *gin.Context) {
	owner := userID(c)
	if owner == "" {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpresp.NotFound(c, "image not found")
		return
	}
	var row Upload
	err = db.Where("id = ? AND user_id = ?", id.String(), owner).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		httpresp.NotFound(c, "image not found")
		return
	}
	if queryFailed(c, err) {
		return
	}
	if err := m.prepareStorage(); err != nil || row.Provider != m.storageProvider() {
		httpresp.Custom(c, 503, 503, "image storage is not configured", nil)
		return
	}
	f, err := m.readImage(c.Request.Context(), row.StorageKey)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, ossx.ErrNotFound) {
		httpresp.NotFound(c, "image not found")
		return
	}
	if queryFailed(c, err) {
		return
	}
	defer f.Close()
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", row.Filename))
	c.DataFromReader(http.StatusOK, row.Size, row.ContentType, io.LimitReader(f, row.Size), nil)
}
