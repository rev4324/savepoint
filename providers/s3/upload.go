package s3

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gosimple/slug"
	"github.com/minio/minio-go/v7"
	"github.com/rev4324/savepoint/config"
	"github.com/rev4324/savepoint/utils"
)

type UploadStats struct {
	Successful int32
	Errors     []error
	TotalBytes int64
	Duration   time.Duration
}

func (u *S3Provider) Upload(ctx context.Context, game *config.OSSpecificGameConfig) error {
	gameSlug := slug.Make(game.Name)

	log.Info("Starting upload", "endpoint", u.config.Bucket.Endpoint, "game", game.Name)

	stats := u.uploadDirectory(ctx, game.SaveDir, gameSlug)

	if len(stats.Errors) > 0 {
		return fmt.Errorf("encountered multiple upload errors: %v", stats.Errors)
	}

	fmt.Println()
	log.Infof("Successful: %d files", stats.Successful)
	log.Infof("Total bytes transferred: %s", utils.ByteCountBinary(stats.TotalBytes))
	log.Infof("Took %fs", stats.Duration.Seconds())
	log.Infof("Average speed: %s/s", utils.ByteCountBinary(stats.TotalBytes/int64(stats.Duration.Seconds())))

	return nil
}

func (u *S3Provider) uploadDirectory(ctx context.Context, sourcePath string, targetKeyPrefix string) UploadStats {
	var stats UploadStats
	start := time.Now()
	paths := make(chan string, 1000)
	results := make(chan error, 1000)
	rootDirName := filepath.Base(sourcePath)

	const numWorkers = 100
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				rel, err := filepath.Rel(sourcePath, path)
				if err != nil {
					results <- fmt.Errorf("error getting relative path for %s: %w", path, err)
				}

				targetPath := filepath.ToSlash(filepath.Join(targetKeyPrefix, rootDirName, rel))

				size, err := uploadFile(ctx, u.client, u.config.Bucket.Bucket, path, targetPath)
				if err != nil {
					results <- fmt.Errorf("error uploading %s: %w", path, err)
				}
				stats.TotalBytes += size
				stats.Successful += 1
			}
		}()
	}

	go func() {
		defer close(paths)
		err := filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				select {
				case paths <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
		if err != nil {
			select {
			case results <- fmt.Errorf("walk error: %w", err):
			default:
			}
		}
	}()

	go func() {
		wg.Wait()
		stats.Duration = time.Since(start)

		close(results)
	}()

	for err := range results {
		stats.Errors = append(stats.Errors, err)
	}

	return stats
}

func uploadFile(ctx context.Context, client *minio.Client, bucket, filePath string, targetPath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("getting file info: %w", err)
	}

	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	uploadInfo, err := client.PutObject(ctx, bucket, targetPath, file, info.Size(),
		minio.PutObjectOptions{ContentType: contentType})

	log.Info("Uploaded", "key", uploadInfo.Key, "size", uploadInfo.Size, "path", info.Name())
	if err != nil {
		return 0, fmt.Errorf("putting object: %w", err)
	}

	return uploadInfo.Size, nil
}
