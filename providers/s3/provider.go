package s3

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rev4324/savepoint/config"
	"github.com/rev4324/savepoint/providers"
)

type S3Provider struct {
	config *config.Config
	client *minio.Client
}

func New(config *config.Config) (providers.Provider, error) {
	client, err := CreateClient(config)

	if err != nil {
		return nil, err
	}

	return &S3Provider{
		config: config,
		client: client,
	}, nil
}

func CreateClient(config *config.Config) (*minio.Client, error) {
	minioClient, err := minio.New(config.Bucket.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Bucket.AccessKey, config.Bucket.SecretKey, ""),
		Secure: true,
	})

	if err != nil {
		return nil, err
	}

	return minioClient, nil
}
