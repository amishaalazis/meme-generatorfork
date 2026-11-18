package meme

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/nyxsr/meme-portal/api/internal/config"
	"github.com/nyxsr/meme-portal/api/internal/platform/token"
)

// Gate memeriksa apakah user berhak membuat meme (langganan aktif).
// Diimplementasikan oleh modul subscription lewat structural typing.
type Gate interface {
	CanCreate(ctx context.Context, userID string) error
}

type Service struct {
	repo    *Repository
	s3      *s3.Client
	bucket  string
	baseURL string // public base URL Garage web endpoint
	gate    Gate
}

func NewService(repo *Repository, s3Client *s3.Client, cfg config.StorageConfig, gate Gate) *Service {
	return &Service{repo: repo, s3: s3Client, bucket: cfg.Bucket, baseURL: cfg.PublicBaseURL, gate: gate}
}

// PrepareUpload membuat record meme (pending) dan mengembalikan presigned PUT URL
// untuk mengunggah gambar dasar langsung ke S3/Garage.
func (s *Service) PrepareUpload(ctx context.Context, userID, title, ext string, size int64, contentType string) (UploadResponse, error) {
	if err := s.gate.CanCreate(ctx, userID); err != nil {
		return UploadResponse{}, err
	}
	contentType = normalizeContentType(contentType)
	// Ekstensi wajib ada — langsung dari filename atau diturunkan dari
	// contentType yang diizinkan (klien lama kadang tidak mengirim ekstensi).
	if !allowedExt(ext) && allowedContentType(contentType) {
		ext = extForContentType(contentType)
	}
	if !allowedExt(ext) {
		return UploadResponse{}, ErrInvalidFile
	}
	// size == 0 berarti klien tidak mengirim ukuran (opsional sejak kontrak
	// awal API); jangan menolak, cukup lewati pin Content-Length di presign.
	if size > maxUploadBytes {
		return UploadResponse{}, ErrUploadTooLarge
	}
	ext = strings.ToLower(ext)
	objectKey := fmt.Sprintf("base/%s%s", token.RandomHex(12), ext)

	m, err := s.repo.Create(ctx, Meme{
		UserID: userID, Title: title,
		BaseImageURL: s.publicURL(objectKey),
	})
	if err != nil {
		return UploadResponse{}, err
	}

	uploadContentType := contentTypeForExt(ext)
	presign := s3.NewPresignClient(s.s3)
	input := &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &objectKey,
		ContentType: aws.String(uploadContentType),
	}
	// Hanya pin Content-Length bila ukurannya dikirim klien; header ini ikut
	// ditandatangani, sehingga PUT dengan panjang body berbeda akan ditolak.
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}
	req, err := presign.PresignPutObject(ctx, input, s3.WithPresignExpires(maxPresignTTL))
	if err != nil {
		return UploadResponse{}, fmt.Errorf("meme service: presign gagal: %w", err)
	}
	return UploadResponse{
		ID:          m.ID,
		UploadURL:   req.URL,
		BaseURL:     m.BaseImageURL,
		ContentType: uploadContentType,
	}, nil
}

// ResultUpload mengembalikan presigned PUT URL untuk gambar hasil render
// (canvas) milik meme <id>; complete kemudian menyimpan URL publiknya.
func (s *Service) ResultUpload(ctx context.Context, userID, id string) (UploadResponse, error) {
	if err := s.gate.CanCreate(ctx, userID); err != nil {
		return UploadResponse{}, err
	}
	if _, err := s.repo.GetByID(ctx, id, userID); err != nil {
		return UploadResponse{}, err
	}
	objectKey := fmt.Sprintf("result/%s.png", id)

	presign := s3.NewPresignClient(s.s3)
	req, err := presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &objectKey,
		ContentType: &[]string{"image/png"}[0],
	}, s3.WithPresignExpires(maxPresignTTL))
	if err != nil {
		return UploadResponse{}, fmt.Errorf("meme service: presign gagal: %w", err)
	}
	return UploadResponse{
		ID:          id,
		UploadURL:   req.URL,
		BaseURL:     s.publicURL(objectKey),
		ContentType: "image/png",
	}, nil
}

func (s *Service) Complete(ctx context.Context, userID, id, resultURL string) (Meme, error) {
	if err := s.gate.CanCreate(ctx, userID); err != nil {
		return Meme{}, err
	}
	return s.repo.SetResultURL(ctx, id, userID, resultURL)
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int32) (ListResponse, error) {
	items, total, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return ListResponse{}, err
	}
	if items == nil {
		items = []Meme{}
	}
	return ListResponse{Items: items, Total: total}, nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *Service) publicURL(key string) string {
	if s.baseURL == "" {
		return key
	}
	return strings.TrimRight(s.baseURL, "/") + "/" + path.Clean(key)
}

var imageContentTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

func allowedExt(ext string) bool {
	_, ok := imageContentTypes[strings.ToLower(ext)]
	return ok
}

func contentTypeForExt(ext string) string {
	return imageContentTypes[strings.ToLower(ext)]
}

// contentTypeExts memetakan MIME type gambar → ekstensi objek. "image/jpg"
// adalah alias tidak resmi yang sering dikirim klien; diterima sebagai JPEG.
var contentTypeExts = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func normalizeContentType(contentType string) string {
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

func allowedContentType(contentType string) bool {
	_, ok := contentTypeExts[normalizeContentType(contentType)]
	return ok
}

func extForContentType(contentType string) string {
	return contentTypeExts[normalizeContentType(contentType)]
}
