-- QuantumLayer Resilience Fabric - Initial Schema
-- Migration: 000001_init_schema
-- Description: Creates core tables for multi-tenancy, images, assets, and drift

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- Multi-tenancy Tables
-- =============================================================================

-- Organizations (tenants)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(63) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Projects within organizations
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(63) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, slug)
);

-- Environments within projects
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(63) NOT NULL, -- prod, staging, dev, dr
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, name)
);

-- Users (linked to Clerk)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(255) UNIQUE NOT NULL, -- Clerk user ID
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(31) NOT NULL DEFAULT 'viewer',
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Golden Image Registry
-- =============================================================================

-- Golden images
CREATE TABLE images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    family VARCHAR(255) NOT NULL,
    version VARCHAR(63) NOT NULL,
    os_name VARCHAR(63),
    os_version VARCHAR(63),
    cis_level INT,
    sbom_url TEXT,
    signed BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(31) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, family, version)
);

-- Platform-specific image coordinates
CREATE TABLE image_coordinates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    platform VARCHAR(31) NOT NULL, -- aws, azure, gcp, vsphere, k8s, baremetal
    region VARCHAR(63),
    identifier TEXT NOT NULL, -- ami-xxx, /subscriptions/.../versions/x, template name
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(image_id, platform, region)
);

-- =============================================================================
-- Fleet Inventory
-- =============================================================================

-- Discovered assets
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    env_id UUID REFERENCES environments(id) ON DELETE SET NULL,
    platform VARCHAR(31) NOT NULL,
    account VARCHAR(63), -- AWS account ID, Azure subscription, GCP project
    region VARCHAR(63),
    site VARCHAR(63), -- Logical site name (dc-london, dc-singapore)
    instance_id VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    image_ref VARCHAR(255), -- AMI ID, template name, etc.
    image_version VARCHAR(63),
    state VARCHAR(31) NOT NULL DEFAULT 'unknown',
    tags JSONB,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, platform, instance_id)
);

-- =============================================================================
-- Drift Analysis
-- =============================================================================

-- Drift reports (point-in-time snapshots)
CREATE TABLE drift_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    env_id UUID REFERENCES environments(id) ON DELETE SET NULL,
    platform VARCHAR(31),
    site VARCHAR(63),
    total_assets INT NOT NULL DEFAULT 0,
    compliant_assets INT NOT NULL DEFAULT 0,
    coverage_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
    status VARCHAR(31) NOT NULL DEFAULT 'unknown',
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Connector Configuration
-- =============================================================================

-- Cloud connector configurations
CREATE TABLE connectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    platform VARCHAR(31) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB NOT NULL DEFAULT '{}', -- Platform-specific config (encrypted sensitive data)
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(31),
    last_sync_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- =============================================================================
-- Indexes
-- =============================================================================

-- Organizations
CREATE INDEX idx_organizations_slug ON organizations(slug);

-- Projects
CREATE INDEX idx_projects_org_id ON projects(org_id);

-- Environments
CREATE INDEX idx_environments_project_id ON environments(project_id);

-- Users
CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_external_id ON users(external_id);

-- Images
CREATE INDEX idx_images_org_id ON images(org_id);
CREATE INDEX idx_images_family ON images(org_id, family);
CREATE INDEX idx_images_status ON images(status);

-- Image Coordinates
CREATE INDEX idx_image_coordinates_image_id ON image_coordinates(image_id);
CREATE INDEX idx_image_coordinates_platform ON image_coordinates(platform);

-- Assets
CREATE INDEX idx_assets_org_id ON assets(org_id);
CREATE INDEX idx_assets_org_env ON assets(org_id, env_id);
CREATE INDEX idx_assets_platform ON assets(platform);
CREATE INDEX idx_assets_region ON assets(region);
CREATE INDEX idx_assets_site ON assets(site);
CREATE INDEX idx_assets_state ON assets(state);
CREATE INDEX idx_assets_image_ref ON assets(image_ref);
CREATE INDEX idx_assets_discovered_at ON assets(discovered_at DESC);

-- Drift Reports
CREATE INDEX idx_drift_reports_org_id ON drift_reports(org_id);
CREATE INDEX idx_drift_reports_env_id ON drift_reports(env_id);
CREATE INDEX idx_drift_reports_calculated_at ON drift_reports(org_id, calculated_at DESC);
CREATE INDEX idx_drift_reports_status ON drift_reports(status);

-- Connectors
CREATE INDEX idx_connectors_org_id ON connectors(org_id);
CREATE INDEX idx_connectors_platform ON connectors(platform);

-- =============================================================================
-- Functions
-- =============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers
CREATE TRIGGER update_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_images_updated_at
    BEFORE UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_connectors_updated_at
    BEFORE UPDATE ON connectors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
