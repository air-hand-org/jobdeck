-- +goose Up
CREATE TABLE job.job_revision (
  revision_id UUID PRIMARY KEY DEFAULT uuidv7()
  , job_id UUID NOT NULL
  , revision_number INTEGER NOT NULL

  , type VARCHAR(30) NOT NULL
  , config JSONB NOT NULL

  , retry_max_attempts INTEGER NOT NULL DEFAULT 1
  , retry_backoff_type VARCHAR(50) NOT NULL
  , retry_initial_delay_ms BIGINT NOT NULL DEFAULT 0
  , retry_max_delay_ms BIGINT NOT NULL

  , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

  , CONSTRAINT fk_job_id FOREIGN KEY (job_id) REFERENCES job.job(job_id) ON DELETE CASCADE
  , CONSTRAINT uq_job_id_rev_number UNIQUE (job_id, revision_number)
  , CONSTRAINT fk_job_type FOREIGN KEY (type) REFERENCES job.job_type(type) ON UPDATE CASCADE
  , CONSTRAINT fk_job_retry_backoff_type FOREIGN KEY (retry_backoff_type) REFERENCES job.job_retry_backoff_type(type) ON UPDATE CASCADE
  , CONSTRAINT uq_job_id_rev_id UNIQUE (job_id, revision_id)
);

CREATE INDEX job_revision_job_id_created_at_idx
ON job.job_revision(job_id, created_at DESC)
;

CREATE TABLE job.job_current_revision (
    job_id UUID PRIMARY KEY
    , revision_id UUID NOT NULL

    , CONSTRAINT uq_rev_id UNIQUE (revision_id)
    , CONSTRAINT fk_job_id FOREIGN KEY (job_id) REFERENCES job.job(job_id) ON DELETE CASCADE
    , CONSTRAINT fk_job_id_rev_id FOREIGN KEY (job_id, revision_id) REFERENCES job.job_revision(job_id, revision_id) ON DELETE NO ACTION
);

INSERT INTO job.job_revision (
  job_id
  , revision_number
  , type
  , config
  , retry_max_attempts
  , retry_backoff_type
  , retry_initial_delay_ms
  , retry_max_delay_ms
)
SELECT
  job_id
  , 1
  , type
  , config
  , retry_max_attempts
  , retry_backoff_type
  , retry_initial_delay_ms
  , retry_max_delay_ms
FROM job.job
;

INSERT INTO job.job_current_revision (
  job_id
  , revision_id
)
SELECT
  job_id
  , revision_id
FROM job.job_revision
;

ALTER TABLE job.execution
  ADD COLUMN job_revision_id UUID NOT NULL
  , ADD CONSTRAINT fk_job_id_rev_id FOREIGN KEY (job_id, job_revision_id) REFERENCES job.job_revision(job_id, revision_id) ON DELETE NO ACTION
  , DROP COLUMN max_attempts
  , ALTER COLUMN attempt SET DEFAULT 0
  , ALTER COLUMN result DROP NOT NULL
;

ALTER TABLE job.job
  DROP COLUMN type
  , DROP COLUMN config
  , DROP COLUMN retry_max_attempts
  , DROP COLUMN retry_backoff_type
  , DROP COLUMN retry_initial_delay_ms
  , DROP COLUMN retry_max_delay_ms
;

CREATE INDEX job_created_at_job_id
ON job.job(created_at, job_id)
;

CREATE VIEW job.job_detail AS
SELECT
  j.job_id
  , j.owner_id
  , j.enabled
  , j.name
  , j.created_at
  , j.updated_at
  , r.type
  , r.config
  , r.retry_max_attempts
  , r.retry_backoff_type
  , r.retry_initial_delay_ms
  , r.retry_max_delay_ms
FROM job.job AS j
INNER JOIN job.job_current_revision AS cr ON j.job_id = cr.job_id
INNER JOIN job.job_revision AS r ON cr.job_id = r.job_id AND cr.revision_id = r.revision_id
;

CREATE VIEW job.execution_detail AS
SELECT
  e.execution_id
  , e.job_id
  , j.owner_id AS job_owner_id
  , e.job_revision_id
  , e.trigger_type
  , e.status AS current_status
  , e.attempt
  , e.lease_token
  , e.lease_expires_at
  , e.next_run_at
  , e.result
  , e.error_message
  , e.created_at
  , e.updated_at

  , r.type AS job_type
  , r.config AS job_config
  , r.retry_max_attempts AS job_retry_max_attempts
  , r.retry_backoff_type AS job_retry_backoff_type
  , r.retry_initial_delay_ms AS job_retry_initial_delay_ms
  , r.retry_max_delay_ms AS job_retry_max_delay_ms
FROM job.execution AS e
INNER JOIN job.job_revision AS r ON e.job_id = r.job_id AND e.job_revision_id = r.revision_id
INNER JOIN job.job AS j ON e.job_id = j.job_id
;

CREATE VIEW job.execution_detail_with_events AS
SELECT
  e.*
  , ee.status AS event_status
  , ee.created_at AS event_created_at
FROM job.execution_detail AS e
INNER JOIN job.execution_event AS ee ON e.execution_id = ee.execution_id
;

CREATE VIEW job.execution_claim_detail AS
SELECT
  e.execution_id
  , e.job_id
  , e.job_revision_id
  , e.attempt
  , e.lease_token
  , e.lease_expires_at
  , r.type AS job_type
  , r.config AS job_config
FROM job.execution AS e
INNER JOIN job.job_revision AS r ON e.job_id = r.job_id AND e.job_revision_id = r.revision_id
;

CREATE INDEX execution_status_next_run_at_idx
ON job.execution(status, next_run_at)
;

-- +goose Down
DROP INDEX execution_status_next_run_at_idx;
DROP VIEW job.execution_claim_detail;
DROP VIEW job.execution_detail_with_events;
DROP VIEW job.execution_detail;

DROP VIEW job.job_detail;

DROP INDEX job_created_at_job_id;

ALTER TABLE job.job
  ADD COLUMN type VARCHAR(30)
  , ADD COLUMN config JSONB

  , ADD COLUMN retry_max_attempts INTEGER
  , ADD COLUMN retry_backoff_type VARCHAR(50)
  , ADD COLUMN retry_initial_delay_ms BIGINT
  , ADD COLUMN retry_max_delay_ms BIGINT
;

UPDATE job.job AS j
SET
  type = t.type
  , config = t.config
  , retry_max_attempts = t.retry_max_attempts
  , retry_backoff_type = t.retry_backoff_type
  , retry_initial_delay_ms = t.retry_initial_delay_ms
  , retry_max_delay_ms = t.retry_max_delay_ms
FROM (
    SELECT jr.* FROM job.job_revision AS jr
    INNER JOIN job.job_current_revision AS jcr USING (revision_id)
) AS t
WHERE j.job_id = t.job_id
;

ALTER TABLE job.job
  ALTER COLUMN type SET NOT NULL
  , ALTER COLUMN config SET NOT NULL

  , ALTER COLUMN retry_max_attempts SET NOT NULL
  , ALTER COLUMN retry_max_attempts SET DEFAULT 1
  , ALTER COLUMN retry_backoff_type SET NOT NULL
  , ALTER COLUMN retry_initial_delay_ms SET NOT NULL
  , ALTER COLUMN retry_initial_delay_ms SET DEFAULT 0
  , ALTER COLUMN retry_max_delay_ms SET NOT NULL
  , ADD CONSTRAINT fk_job_type FOREIGN KEY (type) REFERENCES job.job_type(type) ON UPDATE CASCADE
  , ADD CONSTRAINT fk_job_retry_backoff_type FOREIGN KEY (retry_backoff_type) REFERENCES job.job_retry_backoff_type(type) ON UPDATE CASCADE
;

UPDATE job.execution
SET result = '{}'::jsonb
WHERE result IS NULL;

ALTER TABLE job.execution
  ALTER COLUMN result SET NOT NULL
  , ALTER COLUMN attempt SET DEFAULT 1
  , DROP COLUMN job_revision_id
  , ADD COLUMN max_attempts INTEGER NOT NULL DEFAULT 1
;

DROP TABLE job.job_current_revision;
DROP TABLE job.job_revision;
