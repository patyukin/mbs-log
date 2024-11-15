package usecase

import (
	"bytes"
	"context"
	"github.com/patyukin/mbs-log/internal/db"
	"github.com/patyukin/mbs-pkg/pkg/kafka"
	"github.com/patyukin/mbs-pkg/pkg/rabbitmq"
)

type S3Client interface {
	UploadLogReport(ctx context.Context, buf *bytes.Buffer) (string, error)
}

type UseCase struct {
	db     *db.Registry
	kafka  *kafka.Client
	rabbit *rabbitmq.RabbitMQ
	s3     S3Client
}

func New(db *db.Registry, kafka *kafka.Client, rabbit *rabbitmq.RabbitMQ, s3 S3Client) *UseCase {
	return &UseCase{
		db:     db,
		kafka:  kafka,
		rabbit: rabbit,
		s3:     s3,
	}
}
