-- Create additional databases needed by services sharing the main postgres instance
-- The default database (userauth) is created by POSTGRES_DB env var

SELECT 'CREATE DATABASE onboarding'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'onboarding')\gexec
