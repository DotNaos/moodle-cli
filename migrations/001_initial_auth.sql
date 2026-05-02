create extension if not exists pgcrypto;

create table if not exists users (
  id uuid primary key default gen_random_uuid(),
  moodle_site_url text not null,
  moodle_user_id integer not null,
  display_name text not null default '',
  clerk_user_id text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (moodle_site_url, moodle_user_id),
  unique (clerk_user_id)
);

create table if not exists moodle_accounts (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  school_id text not null default '',
  site_url text not null,
  encrypted_mobile_session_json text not null,
  token_last_validated_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (user_id, site_url)
);

create table if not exists api_keys (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  name text not null,
  key_prefix text not null,
  key_hash text not null unique,
  scopes jsonb not null default '[]'::jsonb,
  last_used_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists api_keys_user_id_idx on api_keys(user_id);
create index if not exists api_keys_active_hash_idx on api_keys(key_hash) where revoked_at is null;

create table if not exists web_sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  session_hash text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create table if not exists calendar_subscriptions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  encrypted_url text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists pdf_view_states (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  course_id text not null,
  resource_id text not null,
  page integer not null,
  page_count integer not null,
  screenshot_ref text,
  updated_at timestamptz not null default now(),
  unique (user_id, course_id, resource_id)
);

create table if not exists audit_events (
  id uuid primary key default gen_random_uuid(),
  user_id uuid references users(id) on delete set null,
  event_type text not null,
  ip_hash text,
  created_at timestamptz not null default now()
);
