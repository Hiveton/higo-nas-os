-- HiGoOS initial PostgreSQL 16 schema.
-- Requires pgcrypto. pgvector is enabled for semantic_embeddings.
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TYPE user_status AS ENUM ('active', 'locked', 'disabled');
CREATE TYPE session_status AS ENUM ('active', 'revoked', 'expired');
CREATE TYPE mfa_factor_type AS ENUM ('totp', 'webauthn', 'recovery_code');
CREATE TYPE permission_effect AS ENUM ('allow', 'deny');
CREATE TYPE access_level AS ENUM ('read', 'write', 'admin', 'owner');
CREATE TYPE window_status_tone AS ENUM ('blue', 'green', 'orange', 'red');
CREATE TYPE file_node_kind AS ENUM ('folder', 'file', 'album');
CREATE TYPE job_state AS ENUM ('queued', 'running', 'succeeded', 'failed', 'paused', 'canceled');
CREATE TYPE risk_level AS ENUM ('low', 'medium', 'high');
CREATE TYPE alert_tone AS ENUM ('blue', 'green', 'orange', 'red');
CREATE TYPE storage_health AS ENUM ('healthy', 'warning', 'critical', 'syncing', 'unknown');
CREATE TYPE disk_state AS ENUM ('healthy', 'warning', 'failed', 'spare', 'missing');
CREATE TYPE download_kind AS ENUM ('http', 'bt', 'magnet', 'rss');
CREATE TYPE container_state AS ENUM ('running', 'stopped', 'paused', 'exited', 'unknown');
CREATE TYPE model_runtime AS ENUM ('local', 'cloud', 'hybrid');
CREATE TYPE message_role AS ENUM ('system', 'user', 'assistant', 'tool');

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT NOT NULL UNIQUE CHECK (length(username) BETWEEN 2 AND 64),
  display_name TEXT NOT NULL CHECK (length(display_name) BETWEEN 1 AND 128),
  email TEXT UNIQUE CHECK (email IS NULL OR email ~* '^[^@]+@[^@]+\.[^@]+$'),
  password_hash TEXT,
  status user_status NOT NULL DEFAULT 'active',
  locale TEXT NOT NULL DEFAULT 'zh-CN',
  timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE groups (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9_-]{1,62}$'),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE group_members (
  group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_label TEXT NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, user_id)
);

CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9_-]{1,62}$'),
  name TEXT NOT NULL,
  permissions JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(permissions) = 'array'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_roles (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status session_status NOT NULL DEFAULT 'active',
  refresh_token_hash TEXT NOT NULL UNIQUE,
  ip_address INET,
  user_agent TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,
  CHECK (expires_at > created_at)
);

CREATE TABLE mfa_factors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  factor_type mfa_factor_type NOT NULL,
  label TEXT NOT NULL,
  secret_ref TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_used_at TIMESTAMPTZ
);

CREATE TABLE trusted_devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  device_name TEXT NOT NULL,
  fingerprint_hash TEXT NOT NULL,
  ip_address INET,
  last_seen_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, fingerprint_hash)
);

CREATE TABLE api_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  scopes JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(scopes) = 'array'),
  expires_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ
);

CREATE TABLE spaces (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9_-]{1,62}$'),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  owner_group_id UUID REFERENCES groups(id) ON DELETE SET NULL,
  quota_bytes BIGINT CHECK (quota_bytes IS NULL OR quota_bytes >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (owner_user_id IS NOT NULL OR owner_group_id IS NOT NULL)
);

CREATE TABLE dock_apps (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  icon_path TEXT NOT NULL,
  badge_count INTEGER CHECK (badge_count IS NULL OR badge_count >= 0),
  is_utility BOOLEAN NOT NULL DEFAULT false,
  sort_order INTEGER NOT NULL UNIQUE,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE desktop_windows (
  id TEXT PRIMARY KEY,
  app_id TEXT NOT NULL REFERENCES dock_apps(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  subtitle TEXT NOT NULL,
  status_text TEXT NOT NULL,
  status_tone window_status_tone NOT NULL,
  x INTEGER NOT NULL CHECK (x >= 0),
  y INTEGER NOT NULL CHECK (y >= 0),
  width INTEGER NOT NULL CHECK (width BETWEEN 240 AND 2000),
  height INTEGER NOT NULL CHECK (height BETWEEN 180 AND 1600),
  z_index INTEGER NOT NULL CHECK (z_index >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE folder_acl (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  principal_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  principal_group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
  path_prefix TEXT NOT NULL DEFAULT '/',
  effect permission_effect NOT NULL DEFAULT 'allow',
  access access_level NOT NULL,
  inherited BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (principal_user_id IS NOT NULL OR principal_group_id IS NOT NULL)
);

CREATE TABLE app_permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  app_id TEXT NOT NULL REFERENCES dock_apps(id) ON DELETE CASCADE,
  principal_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  principal_group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
  permission_key TEXT NOT NULL,
  effect permission_effect NOT NULL DEFAULT 'allow',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (principal_user_id IS NOT NULL OR principal_group_id IS NOT NULL)
);

CREATE TABLE agent_permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  agent_key TEXT NOT NULL,
  principal_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  principal_group_id UUID REFERENCES groups(id) ON DELETE CASCADE,
  tool_key TEXT NOT NULL,
  effect permission_effect NOT NULL DEFAULT 'allow',
  requires_confirmation BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (principal_user_id IS NOT NULL OR principal_group_id IS NOT NULL)
);

CREATE TABLE permission_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL,
  snapshot JSONB NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  event_type TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT,
  summary TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  ip_address INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rollback_operations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  snapshot_id UUID REFERENCES permission_snapshots(id) ON DELETE SET NULL,
  requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
  status job_state NOT NULL DEFAULT 'queued',
  plan JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(plan) = 'object'),
  result JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE risk_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  risk_level risk_level NOT NULL,
  title TEXT NOT NULL,
  detail TEXT NOT NULL,
  action_key TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  rollback_operation_id UUID REFERENCES rollback_operations(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE share_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  token_hash TEXT NOT NULL UNIQUE,
  path TEXT NOT NULL,
  access access_level NOT NULL DEFAULT 'read',
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE security_findings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  detail TEXT NOT NULL,
  risk_level risk_level NOT NULL,
  source TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT,
  status job_state NOT NULL DEFAULT 'queued',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ
);

CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  category TEXT NOT NULL,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_models (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider TEXT NOT NULL,
  model_key TEXT NOT NULL,
  display_name TEXT NOT NULL,
  runtime model_runtime NOT NULL,
  context_tokens INTEGER CHECK (context_tokens IS NULL OR context_tokens > 0),
  enabled BOOLEAN NOT NULL DEFAULT true,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (provider, model_key)
);

CREATE TABLE model_policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  runtime model_runtime NOT NULL,
  model_id UUID REFERENCES ai_models(id) ON DELETE SET NULL,
  scope JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(scope) = 'object'),
  enforce_local_for_private BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE privacy_policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  rules JSONB NOT NULL CHECK (jsonb_typeof(rules) = 'array'),
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  event_filter JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(event_filter) = 'object'),
  channels JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(channels) = 'array'),
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE system_backups (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  target_uri TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE file_nodes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  parent_id UUID REFERENCES file_nodes(id) ON DELETE CASCADE,
  name TEXT NOT NULL CHECK (name <> ''),
  node_kind file_node_kind NOT NULL,
  mime_type TEXT,
  size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
  path TEXT NOT NULL CHECK (path LIKE '/%'),
  permission_label TEXT NOT NULL DEFAULT '',
  ai_summary TEXT NOT NULL DEFAULT '',
  checksum_sha256 TEXT,
  modified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ,
  UNIQUE (space_id, path)
);

CREATE TABLE file_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID NOT NULL REFERENCES file_nodes(id) ON DELETE CASCADE,
  version_no INTEGER NOT NULL CHECK (version_no > 0),
  storage_uri TEXT NOT NULL,
  size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
  checksum_sha256 TEXT,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (file_id, version_no)
);

CREATE TABLE file_tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  color TEXT NOT NULL DEFAULT '#64748b' CHECK (color ~ '^#[0-9a-fA-F]{6}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE file_node_tags (
  file_id UUID NOT NULL REFERENCES file_nodes(id) ON DELETE CASCADE,
  tag_id UUID NOT NULL REFERENCES file_tags(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (file_id, tag_id)
);

CREATE TABLE file_favorites (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  file_id UUID NOT NULL REFERENCES file_nodes(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, file_id)
);

CREATE TABLE file_previews (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID NOT NULL REFERENCES file_nodes(id) ON DELETE CASCADE,
  preview_kind TEXT NOT NULL,
  preview_uri TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE recycle_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID NOT NULL REFERENCES file_nodes(id) ON DELETE CASCADE,
  original_path TEXT NOT NULL,
  deleted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  deleted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  purge_after TIMESTAMPTZ NOT NULL
);

CREATE TABLE index_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID REFERENCES file_nodes(id) ON DELETE CASCADE,
  job_type TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  error_message TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE document_chunks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID NOT NULL REFERENCES file_nodes(id) ON DELETE CASCADE,
  chunk_no INTEGER NOT NULL CHECK (chunk_no >= 0),
  content TEXT NOT NULL,
  token_count INTEGER CHECK (token_count IS NULL OR token_count >= 0),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (file_id, chunk_no)
);

CREATE TABLE semantic_embeddings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  chunk_id UUID NOT NULL REFERENCES document_chunks(id) ON DELETE CASCADE,
  model_id UUID REFERENCES ai_models(id) ON DELETE SET NULL,
  embedding vector(1536) NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (chunk_id, model_id)
);

CREATE TABLE entity_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID REFERENCES file_nodes(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL,
  entity_value TEXT NOT NULL,
  confidence NUMERIC(5,4) CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE knowledge_edges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_entity TEXT NOT NULL,
  target_entity TEXT NOT NULL,
  relation TEXT NOT NULL,
  confidence NUMERIC(5,4) CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (source_entity, target_entity, relation)
);

CREATE TABLE storage_pools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  pool_type TEXT NOT NULL,
  total_bytes BIGINT NOT NULL CHECK (total_bytes > 0),
  used_percent NUMERIC(5,2) NOT NULL CHECK (used_percent BETWEEN 0 AND 100),
  health storage_health NOT NULL DEFAULT 'unknown',
  temperature_c NUMERIC(5,2) CHECK (temperature_c IS NULL OR temperature_c BETWEEN -20 AND 120),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE disks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slot TEXT NOT NULL UNIQUE,
  serial_number TEXT UNIQUE,
  size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
  state disk_state NOT NULL DEFAULT 'unknown',
  temperature_c NUMERIC(5,2) CHECK (temperature_c IS NULL OR temperature_c BETWEEN -20 AND 120),
  model TEXT,
  firmware TEXT,
  pool_id UUID REFERENCES storage_pools(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE volumes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pool_id UUID NOT NULL REFERENCES storage_pools(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  mount_path TEXT NOT NULL UNIQUE CHECK (mount_path LIKE '/%'),
  size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
  used_bytes BIGINT NOT NULL DEFAULT 0 CHECK (used_bytes >= 0),
  filesystem TEXT NOT NULL DEFAULT 'btrfs',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (used_bytes <= size_bytes),
  UNIQUE (pool_id, name)
);

CREATE TABLE snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  volume_id UUID NOT NULL REFERENCES volumes(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'succeeded',
  size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ,
  UNIQUE (volume_id, name)
);

CREATE TABLE smart_reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  disk_id UUID NOT NULL REFERENCES disks(id) ON DELETE CASCADE,
  health storage_health NOT NULL,
  temperature_c NUMERIC(5,2),
  attributes JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(attributes) = 'object'),
  checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE storage_tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pool_id UUID REFERENCES storage_pools(id) ON DELETE CASCADE,
  task_type TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE metrics_samples (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  metric_key TEXT NOT NULL,
  label TEXT NOT NULL,
  value_text TEXT NOT NULL,
  numeric_value NUMERIC,
  unit TEXT,
  trend_text TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT 'devstub',
  observed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE system_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  severity TEXT NOT NULL CHECK (severity IN ('debug', 'info', 'warn', 'error')),
  source TEXT NOT NULL,
  message TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE alerts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  detail TEXT NOT NULL,
  tone alert_tone NOT NULL,
  source TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ
);

CREATE TABLE diagnostic_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  result JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result) = 'object'),
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE backup_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  source_uri TEXT NOT NULL,
  target_uri TEXT NOT NULL,
  schedule_cron TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE backup_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_id UUID NOT NULL REFERENCES backup_plans(id) ON DELETE CASCADE,
  status job_state NOT NULL DEFAULT 'queued',
  progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE backup_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  run_id UUID NOT NULL REFERENCES backup_runs(id) ON DELETE CASCADE,
  file_id UUID REFERENCES file_nodes(id) ON DELETE SET NULL,
  path TEXT NOT NULL,
  size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
  status job_state NOT NULL DEFAULT 'queued'
);

CREATE TABLE restore_points (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_id UUID NOT NULL REFERENCES backup_plans(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  target_uri TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (plan_id, name)
);

CREATE TABLE integrity_checks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  restore_point_id UUID NOT NULL REFERENCES restore_points(id) ON DELETE CASCADE,
  status job_state NOT NULL DEFAULT 'queued',
  checked_items INTEGER NOT NULL DEFAULT 0 CHECK (checked_items >= 0),
  failed_items INTEGER NOT NULL DEFAULT 0 CHECK (failed_items >= 0),
  result JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE download_tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  task_kind download_kind NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  target_path TEXT NOT NULL,
  size_bytes BIGINT CHECK (size_bytes IS NULL OR size_bytes >= 0),
  progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE download_sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID NOT NULL REFERENCES download_tasks(id) ON DELETE CASCADE,
  uri TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE TABLE rss_subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  feed_url TEXT NOT NULL,
  target_path TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE speed_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  max_download_bytes BIGINT CHECK (max_download_bytes IS NULL OR max_download_bytes > 0),
  max_upload_bytes BIGINT CHECK (max_upload_bytes IS NULL OR max_upload_bytes > 0),
  active_hours JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(active_hours) = 'object')
);

CREATE TABLE archive_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  source_path TEXT NOT NULL,
  target_path TEXT NOT NULL,
  matcher JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(matcher) = 'object'),
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE media_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id UUID REFERENCES file_nodes(id) ON DELETE SET NULL,
  media_type TEXT NOT NULL CHECK (media_type IN ('photo', 'video', 'audio')),
  title TEXT NOT NULL,
  taken_at TIMESTAMPTZ,
  duration_seconds INTEGER CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE albums (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  cover_media_id UUID REFERENCES media_items(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE album_items (
  album_id UUID NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
  media_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (album_id, media_id)
);

CREATE TABLE people (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  display_name TEXT NOT NULL UNIQUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE media_people (
  media_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
  person_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
  confidence NUMERIC(5,4) CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
  PRIMARY KEY (media_id, person_id)
);

CREATE TABLE places (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  latitude NUMERIC(9,6),
  longitude NUMERIC(9,6),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE TABLE media_places (
  media_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
  place_id UUID NOT NULL REFERENCES places(id) ON DELETE CASCADE,
  PRIMARY KEY (media_id, place_id)
);

CREATE TABLE memory_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  input_filter JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input_filter) = 'object'),
  output_uri TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE subtitle_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
  status job_state NOT NULL DEFAULT 'queued',
  language TEXT NOT NULL DEFAULT 'zh-CN',
  output_uri TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transcode_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  media_id UUID NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
  status job_state NOT NULL DEFAULT 'queued',
  profile TEXT NOT NULL,
  output_uri TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE compose_stacks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  compose_path TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE containers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  stack_id UUID REFERENCES compose_stacks(id) ON DELETE SET NULL,
  container_name TEXT NOT NULL UNIQUE,
  image TEXT NOT NULL,
  state container_state NOT NULL DEFAULT 'unknown',
  ports JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(ports) = 'array'),
  cpu_percent NUMERIC(5,2) CHECK (cpu_percent IS NULL OR cpu_percent BETWEEN 0 AND 100),
  memory_bytes BIGINT CHECK (memory_bytes IS NULL OR memory_bytes >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE container_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  container_id UUID REFERENCES containers(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  detail TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_catalog (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  app_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  image TEXT,
  default_config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(default_config) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_installs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_id UUID NOT NULL REFERENCES app_catalog(id) ON DELETE CASCADE,
  stack_id UUID REFERENCES compose_stacks(id) ON DELETE SET NULL,
  installed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  status job_state NOT NULL DEFAULT 'queued',
  config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE remote_channels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  channel_type TEXT NOT NULL CHECK (channel_type IN ('ddns', 'tunnel', 'vpn', 'share')),
  status job_state NOT NULL DEFAULT 'queued',
  config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tunnel_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  channel_id UUID NOT NULL REFERENCES remote_channels(id) ON DELETE CASCADE,
  client_name TEXT NOT NULL,
  ip_address INET,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at TIMESTAMPTZ
);

CREATE TABLE ddns_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  hostname TEXT NOT NULL UNIQUE,
  provider TEXT NOT NULL,
  last_ip INET,
  status job_state NOT NULL DEFAULT 'queued',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE login_alerts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  ip_address INET,
  user_agent TEXT,
  risk_level risk_level NOT NULL,
  detail TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  acknowledged_at TIMESTAMPTZ
);

CREATE TABLE bound_devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  device_name TEXT NOT NULL,
  device_type TEXT NOT NULL,
  public_key TEXT,
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, device_name)
);

CREATE TABLE ai_index_sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_key TEXT NOT NULL UNIQUE,
  space_id UUID REFERENCES spaces(id) ON DELETE CASCADE,
  path_prefix TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ai_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_type TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  input JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input) = 'object'),
  output JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(output) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE agent_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL,
  tools JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(tools) = 'array'),
  risk_level risk_level NOT NULL DEFAULT 'medium',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE agent_instances (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID NOT NULL REFERENCES agent_templates(id) ON DELETE CASCADE,
  owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  config JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
  status job_state NOT NULL DEFAULT 'queued',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflow_definitions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  agent_template_id UUID REFERENCES agent_templates(id) ON DELETE SET NULL,
  workflow_key TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  definition JSONB NOT NULL CHECK (jsonb_typeof(definition) = 'object'),
  version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflow_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_id UUID NOT NULL REFERENCES workflow_definitions(id) ON DELETE CASCADE,
  status job_state NOT NULL DEFAULT 'queued',
  input JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input) = 'object'),
  output JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(output) = 'object'),
  started_by UUID REFERENCES users(id) ON DELETE SET NULL,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE confirmations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_run_id UUID REFERENCES workflow_runs(id) ON DELETE CASCADE,
  requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
  status job_state NOT NULL DEFAULT 'queued',
  title TEXT NOT NULL,
  detail TEXT NOT NULL,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at TIMESTAMPTZ
);

CREATE TABLE tool_calls (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workflow_run_id UUID REFERENCES workflow_runs(id) ON DELETE CASCADE,
  confirmation_id UUID REFERENCES confirmations(id) ON DELETE SET NULL,
  tool_key TEXT NOT NULL,
  arguments JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(arguments) = 'object'),
  result JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result) = 'object'),
  status job_state NOT NULL DEFAULT 'queued',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE conversation_threads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  thread_id UUID NOT NULL REFERENCES conversation_threads(id) ON DELETE CASCADE,
  role message_role NOT NULL,
  content TEXT NOT NULL,
  model_id UUID REFERENCES ai_models(id) ON DELETE SET NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE assistant_actions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  thread_id UUID NOT NULL REFERENCES conversation_threads(id) ON DELETE CASCADE,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
  action_key TEXT NOT NULL,
  status job_state NOT NULL DEFAULT 'queued',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload) = 'object'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ
);

CREATE TABLE retrieval_citations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  file_id UUID REFERENCES file_nodes(id) ON DELETE SET NULL,
  chunk_id UUID REFERENCES document_chunks(id) ON DELETE SET NULL,
  label TEXT NOT NULL,
  excerpt TEXT NOT NULL DEFAULT '',
  score NUMERIC(6,5) CHECK (score IS NULL OR score BETWEEN 0 AND 1),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX users_status_idx ON users(status);
CREATE INDEX sessions_user_status_idx ON sessions(user_id, status);
CREATE INDEX folder_acl_space_path_idx ON folder_acl(space_id, path_prefix);
CREATE INDEX audit_events_created_idx ON audit_events(created_at DESC);
CREATE INDEX audit_events_target_idx ON audit_events(target_type, target_id);
CREATE INDEX share_links_space_idx ON share_links(space_id);
CREATE INDEX file_nodes_space_parent_idx ON file_nodes(space_id, parent_id);
CREATE INDEX file_nodes_space_path_idx ON file_nodes(space_id, path);
CREATE INDEX file_nodes_name_trgm_hint_idx ON file_nodes(lower(name));
CREATE INDEX document_chunks_file_idx ON document_chunks(file_id, chunk_no);
CREATE INDEX semantic_embeddings_embedding_hnsw_idx ON semantic_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX entity_links_value_idx ON entity_links(entity_type, entity_value);
CREATE INDEX storage_pools_health_idx ON storage_pools(health);
CREATE INDEX disks_pool_idx ON disks(pool_id);
CREATE INDEX metrics_samples_key_time_idx ON metrics_samples(metric_key, observed_at DESC);
CREATE INDEX alerts_status_time_idx ON alerts(status, created_at DESC);
CREATE INDEX system_logs_time_idx ON system_logs(created_at DESC);
CREATE INDEX download_tasks_status_idx ON download_tasks(status, created_at DESC);
CREATE INDEX media_items_taken_idx ON media_items(taken_at DESC);
CREATE INDEX containers_state_idx ON containers(state);
CREATE INDEX remote_channels_status_idx ON remote_channels(status);
CREATE INDEX workflow_runs_status_idx ON workflow_runs(status, started_at DESC);
CREATE INDEX messages_thread_time_idx ON messages(thread_id, created_at);
CREATE INDEX retrieval_citations_message_idx ON retrieval_citations(message_id);
