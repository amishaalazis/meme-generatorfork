// Package storage menyediakan klien S3 (Garage) untuk penyimpanan objek.
package storage

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	cfgpkg "github.com/nyxsr/meme-portal/api/internal/config"
)

func NewStorage(ctx context.Context, cfg cfgpkg.StorageConfig) (*s3.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: gagal memuat konfigurasi aws: %w", err)
	}

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(scheme + "://" + cfg.Endpoint)
		o.UsePathStyle = true
		o.HTTPClient = &http.Client{}
	}), nil
}

// EnsureBucketCORS memasang aturan CORS pada bucket agar browser bisa
// mengunggah langsung ke S3/Garage memakai presigned URL. Tanpa ini,
// preflight OPTIONS ditolak dan upload dari origin web akan gagal.
func EnsureBucketCORS(ctx context.Context, client *s3.Client, bucket, allowedOrigins string) error {
	var origins []string
	for _, o := range strings.Split(allowedOrigins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return nil
	}

	_, err := client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{{
				AllowedOrigins: origins,
				AllowedMethods: []string{"GET", "PUT", "HEAD"},
				AllowedHeaders: []string{"*"},
				ExposeHeaders:  []string{"ETag"},
				MaxAgeSeconds:  aws.Int32(3600),
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("storage: gagal memasang CORS bucket: %w", err)
	}
	return nil
}
