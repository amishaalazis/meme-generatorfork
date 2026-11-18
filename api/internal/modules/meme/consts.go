package meme

import "time"

const maxPresignTTL = 15 * time.Minute

// maxUploadTTL: masa berlaku token URL upload gambar dasar.
const maxUploadTTL = 15 * time.Minute

const maxUploadBytes int64 = 10 << 20

const (
	defaultLimit = 20
	maxLimit     = 100
)
