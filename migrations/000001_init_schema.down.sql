-- QuantumLayer Resilience Fabric - Initial Schema Rollback
-- Migration: 000001_init_schema
-- Description: Drops all tables created in the up migration

-- Drop triggers first
DROP TRIGGER IF EXISTS update_connectors_updated_at ON connectors;
DROP TRIGGER IF EXISTS update_assets_updated_at ON assets;
DROP TRIGGER IF EXISTS update_images_updated_at ON images;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of creation (respecting foreign keys)
DROP TABLE IF EXISTS connectors;
DROP TABLE IF EXISTS drift_reports;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS image_coordinates;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS environments;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organizations;

-- Note: We don't drop the uuid-ossp extension as it may be used by other schemas
