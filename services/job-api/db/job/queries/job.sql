-- name: GetJob :one
SELECT * FROM job.job
WHERE job_id = $1 AND owner_id = $2;

-- name: ListJobsByOwner :many
SELECT * FROM job.job
WHERE owner_id = $1
ORDER BY job_id;

-- name: CreateJob :one
INSERT INTO job.job (
	owner_id
	, enabled
	, name
	, type
	, config
	, retry_max_attempts
	, retry_backoff_type
	, retry_initial_delay_ms
	, retry_max_delay_ms
) VALUES (
	$1
	, TRUE
	, $2
	, $3
	, $4
	, $5
	, $6
	, $7
	, $8
) RETURNING *
;

-- name: UpdateJobSpec :one
UPDATE job.job SET
  name = $1
  , type = $2
  , config = $3
  , retry_max_attempts = $4
  , retry_backoff_type = $5
  , retry_initial_delay_ms = $6
  , retry_max_delay_ms = $7
WHERE
  job_id = $8
  AND owner_id = $9
RETURNING *
;

-- name: SetJobEnabled :one
UPDATE job.job SET
  enabled = $1
WHERE
  job_id = $2
  AND owner_id = $3
RETURNING *
;

-- name: DeleteJob :execrows
DELETE FROM job.job
WHERE job_id = $1 AND owner_id = $2;
