-- +goose Up
CREATE TABLE auth_audit_log
(
    database   String,
    schema     String,
    table      String,
    operation  String,
    event_time DateTime64(3, 'UTC'),
    data       String,
    event_date Date,
    created_at Datetime
) ENGINE = MergeTree()
      PARTITION BY event_date
      ORDER BY (database, schema, table, event_time);

-- +goose Down
DROP TABLE IF EXISTS auth_audit_log;
