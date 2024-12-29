package upload

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"os"
	"path/filepath"
	"sync"

	"github.com/gosimple/slug"
	"github.com/minio/minio-go/v7"
	"github.com/rev4324/savepoint/configManager"
)

type Uploader interface {
	Upload(context.Context, *configManager.OSSpecificGameConfig) error
}

type GameUploader struct {
	config *configManager.Config
	client *minio.Client
}

func New(client *minio.Client, config *configManager.Config) Uploader {
	return &GameUploader{
		config: config,
		client: client,
	}
}

func (u *GameUploader) Upload(ctx context.Context, game *configManager.OSSpecificGameConfig) error {
	gameSlug := slug.Make(game.Name)

	fmt.Printf("Endpoint: %s. GameId: %s\n", u.config.Bucket.Endpoint, gameSlug)

	err := u.uploadDirectory(ctx, game.SaveDir, gameSlug)
	if err != nil {
		return err
	}

	return nil
}

func (u *GameUploader) uploadDirectoryTar(ctx context.Context, sourcePath string, targetKeyPrefix string) error {
	rootDirName := filepath.Base(sourcePath)
	key := filepath.Join(targetKeyPrefix, fmt.Sprintf("%s.tar", rootDirName))

	tempFile, err := os.CreateTemp("", slug.Make(sourcePath))
	if err != nil {
		return err
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	log.Printf("created temp file: %s\n", tempFile.Name())

	tarWriter := tar.NewWriter(tempFile)
	defer tarWriter.Close()
	defer os.Remove(tempFile.Name())

	sourceDirFs := os.DirFS(sourcePath)
	err = addDirFSIgnoreLinks(sourceDirFs, tarWriter)
	if err != nil {
		return err
	}

	log.Println("compressed data into the temp file")

	tempFileStat, err := tempFile.Stat()
	if err != nil {
		return err
	}

	tempFile.Sync()
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	info, err := u.client.PutObject(ctx, u.config.Bucket.Bucket, key, tempFile, tempFileStat.Size(), minio.PutObjectOptions{
		ContentType:           "application/x-tar",
		NumThreads:            100,
		ConcurrentStreamParts: true,
	})
	if err != nil {
		return err
	}

	log.Printf("succesfully uploaded tarball %s to bucket %s with size %d\n", info.Key, info.Bucket, info.Size)

	return nil
}

func (u *GameUploader) uploadDirectory(ctx context.Context, sourcePath string, targetKeyPrefix string) error {
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

				targetPath := filepath.Join(targetKeyPrefix, rootDirName, rel)

				err = uploadFile(ctx, u.client, u.config.Bucket.Bucket, path, targetPath)
				if err != nil {
					results <- fmt.Errorf("error uploading %s: %w", path, err)
				}
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
		close(results)
	}()

	var errs []error
	for err := range results {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("upload errors: %v", errs)
	}
	return nil
}

func uploadFile(ctx context.Context, client *minio.Client, bucket, filePath string, targetPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	uploadInfo, err := client.PutObject(ctx, bucket, targetPath, file, info.Size(),
		minio.PutObjectOptions{ContentType: contentType})

	fmt.Printf("Successfully uploaded %s\n", uploadInfo.Key)
	if err != nil {
		return fmt.Errorf("putting object: %w", err)
	}

	return nil
}

func addDirFSIgnoreLinks(fsys fs.FS, tw *tar.Writer) error {
	return fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}
		h, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		h.Name = name
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		f, err := fsys.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
