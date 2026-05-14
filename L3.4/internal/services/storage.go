package services

import (
	"L3.4/internal/config"
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"os"
	"path/filepath"
	"time"
)

// StorageService сервис хранилища
type StorageService struct {
	client *minio.Client
	bucket string
	temp   string
}

// NewStorageService создать
func NewStorageService(cfg *config.Config) *StorageService {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Println(err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.MinIOBucket)
	if err != nil {
		log.Println(err)
	}

	if !exists {
		err = client.MakeBucket(ctx, cfg.MinIOBucket, minio.MakeBucketOptions{})
		if err != nil {
			log.Println(err)
		}
	}

	os.MkdirAll(cfg.TempDir, 0755)

	return &StorageService{
		client: client,
		bucket: cfg.MinIOBucket,
		temp:   cfg.TempDir,
	}
}

// SaveTemp сохраняет файл во временную директорию
func (s *StorageService) SaveTemp(data []byte, fileName string) (string, error) {
	tempFIle := filepath.Join(s.temp, fileName)
	err := os.WriteFile(tempFIle, data, 0755)
	return tempFIle, err
}

// GetTemp читает файл из временной директории
func (s *StorageService) GetTemp(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// DeleteTemp удаляет временный файл
func (s *StorageService) DeleteTemp(path string) error {
	return os.Remove(path)
}

// Upload загружает файл в постоянное хранилище
func (s *StorageService) Upload(ctx context.Context, path string, data []byte, contentType string) (string, error) {
	reader := bytes.NewReader(data)
	objectName := fmt.Sprintf("image_processor/files/%s", path)

	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}

	// Возвращаем URL для доступа к файлу
	return objectName, nil
}

// GetDownloadURL генерирует временную ссылку для скачивания
func (s *StorageService) GetDownloadURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, path, expires, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// Delete удаляет из хранилища
func (s *StorageService) Delete(ctx context.Context, path string) error {
	err := s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("не удалось удалить из хранилища %v", err)
	}
	return nil
}
