-- +goose Up
CREATE SCHEMA IF NOT EXISTS auth;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION auth.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

CREATE TABLE auth.actor_type (
    type VARCHAR(30) PRIMARY KEY
);
INSERT INTO
    auth.actor_type (type)
VALUES
    ('USER')
    , ('SERVICE')
;

CREATE TABLE auth.actor (
    actor_id UUID PRIMARY KEY DEFAULT uuidv7()
    , name VARCHAR(50) NOT NULL
    , type VARCHAR(30) NOT NULL
    , active BOOLEAN NOT NULL DEFAULT TRUE

    , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    , updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

    , CONSTRAINT fk_type FOREIGN KEY (type) REFERENCES auth.actor_type(type) ON UPDATE CASCADE
);

CREATE TRIGGER actor_updated_at
BEFORE UPDATE ON auth.actor
FOR EACH ROW
EXECUTE FUNCTION auth.update_updated_at_column()
;

CREATE TABLE auth.actor_identity (
    actor_id UUID PRIMARY KEY

    , issuer TEXT NOT NULL
    , subject TEXT NOT NULL

    , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    , updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

    , CONSTRAINT fk_actor_id FOREIGN KEY (actor_id) REFERENCES auth.actor(actor_id) ON DELETE CASCADE
    , CONSTRAINT uq_issuer_subject UNIQUE (issuer, subject)
);

CREATE TRIGGER actor_identity_updated_at
BEFORE UPDATE ON auth.actor_identity
FOR EACH ROW
EXECUTE FUNCTION auth.update_updated_at_column()
;

CREATE TABLE auth.local_auth_client_credential (
    actor_id UUID PRIMARY KEY

    , client_id TEXT NOT NULL
    , secret_hash TEXT NOT NULL

    , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    , updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

    , CONSTRAINT fk_actor_id FOREIGN KEY (actor_id) REFERENCES auth.actor(actor_id) ON DELETE CASCADE
    , CONSTRAINT uq_client_id UNIQUE (client_id)
);

CREATE TRIGGER local_auth_client_credential_updated_at
BEFORE UPDATE ON auth.local_auth_client_credential
FOR EACH ROW
EXECUTE FUNCTION auth.update_updated_at_column()
;

CREATE OR REPLACE VIEW auth.local_auth_client_detail AS
SELECT
    c.actor_id
    , c.client_id
    , c.secret_hash
    , a.name
    , a.type
    , a.active
FROM auth.local_auth_client_credential AS c
INNER JOIN auth.actor AS a USING (actor_id)
;


WITH master_data (actor_id, type, name, has_identity, issuer, subject, is_local_auth, client_id, secret_hash) AS (
    VALUES
        -- Fallback user
        (
            'ffffffff-ffff-ffff-ffff-ffffffffffff'::uuid
            , 'USER'
            , 'fallback user'
            , FALSE
            , NULL
            , NULL
            , FALSE
            , NULL
            , NULL
        )
        ,(
            'ffffffff-ffff-ffff-ffff-fffffffffffe'::uuid
            , 'SERVICE'
            , 'worker'
            , TRUE
            , 'local-auth'
            , 'job-worker'
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
DROP SCHEMA IF EXISTS auth CASCADE;
