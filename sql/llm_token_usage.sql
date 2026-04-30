-- LLM Token使用记录表
create sequence llm_token_usage_seq increment by 1 minvalue 1 no maxvalue start with 1;

create table "public"."llm_token_usage" (
    "id" bigint not null default nextval('llm_token_usage_seq'::regclass),
    "config_id" bigint not null references llm_config(id) on delete cascade,
    "model_entry_id" bigint not null references llm_model_entry(id) on delete cascade,
    "session_id" varchar(128) not null default '',
    "request_id" varchar(128) not null default '',
    "prompt_tokens" bigint not null default 0,
    "completion_tokens" bigint not null default 0,
    "total_tokens" bigint not null default 0,
    "model_name" varchar(256) not null default '',
    "action_type" varchar(64) not null default 'chat',
    "user_id" varchar(128) not null default '',
    "project_path" text not null default '',
    "created_at" timestamptz(6) not null default now()
);

-- 索引
create index idx_llm_token_usage_config_id on llm_token_usage(config_id);
create index idx_llm_token_usage_model_entry_id on llm_token_usage(model_entry_id);
create index idx_llm_token_usage_session_id on llm_token_usage(session_id);
create index idx_llm_token_usage_created_at on llm_token_usage(created_at);
create index idx_llm_token_usage_user_id on llm_token_usage(user_id);