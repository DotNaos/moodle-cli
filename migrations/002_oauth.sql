create table if not exists oauth_clients (
  client_id text primary key,
  client_name text not null default '',
  redirect_uris jsonb not null default '[]'::jsonb,
  grant_types jsonb not null default '["authorization_code","refresh_token"]'::jsonb,
  response_types jsonb not null default '["code"]'::jsonb,
  scope text not null default '',
  created_at timestamptz not null default now()
);

create table if not exists oauth_authorization_codes (
  code_hash text primary key,
  client_id text not null references oauth_clients(client_id) on delete cascade,
  user_id uuid not null references users(id) on delete cascade,
  redirect_uri text not null,
  code_challenge text not null,
  code_challenge_method text not null,
  resource text not null,
  scope text not null default '',
  expires_at timestamptz not null,
  used_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists oauth_authorization_codes_client_idx
  on oauth_authorization_codes(client_id);

create table if not exists oauth_access_tokens (
  token_hash text primary key,
  user_id uuid not null references users(id) on delete cascade,
  client_id text not null references oauth_clients(client_id) on delete cascade,
  resource text not null,
  scope text not null default '',
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now(),
  last_used_at timestamptz
);

create table if not exists oauth_refresh_tokens (
  token_hash text primary key,
  user_id uuid not null references users(id) on delete cascade,
  client_id text not null references oauth_clients(client_id) on delete cascade,
  resource text not null,
  scope text not null default '',
  expires_at timestamptz not null,
  revoked_at timestamptz,
  created_at timestamptz not null default now(),
  last_used_at timestamptz
);

create index if not exists oauth_access_tokens_active_hash_idx
  on oauth_access_tokens(token_hash)
  where revoked_at is null;

create index if not exists oauth_refresh_tokens_active_hash_idx
  on oauth_refresh_tokens(token_hash)
  where revoked_at is null;
