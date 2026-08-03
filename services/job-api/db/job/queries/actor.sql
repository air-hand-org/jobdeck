-- name: GetActor :one
SELECT
    *
FROM job.actor
WHERE
    actor_id = sqlc.arg(actor_id)
;

-- name: CreateActor :one
INSERT INTO job.actor_projection
    (
        actor_id
        , type
    )
VALUES
    (
        sqlc.arg(actor_id)
        , sqlc.arg(type)
    )
RETURNING actor_id
;
