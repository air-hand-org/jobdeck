-- name: CreateActor :execrows
WITH a AS (
    INSERT INTO auth.actor (
        name
        , type
    ) VALUES (
        sqlc.arg(name)
        , sqlc.arg(type)
    )
    RETURNING *
)
INSERT INTO auth.actor_identity (
    actor_id
    , issuer
    , subject
)
SELECT
    a.actor_id
    , sqlc.arg(issuer)
    , sqlc.arg(subject)
FROM a
;

-- name: GetMatchedActorIdentity :one
SELECT a.*
FROM auth.actor_identity AS i
INNER JOIN auth.actor AS a USING (actor_id)
WHERE
    i.issuer = sqlc.arg(issuer)
    AND
    i.subject = sqlc.arg(subject)
;

-- name: GetMatchedLocalAuthClient :one
SELECT *
FROM auth.local_auth_client_detail
WHERE
    client_id = sqlc.arg(client_id)
;

-- name: UpdateLocalAuthClientSecretHash :execrows
UPDATE auth.local_auth_client_credential
SET secret_hash = sqlc.arg(secret_hash)
WHERE
    actor_id = sqlc.arg(actor_id)
;
