package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
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
		log.Fatalln(err)
	}

	for object := range mc.ListObjects(context.TODO(), bucket, minio.ListObjectsOptions{Recursive: true, WithMetadata: true}) {
		if !strings.HasSuffix(object.Key, ".mp4") {
			continue
		}
		log.Printf("found object %v", object.Key)
		smallKey := strings.Replace(object.Key, "original", "small", 1)
		smallKey = strings.Replace(smallKey, "mp4", "jpeg", 1)
		log.Printf("guessing thumbnail %s", smallKey)

		small, err := mc.StatObject(context.TODO(), bucket, smallKey, minio.StatObjectOptions{})
		if err != nil {
			log.Printf("no thumbnail found, skipping processing. err=%s", err.Error())
			continue
		}
		log.Printf("found small")
		if small.UserMetadata["Gts-Thumbnailer-Processed"] == "true" {
			log.Printf("already processed %s", small.Key)
			continue
		}
		f, err := os.CreateTemp("", "processed")
		if err != nil {
			log.Printf("could not open temp file for processing: %v", err.Error())
			continue
		}
		log.Printf("created temp file %s", f.Name())
		defer f.Close()
		defer os.Remove(f.Name())

		objdata, err := mc.GetObject(context.TODO(), bucket, object.Key, minio.GetObjectOptions{})
		if err != nil {
			log.Printf("unable to stream data", err.Error())
			continue
		}

		args := []string{
			"-i", "pipe:",
			"-vf", "thumbnail=n=10",
			"-frames:v", "1",
			"-f", "image2pipe",
			"-c:v", "mjpeg",
			"pipe:1",
		}

		cmd := exec.Command("ffmpeg", args...)
		cmd.Stdin = objdata
		cmd.Stdout = f
		err = cmd.Run()
		if err != nil {
			log.Printf("failed to create thumbnail: %s", err.Error())
			continue
		}
		mc.FPutObject(context.TODO(), bucket, smallKey, f.Name(), minio.PutObjectOptions{
			UserMetadata: map[string]string{
				"Gts-Thumbnailer-Processed": "true",
			},
		})
	}
}
