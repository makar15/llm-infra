-- Creates two databases if they do not already exist.
-- Runs once on first container start against the default "postgres" DB.

SELECT 'CREATE DATABASE langfuse'
  WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'langfuse')\gexec

SELECT 'CREATE DATABASE litellm'
  WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'litellm')\gexec
