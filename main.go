package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	endpoint := os.Getenv("GTS_STORAGE_S3_ENDPOINT")
	bucket := os.Getenv("GTS_STORAGE_S3_BUCKET")
	accessKey := os.Getenv("GTS_STORAGE_S3_ACCESS_KEY")
	secretKey := os.Getenv("GTS_STORAGE_S3_SECRET_KEY")
	useSSL := true

	// Initialize minio client object.
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed initializing the minio client")
	}

	fname := os.TempDir()
	fname = filepath.Join(fname, "process.mp4")

	for object := range mc.ListObjects(context.TODO(), bucket, minio.ListObjectsOptions{Recursive: true, WithMetadata: true}) {
		if !strings.HasSuffix(object.Key, ".mp4") {
			continue
		}
		reqLogger := log.With().Str("key", object.Key).Logger()
		reqLogger.Trace().Msg("found object")
		smallKey := strings.Replace(object.Key, "original", "small", 1)
		smallKey = strings.Replace(smallKey, "mp4", "jpg", 1)
		reqLogger.Debug().Str("thumbnail", smallKey).Msg("computed thumbnail")

		small, err := mc.StatObject(context.TODO(), bucket, smallKey, minio.StatObjectOptions{})
		if err != nil {
			reqLogger.Warn().Err(err).Msg("no thumbnail found, skipping processing")
			continue
		}
		if small.UserMetadata["Gts-Thumbnailer-Processed"] == "true" {
			reqLogger.Debug().Msg("already processed")
			continue
		}
		f, err := os.CreateTemp("", "processed")
		if err != nil {
			reqLogger.Error().Err(err).Msg("failed to open temp file")
			continue
		}
		defer f.Close()
		defer os.Remove(f.Name())
		reqLogger = reqLogger.With().Str("tempfile", f.Name()).Logger()

		err = mc.FGetObject(context.TODO(), bucket, object.Key, fname, minio.GetObjectOptions{})
		if err != nil {
			reqLogger.Error().Err(err).Msg("unable to retreive data from object storage")
			continue
		}

		args := []string{
			"-i", fname,
			"-vf", "thumbnail=n=10",
			"-frames:v", "1",
			"-f", "image2pipe",
			"-c:v", "mjpeg",
			"pipe:1",
		}

		cmd := exec.Command("ffmpeg", args...)
		cmd.Stdout = f
		err = cmd.Run()
		if err != nil {
			reqLogger.Error().Err(err).Msg("failed to create thumbnail")
			continue
		}
		mc.FPutObject(context.TODO(), bucket, smallKey, f.Name(), minio.PutObjectOptions{
			UserMetadata: map[string]string{
				"Gts-Thumbnailer-Processed": "true",
			},
		})
		reqLogger.Info().Msg("succesfully processed")
	}
}
