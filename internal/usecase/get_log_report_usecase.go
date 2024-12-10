package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/google/uuid"
	authpb "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
	"github.com/rs/zerolog/log"
)

func (u *UseCase) GetLogReport(ctx context.Context, in *authpb.LogReportRequest) (string, error) {
	logs, err := u.db.GetRepo().SelectLogs(ctx, in)
	if err != nil {
		return "", fmt.Errorf("failed u.db.GetRepo().SelectLogs: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{"database", "schema", "table", "operation", "event_time", "data", "event_date"}
	if err = writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write headers to CSV: %w", err)
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
			return "", fmt.Errorf("failed to write record to CSV: %w", err)
		}
	}

	writer.Flush()
	if err = writer.Error(); err != nil {
		return "", fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	now := time.Now()
	objectName := fmt.Sprintf(
		"%04d/%02d/%02d-%s.csv",
		now.Year(),
		int(now.Month()),
		now.Day(),
		uuid.New().String(),
	)

	fileUrl, err := u.mn.UploadCSVBuffer(context.Background(), objectName, &buf)
	if err != nil {
		return "", fmt.Errorf("failed u.mn.UploadCSVBuffer: %w", err)
	}

	log.Debug().Msgf("fileUrl: %v", fileUrl)

	return fileUrl, nil
}
