-- +goose Up
DROP VIEW job.execution_detail_with_events;

-- +goose Down
CREATE VIEW job.execution_detail_with_events AS
SELECT
  e.*
  , ee.status AS event_status
  , ee.created_at AS event_created_at
FROM job.execution_detail AS e
INNER JOIN job.execution_event AS ee ON e.execution_id = ee.execution_id
;
