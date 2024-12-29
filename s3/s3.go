package s3

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rev4324/savepoint/configManager"
)

type S3Manager interface {
	UploadFile(string)
}

var client *minio.Client

func Client(config *configManager.Config) (*minio.Client, error) {
	if (client != nil) {
		return client, nil
	}

	minioClient, err := minio.New(config.Bucket.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.Bucket.AccessKey, config.Bucket.SecretKey, ""),
		Secure: true,
	})

	if err != nil {
		return nil, err
	}

	client = minioClient

	return client, nil
}
