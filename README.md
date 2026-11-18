# Meme Portal

Aplikasi meme generator berbasis langganan: user berlangganan dulu, upload gambar,
lalu bisa tambah teks dan voila meme jadi!.

## Stack

- **Backend** (`api/`): Go, chi-free stdlib `net/http`, pgx/v5, sqlc, golang-migrate, JWT (golang-jwt), Redis (go-redis), S3 (aws-sdk-go-v2 → Garage)
- **Frontend** (`web/`): Next.js (App Router), TanStack Query, TanStack Table, TanStack Form, shadcn/ui, axios, bun
- **Infrastruktur lokal**: docker compose (postgres, redis, garage s3) + process-compose

## Struktur

```
api/
  cmd/server/           entrypoint
  internal/
    app/                bootstrap & wiring aplikasi
    config/             konfigurasi dari env
    db/                 sqlc generated + queries/
    middleware/         request logger, auth JWT
    modules/            auth, subscription, meme (handler/service/repository/dto/mount per modul)
    platform/           database, cache, storage, token, cookie, logger
    response/           envelope sukses/error standar
    server/             http server + route mounting
  migrations/           migrasi SQL (golang-migrate)
web/
  src/app/              halaman (App Router)
  src/components/       komponen (ui/ = shadcn)
  src/lib/api/          axios client
  src/lib/queries/      TanStack Query hooks
  src/lib/types/        tipe API
```

## Requirements

Pastikan tools berikut sudah terpasang sebelum menjalankan project:

- **Docker** dengan docker compose (untuk postgres, redis, garage s3)
- **Go** >= 1.27 (backend, `api/`)
- **Node.js** >= 20 (frontend, `web/`)
- **Bun** (package manager & runtime frontend)
- **process-compose** (orchestrator dev process, dipakai `make dev`)
- **golang-migrate** CLI (migrasi database, di-install otomatis oleh `make setup`)
- **sqlc** CLI (generate kode dari SQL, di-install otomatis oleh `make setup`)

Install manual bila perlu:

```sh
# process-compose
brew install process-compose        # macOS
# atau download binary: https://github.com/F1bonacc1/process-compose/releases

# bun
curl -fsSL https://bun.sh/install | bash

# golang-migrate & sqlc (opsional, make setup akan pasang otomatis)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Menjalankan

```sh
cp .env.example .env
make setup   # cek & pasang dependency (docker, go, bun, migrate, sqlc, air)
make dev     # infra + migrate + api (:8080) + web (:3000) via process-compose
```

Perintah lain: `make migrate-new name=x`, `make migrate-up`, `make sqlc-generate`,
`make build`, `make lint`, `make test`, `make check`.

## Catatan desain

- **Render meme di browser**: backend menyimpan gambar dasar + konfigurasi teks (top/bottom,
  ukuran, warna, stroke); browser merender via canvas dan mengunggah hasilnya ke S3 lewat API.
- **Subscription**: model plans/payments/subscriptions dengan payment gateway *mock*
  (endpoint checkout yang langsung mengonfirmasi pembayaran — siap ditukar ke provider nyata).
- **Auth**: JWT access + refresh, refresh token disimpan di Redis agar bisa direvoke.
# meme-generator
# meme-generator
