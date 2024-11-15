package model

import (
	"time"
)

type DebeziumMessage struct {
	Payload DebeziumPayload `json:"payload"`
}

type DebeziumPayload struct {
	Before map[string]interface{} `json:"before"`
	After  map[string]interface{} `json:"after"`
	Source DebeziumSource         `json:"source"`
	Op     string                 `json:"op"`
	TsMs   int64                  `json:"ts_ms"`
}

type DebeziumSource struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Name      string `json:"name"`
	TsMs      int64  `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	Db        string `json:"db"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
}

type LogReport struct {
	Database  string
	Schema    string
	Table     string
	Operation string
	EventTime time.Time
	Data      string
	EventDate time.Time
}
