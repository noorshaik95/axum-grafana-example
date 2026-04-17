-- Create additional databases needed by services sharing the main postgres instance
-- The default database (userauth) is created by POSTGRES_DB env var

SELECT 'CREATE DATABASE onboarding'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'onboarding')\gexec

SELECT 'CREATE DATABASE tenantdb'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'tenantdb')\gexec

SELECT 'CREATE DATABASE emaildb'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'emaildb')\gexec

SELECT 'CREATE DATABASE metricsdb'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'metricsdb')\gexec
