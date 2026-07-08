-- +goose Up
DROP VIEW job.execution_detail;

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

ALTER TABLE job.execution
  DROP COLUMN error_message
;

-- +goose Down
ALTER TABLE job.execution
  ADD COLUMN error_message TEXT
;

DROP VIEW job.execution_detail;

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
