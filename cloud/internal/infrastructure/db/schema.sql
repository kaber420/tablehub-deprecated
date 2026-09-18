-- 001_init.sql
-- Initial schema for Tablehub Cloud SaaS
-- Creates the core multi-tenant tables: organizations and hub_registry.

BEGIN;

-- ===========================================================================
-- Extensions
-- ===========================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ===========================================================================
-- Table: organizations
-- The primary tenant entity. Each organization is an isolated customer.
-- ===========================================================================
CREATE TABLE IF NOT EXISTS organizations (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug       TEXT        NOT NULL,
    name       TEXT        NOT NULL DEFAULT '',
    plan       TEXT        NOT NULL DEFAULT 'free'
                           CHECK (plan IN ('free', 'starter', 'pro', 'enterprise')),
    settings   JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT organizations_slug_unique UNIQUE (slug)
);

-- Index for slug lookups (used in auth and webhook routing)
CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations (slug);

-- Index for plan-based filtering (admin dashboards)
CREATE INDEX IF NOT EXISTS idx_organizations_plan ON organizations (plan);

-- ===========================================================================
-- Table: hub_registry
-- Tracks every physical hub device registered to a organization.
-- A organization can have multiple hubs.
-- ===========================================================================
CREATE TABLE IF NOT EXISTS hub_registry (
    id                TEXT        PRIMARY KEY,
    organization_id     UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    public_key        TEXT        NOT NULL,
    connection_status TEXT        NOT NULL DEFAULT 'offline'
                                  CHECK (connection_status IN ('online', 'offline', 'authenticating')),
    last_seen         TIMESTAMPTZ,
    metadata          JSONB       NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for organization lookups (list all hubs for a organization)
CREATE INDEX IF NOT EXISTS idx_hub_registry_organization_id ON hub_registry (organization_id);

-- Index for online hub queries (active tunnel lookup)
CREATE INDEX IF NOT EXISTS idx_hub_registry_status ON hub_registry (connection_status)
    WHERE connection_status = 'online';

-- Composite index for organization and connection status
CREATE INDEX IF NOT EXISTS idx_hub_registry_rest_status ON hub_registry (organization_id, connection_status);

-- Index for public key lookups (used in auth)
CREATE INDEX IF NOT EXISTS idx_hub_registry_pubkey ON hub_registry (public_key);

-- Partial index for last_seen (useful for stale connection detection)
CREATE INDEX IF NOT EXISTS idx_hub_registry_last_seen ON hub_registry (last_seen)
    WHERE last_seen IS NOT NULL;

-- ===========================================================================
-- Auto-update updated_at trigger
-- ===========================================================================
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_organizations_updated_at ON organizations;
CREATE TRIGGER set_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

DROP TRIGGER IF EXISTS set_hub_registry_updated_at ON hub_registry;
CREATE TRIGGER set_hub_registry_updated_at
    BEFORE UPDATE ON hub_registry
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMIT;
-- 002_webhook_events.sql
-- Adds the webhook_events table for auditing and tracking webhook deliveries.

BEGIN;

CREATE TABLE webhook_events (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,
    event_type        TEXT NOT NULL,
    payload           JSONB NOT NULL,
    idempotency_key   TEXT,
    status            TEXT NOT NULL DEFAULT 'received'
                      CHECK (status IN ('received', 'delivered', 'failed', 'hub_offline')),
    retry_count       INT NOT NULL DEFAULT 0,
    hub_id            TEXT,
    error_detail      TEXT,
    received_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at      TIMESTAMPTZ
);

-- Índice compuesto para la query principal (listado por organizatione)
CREATE INDEX idx_webhook_events_organization_received
    ON webhook_events(organization_id, received_at DESC);

-- Índice único parcial para idempotencia (dedup de proveedores)
CREATE UNIQUE INDEX idx_webhook_idempotency
    ON webhook_events(organization_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Índice para monitoreo por status
CREATE INDEX idx_webhook_events_status_received
    ON webhook_events(status, received_at DESC);

COMMIT;
-- 003_retry_ack.sql
-- Adds ack_deadline, next_retry_at, expires_at to webhook_events and updates status check

BEGIN;

ALTER TABLE webhook_events
  ADD COLUMN ack_deadline TIMESTAMPTZ,
  ADD COLUMN next_retry_at TIMESTAMPTZ,
  ADD COLUMN expires_at TIMESTAMPTZ;

-- Drop the old constraint
ALTER TABLE webhook_events DROP CONSTRAINT webhook_events_status_check;

-- Add the new constraint with 'expired'
ALTER TABLE webhook_events ADD CONSTRAINT webhook_events_status_check 
  CHECK (status IN ('received', 'delivered', 'failed', 'hub_offline', 'expired'));

-- Indices para busqueda de retries y expiraciones en el worker
CREATE INDEX idx_webhook_events_retry
    ON webhook_events(next_retry_at)
    WHERE next_retry_at IS NOT NULL;

CREATE INDEX idx_webhook_events_ack_deadline
    ON webhook_events(ack_deadline)
    WHERE ack_deadline IS NOT NULL;

COMMIT;
-- 004_users_business.sql
-- Creates the users table linking Zitadel identities to Tablehub organizations.

BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    zitadel_user_id TEXT        NOT NULL UNIQUE,
    organization_id UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role            TEXT        NOT NULL DEFAULT 'owner' CHECK (role IN ('owner', 'admin', 'viewer')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_zitadel_user_id ON users(zitadel_user_id);
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);

DROP TRIGGER IF EXISTS set_users_updated_at ON users;
CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMIT;
-- 005_branches_and_access.sql
-- Creates the branches and access_requests tables, and alters existing structures.

BEGIN;

-- 1. Create branches table
CREATE TABLE IF NOT EXISTS branches (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL DEFAULT '',
    address         TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_branches_organization_id ON branches(organization_id);

-- 2. Modify users table to allow NULL organization_id, support 'superadmin' role, and add name/email columns
ALTER TABLE users ALTER COLUMN organization_id DROP NOT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';

-- Drop check constraint if exists and recreate it to include 'superadmin'
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('superadmin', 'owner', 'admin', 'viewer'));

-- 3. Create access_requests table
CREATE TABLE IF NOT EXISTS access_requests (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_role  TEXT        NOT NULL CHECK (requested_role IN ('admin', 'viewer')),
    status          TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_pending_request UNIQUE (user_id, organization_id, status)
);

CREATE INDEX IF NOT EXISTS idx_access_requests_org_status ON access_requests(organization_id, status);

-- 4. Alter hub_registry to link to branches
ALTER TABLE hub_registry ADD COLUMN IF NOT EXISTS branch_id UUID REFERENCES branches(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_hub_registry_branch_id ON hub_registry(branch_id);

-- 5. Triggers for updated_at
DROP TRIGGER IF EXISTS set_branches_updated_at ON branches;
CREATE TRIGGER set_branches_updated_at
    BEFORE UPDATE ON branches
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

DROP TRIGGER IF EXISTS set_access_requests_updated_at ON access_requests;
CREATE TRIGGER set_access_requests_updated_at
    BEFORE UPDATE ON access_requests
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMIT;
-- 006_user_provider_agnostic.sql
-- Renames zitadel_user_id to provider_user_id in users table to make the system identity-provider agnostic.

BEGIN;

ALTER TABLE users RENAME COLUMN zitadel_user_id TO provider_user_id;

DROP INDEX IF EXISTS idx_users_zitadel_user_id;
CREATE INDEX IF NOT EXISTS idx_users_provider_user_id ON users(provider_user_id);

COMMIT;
-- 007_org_provider_id.sql
-- Adds provider_org_id to organizations to associate internal UUIDs with Logto/Zitadel organization IDs.

BEGIN;

ALTER TABLE organizations ADD COLUMN provider_org_id TEXT NOT NULL UNIQUE;

CREATE INDEX IF NOT EXISTS idx_organizations_provider_org_id ON organizations(provider_org_id);

COMMIT;
-- 008_hub_bootstrap_token.sql
-- Relaxes the public_key NOT NULL constraint in hub_registry and adds a bootstrap_token field.

BEGIN;

ALTER TABLE hub_registry ALTER COLUMN public_key DROP NOT NULL;
ALTER TABLE hub_registry ADD COLUMN IF NOT EXISTS bootstrap_token TEXT;

COMMIT;
BEGIN;
DROP TABLE IF EXISTS access_requests CASCADE;
COMMIT;
-- 010_roles_redesign.sql
-- Replaces 'superadmin' role with staff roles: platform_admin, support, billing

BEGIN;

-- Update existing superadmin users to platform_admin
UPDATE users SET role = 'platform_admin' WHERE role = 'superadmin';

-- Replace role constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check 
    CHECK (role IN ('platform_admin', 'support', 'billing', 'owner', 'admin', 'viewer'));

COMMIT;
-- 1. Agregar columna con default seguro para usuarios existentes
ALTER TABLE users ADD COLUMN onboarding_state TEXT DEFAULT 'active';

-- 2. Migrar datos: usuarios con org → 'active'
UPDATE users SET onboarding_state = 'active' WHERE organization_id IS NOT NULL;

-- 3. Hacer organization_id nullable (users nuevos no tendrán org aún)
ALTER TABLE users ALTER COLUMN organization_id DROP NOT NULL;

-- 4. Crear tabla pending_operations
CREATE TABLE pending_operations (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    operation_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', 
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    next_retry_at TIMESTAMP,
    last_error TEXT,
    result JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 5. Índice para prevenir duplicados en race conditions
CREATE UNIQUE INDEX idx_pending_ops_user_active 
ON pending_operations (user_id) 
WHERE status IN ('pending', 'running');

COMMIT;

-- 012_branches_unique.sql
-- Adds UNIQUE constraint to branches on organization_id and name
BEGIN;

ALTER TABLE branches ADD CONSTRAINT unique_branch_name_per_org UNIQUE (organization_id, name);

COMMIT;

