package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gosimple/slug"
	"github.com/minio/minio-go/v7"
	"github.com/rev4324/savepoint/config"
)

func (u *S3Provider) Download(ctx context.Context, game *config.OSSpecificGameConfig) error {
	keyPrefix := filepath.ToSlash(filepath.Join(
		slug.Make(game.Name),
		filepath.Base(game.SaveDir),
	))

	u.downloadInternal(ctx, keyPrefix, game.SaveDir)

	return nil
}

func (u *S3Provider) downloadInternal(ctx context.Context, keyPrefix string, savePath string) error {
	var wg sync.WaitGroup
	errch := make(chan error, 1000)
	count := 0

	objch := u.client.ListObjects(ctx, u.config.Bucket.Bucket, minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    keyPrefix,
	})

	for obj := range objch {
		wg.Add(1)
		count++
		go func() {
			defer wg.Done()
			objBody, err := u.client.GetObject(ctx, u.config.Bucket.Bucket, obj.Key, minio.GetObjectOptions{})
			if err != nil {
				errch <- err
				return
			}
			defer objBody.Close()

			relativeKey, err := filepath.Rel(keyPrefix, obj.Key)
			if err != nil {
				errch <- err
				return
			}

			targetPath := filepath.Join(savePath, relativeKey)
			targetPathDir := filepath.Dir(targetPath)

			log.Infof("Starting download of %s to %s", obj.Key, targetPath)

			// create all necessary parent dirs
			err = os.MkdirAll(targetPathDir, 0755)
			if err != nil {
				errch <- err
				return
			}

			// create the file
			newFile, err := os.Create(targetPath)
			if err != nil {
				errch <- err
				return
			}
			defer newFile.Close()

			// copy the contents from s3 to the new file
			written, err := io.Copy(newFile, objBody)
			if err != nil {
				errch <- err
				return
			}

			log.Infof("Successfully downloaded %s with size %d", targetPath, written)
			count--
		}()
	}

	go func() {
		wg.Wait()
		close(errch)
	}()

	go func() {
		prev := 0
		for {
			if prev != count {
				prev = count
				log.Infof("%d files left", count)
			}
		}
	}()

	var errs []error

	for err := range errch {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors: %v", errs)
	}

	return nil
}
