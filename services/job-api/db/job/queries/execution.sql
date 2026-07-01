-- name: CreateExecutionForCurrentJob :one
WITH added AS (
  INSERT INTO job.execution AS e (
    job_id
    , job_revision_id
    , trigger_type
    , status
    , attempt
    , next_run_at
  )
  SELECT
    cr.job_id
    , cr.revision_id
    , sqlc.arg(trigger_type)
    , 'QUEUED'
    , 0
    , NOW()
  FROM job.job_current_revision AS cr
  INNER JOIN job.job AS j ON cr.job_id = j.job_id
  WHERE
    cr.job_id = sqlc.arg(job_id)
    AND
    j.owner_id = sqlc.arg(owner_id)
    AND
    j.enabled = TRUE
  RETURNING e.*
)
, added_event AS (
  INSERT INTO job.execution_event AS ee (
    execution_id
    , status
  )
  SELECT
    execution_id
    , status
  FROM added
  RETURNING ee.*
)
SELECT
  execution_id
FROM added_event;
;

-- name: GetExecutionDetail :one
SELECT *
FROM job.execution_detail AS e
WHERE
  e.execution_id = sqlc.arg(execution_id)
  AND
  e.job_owner_id = sqlc.arg(owner_id)
;

-- name: ListExecutionEvents :many
SELECT *
FROM job.execution_event AS ee
WHERE
  ee.execution_id = sqlc.arg(execution_id)
ORDER BY ee.created_at ASC, ee.execution_event_id ASC
;

-- name: ListExecutionsFirstPage :many
SELECT *
FROM job.execution_detail AS e
WHERE
  e.job_id = sqlc.arg(job_id)
  AND
  e.job_owner_id = sqlc.arg(owner_id)
ORDER BY e.created_at DESC, e.execution_id DESC
LIMIT sqlc.arg(page_size)
;

-- name: ListExecutionsAfter :many
SELECT *
FROM job.execution_detail AS e
WHERE
  e.job_id = sqlc.arg(job_id)
  AND
  e.job_owner_id = sqlc.arg(owner_id)
  AND (
    e.created_at < sqlc.arg(created_at)
    OR (e.created_at = sqlc.arg(created_at) AND e.execution_id < sqlc.arg(execution_id))
  )
ORDER BY e.created_at DESC, e.execution_id DESC
LIMIT sqlc.arg(page_size)
;

-- for ExecutionQueryService

-- name: CreateRetryForUnsuccessfulExecution :one
WITH added AS (
  INSERT INTO job.execution AS e (
    job_id
    , job_revision_id
    , trigger_type
    , status
    , attempt
    , next_run_at
  )
  SELECT
    cp.job_id
    , cp.job_revision_id
    , cp.trigger_type
    , 'QUEUED'
    , cp.attempt
    , sqlc.arg(next_run_at)
  FROM job.execution AS cp
  INNER JOIN job.job_revision AS r ON cp.job_id = r.job_id AND cp.job_revision_id = r.revision_id
  WHERE
    cp.execution_id = sqlc.arg(execution_id)
    AND
    cp.attempt < r.retry_max_attempts
    AND
    cp.status IN ('FAILED', 'TIMED_OUT', 'CANCELLED')
  RETURNING e.*
)
, added_event AS (
  INSERT INTO job.execution_event AS ee (
    execution_id
    , status
  )
  SELECT
    execution_id
    , status
  FROM added
  RETURNING ee.*
)
SELECT
  execution_id
FROM added_event
;

-- name: ClaimExecutions :many
WITH claimed AS (
  SELECT
    e.execution_id
  FROM job.execution AS e
  INNER JOIN job.job_revision AS r ON e.job_id = r.job_id AND e.job_revision_id = r.revision_id
  WHERE
    e.status = 'QUEUED'
    AND e.next_run_at <= NOW()
    AND e.attempt < r.retry_max_attempts
  ORDER BY e.next_run_at, e.created_at, e.execution_id
  LIMIT sqlc.arg(max_size)
  FOR UPDATE OF e SKIP LOCKED
)
, updated AS (
  UPDATE job.execution AS e
  SET
    status = 'RUNNING'
    , attempt = e.attempt + 1
    , lease_token = uuidv7()
    , lease_expires_at = sqlc.arg(lease_expires_at)
  FROM claimed AS c
  WHERE e.execution_id = c.execution_id
  RETURNING e.*
)
, added_event AS (
  INSERT INTO job.execution_event AS ee (
    execution_id
    , status
  )
  SELECT
    u.execution_id
    , u.status
  FROM updated AS u
  RETURNING ee.*
)
SELECT
  execution_id
FROM added_event
;

-- name: ReportExecutionResult :one
WITH updated AS (
  UPDATE job.execution AS e
  SET
    status = sqlc.arg(status)
    , result = sqlc.arg(result)
    , error_message = sqlc.arg(error_message)
  WHERE
    e.execution_id = sqlc.arg(execution_id)
    AND
    e.lease_token = sqlc.arg(lease_token)
    AND
    e.lease_expires_at > NOW()
  RETURNING e.*
)
, added_event AS (
  INSERT INTO job.execution_event AS ee (
    execution_id
    , status
  )
  SELECT
    u.execution_id
    , u.status
  FROM updated AS u
  RETURNING ee.*
)
SELECT
  execution_id
FROM added_event
;

-- name: GetExecutionDetailDirect :one
SELECT *
FROM job.execution_detail AS e
WHERE
  e.execution_id = sqlc.arg(execution_id)
;

-- name: ListExecutionClaimDetails :many
SELECT *
FROM job.execution_claim_detail AS e
WHERE
  e.execution_id = ANY(sqlc.arg(execution_ids)::uuid[])
ORDER BY e.created_at ASC, e.execution_id ASC
;
