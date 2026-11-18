package meme

import "errors"

var (
	ErrNotFound          = errors.New("meme tidak ditemukan")
	ErrNotOwned          = errors.New("bukan pemilik meme")
	ErrInvalidFile       = errors.New("file gambar tidak valid")
	ErrUploadTooLarge    = errors.New("ukuran file melebihi batas 10 MB")
	ErrNotSubscribedGate = errors.New("butuh langganan aktif untuk membuat meme")
)
