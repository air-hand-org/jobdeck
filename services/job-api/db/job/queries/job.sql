-- name: GetJob :one
SELECT * FROM job.job_detail
WHERE job_id = sqlc.arg(job_id) AND owner_id = sqlc.arg(owner_id);

-- name: ListJobsFirstPage :many
SELECT * FROM job.job_detail
WHERE owner_id = sqlc.arg(owner_id)
ORDER BY created_at DESC, job_id DESC
LIMIT sqlc.arg(page_size)
;

-- name: ListJobsAfter :many
SELECT * FROM job.job_detail
WHERE owner_id = sqlc.arg(owner_id)
  AND (
    created_at < sqlc.arg(created_at)
    OR (created_at = sqlc.arg(created_at) AND job_id < sqlc.arg(job_id))
  )
ORDER BY created_at DESC, job_id DESC
LIMIT sqlc.arg(page_size)
;

-- name: CreateJob :one
WITH j AS (
    INSERT INTO job.job (
    	owner_id
    	, enabled
        , name
    ) VALUES (
    	sqlc.arg(owner_id)
    	, TRUE
        , sqlc.arg(name)
    ) RETURNING *
),
r AS (
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
    	j.job_id
    	, 1
    	, sqlc.arg(type)
    	, sqlc.arg(config)
    	, sqlc.arg(retry_max_attempts)
    	, sqlc.arg(retry_backoff_type)
    	, sqlc.arg(retry_initial_delay_ms)
    	, sqlc.arg(retry_max_delay_ms)
    FROM j
    RETURNING *
),
cr AS (
    INSERT INTO job.job_current_revision (
        job_id
        , revision_id
    )
    SELECT
        r.job_id
        , r.revision_id
    FROM r
    RETURNING *
)
SELECT
  job_id
FROM cr
;

-- name: UpdateJobSpec :one
WITH touched_job AS (
    UPDATE job.job AS j
    SET updated_at = NOW()
    WHERE
        j.job_id = sqlc.arg(job_id)
        AND j.owner_id = sqlc.arg(owner_id)
    RETURNING job_id
)
, current AS (
    SELECT
      cr.job_id
      , cr.revision_id
      , r.revision_number
    FROM job.job_current_revision AS cr
    INNER JOIN touched_job AS tj ON tj.job_id = cr.job_id
    INNER JOIN job.job_revision AS r ON cr.job_id = r.job_id AND cr.revision_id = r.revision_id
    FOR UPDATE OF cr
)
, r AS (
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
        current.job_id
        , current.revision_number + 1
       	, sqlc.arg(type)
       	, sqlc.arg(config)
       	, sqlc.arg(retry_max_attempts)
       	, sqlc.arg(retry_backoff_type)
       	, sqlc.arg(retry_initial_delay_ms)
       	, sqlc.arg(retry_max_delay_ms)
    FROM current
    RETURNING job_id, revision_id
)
, updated_current AS (
    UPDATE job.job_current_revision AS cr
    SET revision_id = r.revision_id
    FROM r
    WHERE cr.job_id = r.job_id
    RETURNING cr.*
)
SELECT
  u.job_id
FROM updated_current AS u
;

-- name: SetJobEnabled :one
WITH updated AS (
    UPDATE job.job AS j
    SET
        enabled = sqlc.arg(enabled)
    WHERE
        j.job_id = sqlc.arg(job_id)
        AND j.owner_id = sqlc.arg(owner_id)
    RETURNING *
)
SELECT
  u.job_id
FROM updated AS u
;

-- name: RenameJob :execrows
UPDATE job.job
SET
  name = sqlc.arg(name)
WHERE
  job_id = sqlc.arg(job_id)
  AND owner_id = sqlc.arg(owner_id)
;

-- name: DeleteJob :execrows
DELETE FROM job.job
WHERE
  job_id = sqlc.arg(job_id)
  AND owner_id = sqlc.arg(owner_id)
;
