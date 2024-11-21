package db

import (
	"context"
	"fmt"
	"github.com/patyukin/mbs-log/internal/model"
	authpb "github.com/patyukin/mbs-pkg/pkg/proto/logger_v1"
	"github.com/rs/zerolog/log"
	"time"
)

type Repository struct {
	db QueryExecutor
}

func (r *Repository) InsertIntoAuditLog(ctx context.Context, payload model.DebeziumPayload, operation string, eventTime time.Time, eventDate string, jsonData []byte) error {
	query := fmt.Sprintf("INSERT INTO %s_audit_log (database, schema, table, operation, event_time, data, event_date) VALUES (?, ?, ?, ?, ?, ?, ?)", payload.Source.Db)

	_, err := r.db.ExecContext(
		ctx,
		query,
		payload.Source.Db,
		payload.Source.Schema,
		payload.Source.Table,
		operation,
		eventTime,
		string(jsonData),
		eventDate,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into auth_audit_log: %w", err)
	}

	return nil
}

func (r *Repository) SelectLogs(_ context.Context, in *authpb.LogReportRequest) ([]model.LogReport, error) {
	start := in.StartTime
	end := in.EndTime

	query := fmt.Sprintf(`
SELECT 
	database, schema, "table", operation, event_time, data, event_date 
FROM %s_audit_log 
WHERE event_date BETWEEN ? AND ?
ORDER BY event_date
`, in.ServiceName)

	log.Debug().Msgf("query: %v", query)

	rows, err := r.db.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed r.db.QueryContext: %w", err)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed rows.Err(): %w", err)
	}

	var logs []model.LogReport
	for rows.Next() {
		var l model.LogReport
		if err = rows.Scan(&l.Database, &l.Schema, &l.Table, &l.Operation, &l.EventTime, &l.Data, &l.EventDate); err != nil {
			return nil, fmt.Errorf("failed rows.Scan: %w", err)
		}

		logs = append(logs, l)
	}

	return logs, nil
}
