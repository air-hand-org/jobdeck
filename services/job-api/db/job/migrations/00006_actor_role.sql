-- +goose Up
CREATE TABLE job.actor_role (
    role VARCHAR(20) PRIMARY KEY
);
INSERT INTO job.actor_role
    (role)
VALUES
    ('WORKER')
;

CREATE TABLE job.actor_projection (
    actor_id UUID PRIMARY KEY

    , type VARCHAR(30) NOT NULL
    , active BOOLEAN NOT NULL DEFAULT TRUE

    , created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    , updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER actor_projection_updated_at
BEFORE UPDATE ON job.actor_projection
FOR EACH ROW
EXECUTE FUNCTION job.update_updated_at_column()
;

CREATE TABLE job.actor_role_assignment (
    actor_id UUID PRIMARY KEY

    , role VARCHAR(20) NOT NULL

    , CONSTRAINT fk_actor_id FOREIGN KEY (actor_id) REFERENCES job.actor_projection(actor_id) ON DELETE NO ACTION
    , CONSTRAINT fk_role FOREIGN KEY (role) REFERENCES job.actor_role(role) ON UPDATE CASCADE
);

WITH master_data (actor_id, type, role) AS (
    VALUES
        (
            'ffffffff-ffff-ffff-ffff-fffffffffffe'::uuid
            , 'SERVICE'
            , 'WORKER'
        )
)
, inserted_actors AS (
    INSERT INTO job.actor_projection
        (actor_id, type)
    SELECT
        actor_id, type
    FROM master_data
    RETURNING actor_id
)
INSERT INTO job.actor_role_assignment
    (actor_id, role)
SELECT
    m.actor_id
    , m.role
FROM master_data AS m
INNER JOIN inserted_actors AS ia USING (actor_id)
;

CREATE OR REPLACE VIEW job.actor AS
SELECT
    ap.actor_id
    , ap.type
    , ap.active
    , ar.role
FROM job.actor_projection AS ap
LEFT JOIN job.actor_role_assignment AS ar USING (actor_id)
;

-- +goose Down
DROP VIEW job.actor;
DROP TABLE job.actor_role_assignment;
DROP TABLE job.actor_projection;
DROP TABLE job.actor_role;
