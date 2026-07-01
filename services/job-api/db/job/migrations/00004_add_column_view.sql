-- +goose Up
CREATE OR REPLACE VIEW job.execution_claim_detail AS
SELECT
  e.execution_id
  , e.job_id
  , e.job_revision_id
  , e.attempt
  , e.lease_token
  , e.lease_expires_at
  , r.type AS job_type
  , r.config AS job_config
  , e.created_at -- added column
FROM job.execution AS e
INNER JOIN job.job_revision AS r ON e.job_id = r.job_id AND e.job_revision_id = r.revision_id
;

-- +goose Down
CREATE OR REPLACE VIEW job.execution_claim_detail AS
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
