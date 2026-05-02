create table if not exists moodle_app_users (
  user_id text primary key,
  api_key_hash text not null unique,
  mobile_session_json jsonb not null,
  calendar_url text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists moodle_app_users_api_key_hash_idx
  on moodle_app_users (api_key_hash);
