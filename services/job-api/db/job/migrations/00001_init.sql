-- +goose Up
CREATE SCHEMA IF NOT EXISTS job;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION job.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

CREATE TABLE job.job_type (
  type VARCHAR(30) PRIMARY KEY
  , active BOOLEAN NOT NULL DEFAULT TRUE
);
INSERT INTO
  job.job_type (type)
VALUES
  ('HTTP_REQUEST')
;

CREATE TABLE job.job_retry_backoff_type (
  type VARCHAR(50) PRIMARY KEY
);
INSERT INTO
  job.job_retry_backoff_type (type)
VALUES
  ('FIXED')
;


CREATE TABLE job.job (
  job_id UUID PRIMARY KEY DEFAULT uuidv7()
  , owner_id UUID NOT NULL
  , enabled BOOLEAN NOT NULL DEFAULT TRUE
  , name VARCHAR(255) NOT NULL

  , type VARCHAR(30) NOT NULL
  , config JSONB NOT NULL

  , retry_max_attempts INTEGER NOT NULL DEFAULT 1
  , retry_backoff_type VARCHAR(50) NOT NULL
  , retry_initial_delay_ms BIGINT NOT NULL DEFAULT 0
  , retry_max_delay_ms BIGINT NOT NULL

  , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  , updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

  , FOREIGN KEY (type) REFERENCES job.job_type(type) ON UPDATE CASCADE
  , FOREIGN KEY (retry_backoff_type) REFERENCES job.job_retry_backoff_type(type) ON UPDATE CASCADE
);

CREATE INDEX job_owner_id_idx
ON job.job(owner_id);

CREATE TRIGGER job_updated_at
BEFORE UPDATE ON job.job
FOR EACH ROW
EXECUTE FUNCTION job.update_updated_at_column()
;

CREATE TABLE job.execution_event_status (
  status VARCHAR(30) PRIMARY KEY
  , active BOOLEAN NOT NULL DEFAULT TRUE
);
INSERT INTO
  job.execution_event_status (status)
VALUES
  ('QUEUED')
  , ('RUNNING')
  , ('SUCCEEDED')
  , ('FAILED')
  , ('TIMED_OUT')
  , ('CANCELLED')
;

CREATE TABLE job.execution_trigger (
  trigger_type VARCHAR(30) PRIMARY KEY
);
INSERT INTO
  job.execution_trigger (trigger_type)
VALUES
  ('MANUAL')
;

CREATE TABLE job.execution (
  execution_id UUID PRIMARY KEY DEFAULT uuidv7()
  , job_id UUID NOT NULL

  , trigger_type VARCHAR(30) NOT NULL
  , status VARCHAR(30) NOT NULL

  , attempt INTEGER NOT NULL DEFAULT 1
  , max_attempts INTEGER NOT NULL DEFAULT 1

  , lease_token UUID
  , lease_expires_at TIMESTAMPTZ
  , next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

  , result JSONB NOT NULL
  , error_message TEXT

  , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  , updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

  , FOREIGN KEY (job_id) REFERENCES job.job(job_id) ON DELETE CASCADE
  , FOREIGN KEY (trigger_type) REFERENCES job.execution_trigger(trigger_type) ON UPDATE CASCADE
  , FOREIGN KEY (status) REFERENCES job.execution_event_status(status) ON UPDATE CASCADE
);

CREATE INDEX job_id_idx
ON job.execution(job_id);

CREATE TRIGGER execution_updated_at
BEFORE UPDATE ON job.execution
FOR EACH ROW
EXECUTE FUNCTION job.update_updated_at_column()
;

CREATE TABLE job.execution_event (
  execution_event_id UUID PRIMARY KEY DEFAULT uuidv7()
  , execution_id UUID NOT NULL
  , status VARCHAR(30) NOT NULL
  , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

  , FOREIGN KEY (execution_id) REFERENCES job.execution(execution_id) ON DELETE CASCADE
  , FOREIGN KEY (status) REFERENCES job.execution_event_status(status) ON UPDATE CASCADE
);

CREATE INDEX execution_event_execution_id_idx
ON job.execution_event(execution_id);
CREATE INDEX execution_event_status_idx
ON job.execution_event(status);

-- +goose Down
DROP SCHEMA IF EXISTS job CASCADE;
