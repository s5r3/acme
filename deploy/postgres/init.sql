-- Runs once when the Docker volume is created: a separate database for
-- integration tests (they truncate every table).
CREATE DATABASE acme_test OWNER acme;
