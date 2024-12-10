package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patyukin/mbs-log/internal/model"
	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

func (u *UseCase) LogProcess(ctx context.Context, record *kgo.Record) error {
	var debeziumMsg model.DebeziumMessage

	log.Debug().Msgf("Received record: %v", string(record.Value))

	if err := json.Unmarshal(record.Value, &debeziumMsg); err != nil {
		return fmt.Errorf("failed to unmarshal debezium message: %w", err)
	}

	payload := debeziumMsg.Payload
	var data map[string]interface{}
	var operation string

	switch payload.Op {
	case "c", "r":
		data = payload.After
		operation = "insert"
	case "u":
		data = payload.After
		operation = "update"
	case "d":
		data = payload.Before
		operation = "delete"
	default:
		return fmt.Errorf("unknown operation: %s", payload.Op)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	eventTime := time.UnixMilli(payload.TsMs)
	eventDate := eventTime.Format(time.DateTime)

	err = u.db.GetRepo().InsertIntoAuditLog(ctx, payload, operation, eventTime, eventDate, jsonData)
	if err != nil {
		return fmt.Errorf("failed to insert into auth_audit_log: %w", err)
	}

	return nil
}
