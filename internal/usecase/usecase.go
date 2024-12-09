package usecase

import (
	"bytes"
	"context"
	"github.com/patyukin/mbs-log/internal/db"
	"github.com/patyukin/mbs-pkg/pkg/kafka"
)

type MinioClient interface {
	UploadCSVBuffer(ctx context.Context, objectName string, buf *bytes.Buffer) (string, error)
}

type UseCase struct {
	db    *db.Registry
	kafka *kafka.Client
	mn    MinioClient
}

func New(db *db.Registry, kafka *kafka.Client, mn MinioClient) *UseCase {
	return &UseCase{
		db:    db,
		kafka: kafka,
		mn:    mn,
	}
}
