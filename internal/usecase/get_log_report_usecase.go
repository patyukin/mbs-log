package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	authpb "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

func (u *UseCase) GetLogReport(ctx context.Context, in *authpb.LogReportRequest) error {
	logs, err := u.db.GetRepo().SelectLogs(ctx, in)
	if err != nil {
		return fmt.Errorf("failed u.db.GetRepo().SelectLogs: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{"database", "schema", "table", "operation", "event_time", "data", "event_date"}
	if err = writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers to CSV: %w", err)
	}

	for _, l := range logs {
		record := []string{
			l.Database,
			l.Schema,
			l.Table,
			l.Operation,
			l.EventTime.Format(time.RFC3339),
			l.Data,
			l.EventDate.Format("2006-01-02"),
		}

		if err = writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record to CSV: %w", err)
		}
	}

	writer.Flush()
	if err = writer.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	fileUrl, err := u.s3.UploadLogReport(ctx, &buf)
	if err != nil {
		return fmt.Errorf("failed u.s3.UploadLogReport: %w", err)
	}

	err = u.rabbit.PublishAuthSignUpResultMessage(ctx, []byte(fileUrl), amqp.Table{})
	if err != nil {
		return fmt.Errorf("failed u.rabbit.PublishAuthSignUpResultMessage: %w", err)
	}

	return nil
}
