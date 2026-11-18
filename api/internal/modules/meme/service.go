package meme

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5"

	"github.com/nyxsr/meme-portal/api/internal/config"
	"github.com/nyxsr/meme-portal/api/internal/platform/token"
)

// Gate memeriksa apakah user berhak membuat meme (langganan aktif).
// Diimplementasikan oleh modul subscription lewat structural typing.
type Gate interface {
	CanCreate(ctx context.Context, userID string) error
	// ReserveMemeSlot mengamankan satu slot kuota secara atomik: cek kuota
	// dan pembuatan record meme (fn) berjalan dalam SATU transaksi berkunci
	// per user, sehingga request paralel tidak bisa menembus kuota.
	ReserveMemeSlot(ctx context.Context, userID string, fn func(tx pgx.Tx) error) error
}

type Service struct {
	repo    *Repository
	s3      *s3.Client
	bucket  string
	baseURL string // public base URL Garage web endpoint
	secret  string // HMAC secret untuk token URL upload gambar dasar
	gate    Gate
}

func NewService(repo *Repository, s3Client *s3.Client, cfg config.StorageConfig, secret string, gate Gate) *Service {
	return &Service{
		repo: repo, s3: s3Client,
		bucket: cfg.Bucket, baseURL: cfg.PublicBaseURL,
		secret: secret, gate: gate,
	}
}

// PrepareUpload membuat record meme (pending) dan mengembalikan URL upload
// gambar dasar. Berbeda dari presigned PUT langsung ke S3, upload berjalan
// lewat endpoint API ber-token sehingga ukuran file diberlakukan sungguhan
// di server — klien yang tidak mengirim field size (kontrak lama) pun tidak
// bisa menyimpan object di atas batas.
func (s *Service) PrepareUpload(ctx context.Context, userID, title, ext string, size int64, contentType, origin string) (UploadResponse, error) {
	contentType = normalizeContentType(contentType)
	// Ekstensi wajib ada — langsung dari filename atau diturunkan dari
	// contentType yang diizinkan (klien lama kadang tidak mengirim ekstensi).
	if !allowedExt(ext) && allowedContentType(contentType) {
		ext = extForContentType(contentType)
	}
	if !allowedExt(ext) {
		return UploadResponse{}, ErrInvalidFile
	}
	// Penolakan dini untuk klaim ukuran yang jelas-jelas melampaui batas;
	// ukuran sesungguhnya tetap divalidasi saat body diterima (StoreBaseImage)
	// karena klaim klien tidak bisa dipercaya.
	if size > maxUploadBytes {
		return UploadResponse{}, ErrUploadTooLarge
	}
	ext = strings.ToLower(ext)
	objectKey := fmt.Sprintf("base/%s%s", token.RandomHex(12), ext)

	// Cek kuota + insert record meme dalam satu transaksi berkunci (TKT-109):
	// request paralel untuk user yang sama terserialisasi di gerbang ini.
	var m Meme
	err := s.gate.ReserveMemeSlot(ctx, userID, func(tx pgx.Tx) error {
		created, cerr := s.repo.Create(ctx, tx, Meme{
			UserID: userID, Title: title,
			BaseImageURL: s.publicURL(objectKey),
		})
		if cerr == nil {
			m = created
		}
		return cerr
	})
	if err != nil {
		return UploadResponse{}, err
	}

	uploadContentType := contentTypeForExt(ext)
	expires := time.Now().Add(maxUploadTTL)
	return UploadResponse{
		ID:          m.ID,
		UploadURL:   s.uploadURL(origin, m.ID, expires),
		BaseURL:     m.BaseImageURL,
		ContentType: uploadContentType,
	}, nil
}

// StoreBaseImage menerima body gambar dasar dari endpoint upload ber-token:
// memvalidasi token & tipe, membatasi ukuran secara keras, lalu menyimpannya
// ke S3/Garage. Inilah titik enforcement ukuran yang tidak bisa dilewati
// klien — berbeda dari presigned PUT yang tidak berbatas tanpa Content-Length.
func (s *Service) StoreBaseImage(ctx context.Context, id, rawToken string, expires int64, contentType string, body io.Reader) error {
	if !verifyUploadToken(s.secret, id, rawToken, expires) {
		return ErrUploadToken
	}
	baseImageURL, err := s.repo.GetBaseImageURL(ctx, id)
	if err != nil {
		return err
	}
	ext := path.Ext(baseImageURL)
	if !allowedExt(ext) {
		return ErrInvalidFile
	}
	uploadContentType := contentTypeForExt(ext)
	if ct := normalizeContentType(contentType); ct != "" && ct != uploadContentType {
		return ErrInvalidFile
	}

	data, err := io.ReadAll(body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return ErrUploadTooLarge
		}
		return fmt.Errorf("meme service: baca body upload gagal: %w", err)
	}
	if int64(len(data)) > maxUploadBytes {
		return ErrUploadTooLarge
	}

	key := s.objectKeyOf(baseImageURL)
	if _, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(uploadContentType),
		ContentLength: aws.Int64(int64(len(data))),
	}); err != nil {
		return fmt.Errorf("meme service: simpan gambar dasar gagal: %w", err)
	}
	return nil
}

// uploadURL menyusun URL endpoint upload gambar dasar ber-token (penerus
// presigned PUT untuk base image).
func (s *Service) uploadURL(origin, id string, expires time.Time) string {
	return fmt.Sprintf("%s/api/v1/memes/upload/%s/file?expires=%d&token=%s",
		strings.TrimRight(origin, "/"), id, expires.Unix(),
		uploadToken(s.secret, id, expires))
}

// objectKeyOf mengembalikan object key S3 dari URL publiknya.
func (s *Service) objectKeyOf(publicURL string) string {
	if s.baseURL == "" {
		return publicURL
	}
	return strings.TrimPrefix(publicURL, strings.TrimRight(s.baseURL, "/")+"/")
}

// uploadToken: HMAC-SHA256(id|expires) — mengikat token ke meme & masa berlaku.
func uploadToken(secret, id string, expires time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s|%d", id, expires.Unix())
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyUploadToken(secret, id, raw string, expires int64) bool {
	if raw == "" || time.Now().Unix() > expires {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s|%d", id, expires)
	return hmac.Equal([]byte(raw), []byte(hex.EncodeToString(mac.Sum(nil))))
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
