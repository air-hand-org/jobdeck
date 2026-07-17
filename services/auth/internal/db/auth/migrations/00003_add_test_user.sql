-- +goose Up
WITH master_data (actor_id, type, name, has_identity, issuer, subject, is_local_auth, client_id, secret_hash) AS (
    VALUES
        -- Test user
        (
            'ffffffff-ffff-ffff-ffff-fffffffffffd'::uuid
            , 'USER'
            , 'test user'
            , TRUE
            , 'local-auth'
            , 'test-user'
            , FALSE
            , NULL
            , NULL
        )
)
, inserted_actors AS (
    INSERT INTO auth.actor
        (actor_id, type, name)
    SELECT
        actor_id, type, name
    FROM master_data
    RETURNING actor_id
)
, inserted_actor_identities AS (
    INSERT INTO auth.actor_identity (
        actor_id
        , issuer
        , subject
    )
    SELECT
        m.actor_id
        , m.issuer
        , m.subject
    FROM master_data AS m
    INNER JOIN inserted_actors AS ia USING (actor_id)
    WHERE m.has_identity = TRUE
    RETURNING actor_id
)
INSERT INTO auth.local_auth_client_credential (
    actor_id
    , client_id
    , secret_hash
)
SELECT
    m.actor_id
    , m.client_id
    , m.secret_hash
FROM master_data AS m
INNER JOIN inserted_actors AS ia USING (actor_id)
WHERE m.is_local_auth = TRUE
;

-- +goose Down
WITH master_data (actor_id, type, name, has_identity, issuer, subject, is_local_auth, client_id, secret_hash) AS (
    VALUES
        -- Test user
        (
            'ffffffff-ffff-ffff-ffff-fffffffffffd'::uuid
            , 'USER'
            , 'test user'
            , TRUE
            , 'local-auth'
            , 'test-user'
            , FALSE
            , NULL
            , NULL
        )
)
DELETE FROM auth.actor AS a
USING master_data AS m
WHERE
    a.actor_id = m.actor_id
;
