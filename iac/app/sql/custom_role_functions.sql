-- Create Custom Roles

CREATE OR REPLACE FUNCTION create_role_if_not_exists(
  p_rolname text
)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = p_rolname) THEN
    EXECUTE FORMAT('CREATE ROLE %I NOLOGIN', p_rolname);
  END IF;
END
$$;

CREATE OR REPLACE FUNCTION create_migrator_role(
  service_name text
)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
  migrator_role text := service_name || '-migrator';
BEGIN
  PERFORM create_role_if_not_exists(migrator_role);
  EXECUTE FORMAT('GRANT USAGE, CREATE ON SCHEMA %I TO %I', 'public', migrator_role);
  EXECUTE FORMAT('GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA %I TO %I', 'public', migrator_role);
  EXECUTE FORMAT('GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA %I TO %I', 'public', migrator_role);
  EXECUTE FORMAT('CREATE SCHEMA IF NOT EXISTS %I', service_name);
  EXECUTE FORMAT('GRANT USAGE, CREATE ON SCHEMA %I TO %I', service_name, migrator_role);
  EXECUTE FORMAT('GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA %I TO %I', service_name, migrator_role);
  EXECUTE FORMAT('GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA %I TO %I', service_name, migrator_role);
END
$$;

CREATE OR REPLACE FUNCTION create_runtime_role(
  service_name text
)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
  migrator_role text := service_name || '-migrator';
  runtime_role text := service_name || '-runtime';
BEGIN
  PERFORM create_role_if_not_exists(runtime_role);
  EXECUTE FORMAT('GRANT USAGE ON SCHEMA %I TO %I', service_name, runtime_role);
  EXECUTE FORMAT('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %I', migrator_role, service_name, runtime_role);
  EXECUTE FORMAT('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA %I GRANT USAGE, SELECT ON SEQUENCES TO %I', migrator_role, service_name, runtime_role);
END
$$;
