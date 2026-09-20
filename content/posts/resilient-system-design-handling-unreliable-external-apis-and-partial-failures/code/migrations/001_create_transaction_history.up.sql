CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE transaction_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  status VARCHAR(30) NOT NULL,
  next_retry TIMESTAMPTZ,
  idempotency_key TEXT NOT NULL UNIQUE,
  purchase_payload JSONB NOT NULL,
  broker_order_id TEXT,
  attempts INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHECK (status IN ('PENDING', 'COMPLETED', 'REJECTED', 'UNKNOWN', 'MANUAL_REVIEW'))
);

CREATE INDEX transaction_history_unknown_retry_idx
  ON transaction_history (next_retry)
  WHERE status = 'UNKNOWN';
