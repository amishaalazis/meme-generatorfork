package meme

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"github.com/nyxsr/meme-portal/api/internal/middleware"
	"github.com/nyxsr/meme-portal/api/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type prepareUploadDTO struct {
	Title       string `json:"title"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

// PrepareUpload: user minta URL upload untuk gambar dasarnya.
func (h *Handler) PrepareUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}
	var dto prepareUploadDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequestBody, "invalid request body")
		return
	}
	ext := path.Ext(dto.Filename)
	res, err := h.service.PrepareUpload(r.Context(), userID, dto.Title, ext, dto.Size, dto.ContentType, requestOrigin(r))
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, http.StatusCreated, CodeUploadPrepared, res, "upload disiapkan")
}

// requestOrigin menyusun base URL publik API dari request saat ini; dipakai
// untuk membangun URL upload ber-token yang bisa diikuti klien mana pun.
func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = p
	}
	return scheme + "://" + r.Host
}

// StoreBase: endpoint tujuan PUT gambar dasar (penerus presigned PUT S3).
// Ukuran body dibatasi keras di sini sehingga file di atas batas tidak pernah
// sampai ke storage — upload URL ber-token, tanpa sesi/cookie.
func (h *Handler) StoreBase(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	expires, err := strconv.ParseInt(q.Get("expires"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusForbidden, response.CodeUnauthorized, "token upload tidak valid")
		return
	}
	// MaxBytesReader memotong pembacaan di batas + 1 byte agar kelebihan
	// ukuran terdeteksi meski klien berbohong soal Content-Length.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1)
	if err := h.service.StoreBaseImage(r.Context(), r.PathValue("id"),
		q.Get("token"), expires, r.Header.Get("Content-Type"), r.Body); err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, CodeMemeUpdated, nil, "gambar dasar tersimpan")
}

type completeDTO struct {
	ResultURL string `json:"resultUrl"`
}

// ResultUpload: minta presigned PUT URL untuk gambar hasil render canvas.
func (h *Handler) ResultUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}
	res, err := h.service.ResultUpload(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, CodeUploadPrepared, res, "upload hasil disiapkan")
}

// Complete: setelah canvas merender & mengunggah hasil ke S3, klien melaporkan URL hasilnya.
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")
	var dto completeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil || dto.ResultURL == "" {
		response.Error(w, http.StatusBadRequest, response.CodeInvalidRequestBody, "resultUrl wajib diisi")
		return
	}
	m, err := h.service.Complete(r.Context(), userID, id, dto.ResultURL)
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, CodeMemeUpdated, m, "meme selesai")
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}
	limit := clampInt(parseQueryInt(r, "limit", defaultLimit), 1, maxLimit)
	offset := max(parseQueryInt(r, "offset", 0), 0)
	res, err := h.service.List(r.Context(), userID, int32(limit), int32(offset))
	if err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, CodeMemeListed, res, "daftar meme")
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "unauthorized")
		return
	}
	if err := h.service.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	response.Success(w, http.StatusOK, CodeMemeDeleted, nil, "meme dihapus")
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidFile):
		response.Error(w, http.StatusBadRequest, response.CodeInvalidImage, err.Error())
	case errors.Is(err, ErrUploadTooLarge):
		response.Error(w, http.StatusRequestEntityTooLarge, response.CodeUploadTooLarge, err.Error())
	case errors.Is(err, ErrUploadToken):
		response.Error(w, http.StatusForbidden, response.CodeUnauthorized, err.Error())
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, response.CodeMemeNotFound, err.Error())
	case errors.Is(err, ErrNotSubscribedGate):
		response.Error(w, http.StatusForbidden, response.CodeNotSubscribed, ErrNotSubscribedGate.Error())
	case errors.Is(err, ErrQuotaExceeded):
		response.Error(w, http.StatusForbidden, response.CodeQuotaExceeded, err.Error())
	default:
		slog.Error("meme handler: internal error", "err", err)
		response.Error(w, http.StatusInternalServerError, response.CodeInternalServer, "internal server error")
	}
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}
