create sequence llm_model_entry_seq increment by 1 minvalue 1 no maxvalue start with 1;

create table "public"."llm_model_entry" (
    "id" bigint not null default nextval('llm_model_entry_seq'::regclass),
    "config_id" bigint not null references llm_config(id) on delete cascade,
    "model_name" varchar(256) not null,
    "description" text not null default '',
    "enabled" boolean not null default true,
    "created_at" timestamptz(6) not null default now(),
    "updated_at" timestamptz(6) not null default now(),
    constraint "llm_model_entry_pkey" primary key ("id")
);

create unique index llm_model_entry_config_model_uidx on llm_model_entry(config_id, model_name);
