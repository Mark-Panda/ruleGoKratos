create sequence mcp_config_seq increment by 1 minvalue 1 no maxvalue start with 1;

create table "public"."mcp_config" (
    "id" bigint not null default nextval('mcp_config_seq'::regclass),
    "name" varchar(128) not null,
    "server" varchar(128) not null,
    "endpoint" text not null,
    "headers_json" text not null default '{}',
    "enabled" boolean not null default true,
    "description" text not null default '',
    "created_at" timestamptz(6) not null default now(),
    "updated_at" timestamptz(6) not null default now(),
    constraint "mcp_config_pkey" primary key ("id")
);

create unique index mcp_config_name_unique_idx on mcp_config(name);
