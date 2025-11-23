-- 创建索引
create sequence rule_chain_seq increment by 1 minvalue 1 no maxvalue start with 1;

CREATE TABLE "public"."rule_chain" (
    "id" bigint NOT NULL DEFAULT nextval('rule_chain_seq'::regclass),
    "user_name" varchar(64) COLLATE "pg_catalog"."default",
    "root" boolean,
    "disabled" boolean,
    "debug_mode" boolean,
    "name" varchar(64) COLLATE "pg_catalog"."default",
    "rule_chain_id" varchar(64) COLLATE "pg_catalog"."default",
    "rule_version" int4 NOT NULL DEFAULT 0,
    "configuration" json DEFAULT null,
    "metadata" json DEFAULT null,
    "additional_info" json DEFAULT null,
    "created_at" timestamptz(6) NOT NULL DEFAULT now(),
    "updated_at" timestamptz(6) NOT NULL DEFAULT now(),
    "deleted_at" timestamptz(6),
    CONSTRAINT "rule_chain_pkey" PRIMARY KEY ("id")
);


COMMENT ON TABLE "public"."rule_chain" IS '规则配置新表';

CREATE UNIQUE INDEX rule_chain_rule_chain_id_version_unique_idx ON rule_chain(rule_chain_id,rule_version);

COMMENT ON COLUMN "public"."rule_chain"."id" IS '主键ID';
COMMENT ON COLUMN "public"."rule_chain"."user_name" IS '用户名';
COMMENT ON COLUMN "public"."rule_chain"."root" IS '是否根节点';
COMMENT ON COLUMN "public"."rule_chain"."disabled" IS '是否禁用';
COMMENT ON COLUMN "public"."rule_chain"."debug_mode" IS '是否调试';
COMMENT ON COLUMN "public"."rule_chain"."name" IS '规则名称';
COMMENT ON COLUMN "public"."rule_chain"."rule_chain_id" IS '规则ID';
COMMENT ON COLUMN "public"."rule_chain"."rule_version" IS '版本号';
COMMENT ON COLUMN "public"."rule_chain"."configuration" IS '规则配置信息';
COMMENT ON COLUMN "public"."rule_chain"."metadata" IS '规则配置信息';
COMMENT ON COLUMN "public"."rule_chain"."additional_info" IS '规则配置信息';
COMMENT ON COLUMN "public"."rule_chain"."created_at" IS '创建时间';
COMMENT ON COLUMN "public"."rule_chain"."updated_at" IS '更新时间';
COMMENT ON COLUMN "public"."rule_chain"."deleted_at" IS '删除时间';