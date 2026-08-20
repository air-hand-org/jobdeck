-- +goose Up
CREATE TABLE job.outbox_execution_status (
    status VARCHAR(20) PRIMARY KEY
);

INSERT INTO job.outbox_execution_status
    (status)
VALUES
    ('REQUESTED')
    , ('PUBLISHED')
;

CREATE TABLE job.outbox_execution_requested (
    execution_id UUID PRIMARY KEY
    , status VARCHAR(20) NOT NULL DEFAULT 'REQUESTED'
    , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    , published_at TIMESTAMPTZ

    , CONSTRAINT fk_status FOREIGN KEY (status) REFERENCES job.outbox_execution_status(status)
);

CREATE INDEX outbox_execution_requested_status_created_at
ON job.outbox_execution_requested(status, created_at)
;

CREATE TABLE job.outbox_execution_completed (
    execution_id UUID PRIMARY KEY
    , requested_at TIMESTAMPTZ NOT NULL
    , published_at TIMESTAMPTZ NOT NULL
    , completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE job.outbox_execution_completed;
DROP TABLE job.outbox_execution_requested;
DROP TABLE job.outbox_execution_status;
