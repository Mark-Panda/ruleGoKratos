-- 运行时由 dao.Init → migrateMcpConfigTable 执行 AutoMigrate；此文件便于 docker-entrypoint-initdb / 手工对照
create sequence mcp_config_seq increment by 1 minvalue 1 no maxvalue start with 1;

create table "public"."mcp_config" (
    "id" bigint not null default nextval('mcp_config_seq'::regclass),
    "name" varchar(128) not null,
    "server" varchar(128) not null,
    "endpoint" text not null default '',
    "headers_json" text not null default '{}',
    "transport" varchar(16) not null default 'http',
    "stdio_command" text not null default '',
    "stdio_args_json" text not null default '[]',
    "stdio_env_json" text not null default '{}',
    "enabled" boolean not null default true,
    "description" text not null default '',
    "created_at" timestamptz(6) not null default now(),
    "updated_at" timestamptz(6) not null default now(),
    constraint "mcp_config_pkey" primary key ("id")
);

create unique index mcp_config_name_unique_idx on mcp_config(name);
