package app

import (
	"context"
	"log/slog"
	"slices"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nyxsr/meme-portal/api/internal/config"
	"github.com/nyxsr/meme-portal/api/internal/platform/cache"
	"github.com/nyxsr/meme-portal/api/internal/platform/database"
	"github.com/nyxsr/meme-portal/api/internal/platform/storage"
	"github.com/redis/go-redis/v9"
)

type dependencies struct {
	db    *pgxpool.Pool
	redis *redis.Client
	s3    *s3.Client
}

func bootstrap(ctx context.Context, cfg config.Config) (deps *dependencies, err error) {
	var cleanups []func()
	defer func() {
		if err != nil {
			for _, cleanup := range slices.Backward(cleanups) {
				cleanup()
			}
		}
	}()

	db, err := database.NewPostgres(ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	cleanups = append(cleanups, db.Close)

	redisClient, err := cache.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}
	cleanups = append(cleanups, func() { _ = redisClient.Close() })

	s3Client, err := storage.NewStorage(ctx, cfg.Storage)
	if err != nil {
		return nil, err
	}
	// Pastikan bucket mengizinkan PUT langsung dari browser (presigned URL).
	// Kegagalan di sini tidak fatal: aplikasi tetap jalan, hanya upload
	// langsung dari browser yang akan ditolak CORS.
	if err := storage.EnsureBucketCORS(ctx, s3Client, cfg.Storage.Bucket, cfg.AllowedOrigins); err != nil {
		slog.Warn("bootstrap: gagal menyetel CORS bucket", "err", err)
	}

	return &dependencies{db: db, redis: redisClient, s3: s3Client}, nil
}
