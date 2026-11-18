package subscription

import "errors"

var (
	ErrPlanNotFound      = errors.New("paket langganan tidak ditemukan")
	ErrAlreadySubscribed = errors.New("masih ada langganan aktif")
	ErrNotSubscribed     = errors.New("belum berlangganan")
	ErrQuotaExceeded     = errors.New("kuota meme untuk paket ini sudah habis")
	ErrPaymentFailed     = errors.New("pembayaran gagal")
	ErrNotFound          = errors.New("langganan tidak ditemukan")
)
