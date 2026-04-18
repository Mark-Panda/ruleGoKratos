create sequence llm_config_seq increment by 1 minvalue 1 no maxvalue start with 1;

create table "public"."llm_config" (
    "id" bigint not null default nextval('llm_config_seq'::regclass),
    "name" varchar(128) not null,
    "provider" varchar(64) not null default 'openai',
    "base_url" text not null default '',
    "api_key" text not null default '',
    "enabled" boolean not null default true,
    "description" text not null default '',
    "created_at" timestamptz(6) not null default now(),
    "updated_at" timestamptz(6) not null default now(),
    constraint "llm_config_pkey" primary key ("id")
);

create unique index llm_config_name_unique_idx on llm_config(name);
