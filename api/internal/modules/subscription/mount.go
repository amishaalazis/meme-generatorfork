package subscription

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewModule menyusun modul subscription dari pool database.
func NewModule(pool *pgxpool.Pool) *Handler {
	repo := NewPostgresRepository(pool)
	return NewHandler(NewService(repo, MockGateway{}))
}

func RegisterRoutes(mux *http.ServeMux, h *Handler, authMW func(http.Handler) http.Handler) {
	// Daftar paket publik agar bisa dilihat sebelum login/berlangganan.
	mux.HandleFunc("GET /api/v1/subscription/plans", h.ListPlans)
	// /status adalah data per-user: no-store + Vary: Cookie,Authorization
	// mencegah proxy WiFi kantor (atau CDN) menyajikan status satu user ke user lain.
	mux.Handle("GET /api/v1/subscription/status", authMW(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			hdr := w.Header()
			hdr.Set("Cache-Control", "no-store, no-cache, must-revalidate")
			hdr.Set("Pragma", "no-cache")
			hdr.Set("Vary", "Cookie, Authorization")
			hdr.Set("Surrogate-Control", "no-store")
			h.Status(w, r)
		})))
	mux.Handle("POST /api/v1/subscription/checkout", authMW(http.HandlerFunc(h.Checkout)))
	// /confirm adalah callback payment gateway: dipanggil server-to-server
	// tanpa sesi/cookie user. Reference transaksi berfungsi sebagai rahasia
	// bearer yang tidak bisa ditebak; jangan pasang authMW di sini.
	mux.HandleFunc("POST /api/v1/subscription/payments/{reference}/confirm", h.Confirm)
}
