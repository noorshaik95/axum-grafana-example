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

SELECT 'CREATE DATABASE feature_flags'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'feature_flags')\gexec

SELECT 'CREATE DATABASE adminauth'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'adminauth')\gexec

SELECT 'CREATE DATABASE incidentdb'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'incidentdb')\gexec

SELECT 'CREATE DATABASE scheduling'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'scheduling')\gexec

SELECT 'CREATE DATABASE discussion_service'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'discussion_service')\gexec

SELECT 'CREATE DATABASE slate_video'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'slate_video')\gexec
