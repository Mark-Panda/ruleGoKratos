/**
 * 规则链 configuration 下的 flowgram 扩展块：入参/出参说明、编辑器等，持久化到 DB。
 * 读取时若仅有历史键 devpilot，则兼容解析；写入统一使用 configuration.flowgram。
 */

import { emptyRuleChainParamsJson } from './rule-chain-request-params';

export const FLOWGRAM_DSL_SCHEMA_VERSION = 1;

/** 当前写入的 configuration 键名 */
export const RULE_CHAIN_FLOWGRAM_CONFIG_KEY = 'flowgram' as const;
/** 历史数据兼容（仅解析，新写入不再使用） */
const LEGACY_RULE_CHAIN_CONFIG_KEY = 'devpilot' as const;

function asString(v: unknown): string {
  return typeof v === 'string' ? v : '';
}

function parseParamsArrayFromIO(raw: unknown): string {
  if (raw == null) return emptyRuleChainParamsJson();
  if (typeof raw === 'string') {
    const t = raw.trim();
    if (!t) return emptyRuleChainParamsJson();
    try {
      JSON.parse(t);
      return t;
    } catch {
      return emptyRuleChainParamsJson();
    }
  }
  if (Array.isArray(raw)) {
    try {
      return JSON.stringify(raw, null, 2);
    } catch {
      return emptyRuleChainParamsJson();
    }
  }
  return emptyRuleChainParamsJson();
}

export type RuleChainFlowgramParsed = {
  description: string;
  /** 异步/同步触发使用的消息类型提示，可选；对应 DSL 键 entry_msg_type */
  entryMsgType: string;
  requestMetadataParamsJson: string;
  requestMessageBodyParamsJson: string;
  responseMessageBodyParamsJson: string;
  editorJson: string;
  skillDirName: string;
};

function parseFlowgramBlock(block: Record<string, unknown>): RuleChainFlowgramParsed {
  const io =
    block.io && typeof block.io === 'object' && !Array.isArray(block.io)
      ? (block.io as Record<string, unknown>)
      : {};
  const editor =
    block.editor && typeof block.editor === 'object' && !Array.isArray(block.editor)
      ? (block.editor as Record<string, unknown>)
      : {};
  const skill =
    block.skill && typeof block.skill === 'object' && !Array.isArray(block.skill)
      ? (block.skill as Record<string, unknown>)
      : {};
  const scratch = asString(editor.scratch_json);
  const entryRaw =
    (block as Record<string, unknown>).entry_msg_type ??
    (block as Record<string, unknown>).entryMsgType;
  return {
    description: asString(block.description),
    entryMsgType: typeof entryRaw === 'string' ? entryRaw.trim() : '',
    requestMetadataParamsJson: parseParamsArrayFromIO(io.request_metadata_params),
    requestMessageBodyParamsJson: parseParamsArrayFromIO(io.request_message_body_params),
    responseMessageBodyParamsJson: parseParamsArrayFromIO(io.response_message_body_params),
    editorJson: scratch,
    skillDirName: asString(skill.dir_name),
  };
}

/** 从 ruleChain.configuration 解析 flowgram 块（优先 flowgram，其次兼容旧键 devpilot） */
export function parseRuleChainFlowgramFromConfiguration(
  configuration: unknown
): RuleChainFlowgramParsed {
  if (!configuration || typeof configuration !== 'object' || Array.isArray(configuration)) {
    return emptyParsed();
  }
  const cfg = configuration as Record<string, unknown>;
  let block: Record<string, unknown> | null = null;
  const fg = cfg[RULE_CHAIN_FLOWGRAM_CONFIG_KEY];
  if (fg && typeof fg === 'object' && !Array.isArray(fg)) {
    block = fg as Record<string, unknown>;
  } else {
    const leg = cfg[LEGACY_RULE_CHAIN_CONFIG_KEY];
    if (leg && typeof leg === 'object' && !Array.isArray(leg)) {
      block = leg as Record<string, unknown>;
    }
  }
  if (!block) return emptyParsed();
  return parseFlowgramBlock(block);
}

function emptyParsed(): RuleChainFlowgramParsed {
  return {
    description: '',
    entryMsgType: '',
    requestMetadataParamsJson: emptyRuleChainParamsJson(),
    requestMessageBodyParamsJson: emptyRuleChainParamsJson(),
    responseMessageBodyParamsJson: emptyRuleChainParamsJson(),
    editorJson: '',
    skillDirName: '',
  };
}

function paramsJsonStringToArray(s: string): unknown[] {
  const t = s?.trim() || '[]';
  try {
    const v = JSON.parse(t);
    return Array.isArray(v) ? v : [];
  } catch {
    return [];
  }
}

export type RuleChainFlowgramIoArrays = {
  request_metadata_params: unknown[];
  request_message_body_params: unknown[];
  response_message_body_params: unknown[];
};

export function paramsJsonStringsToIOArrays(
  requestMetadataParamsJson: string,
  requestMessageBodyParamsJson: string,
  responseMessageBodyParamsJson: string
): RuleChainFlowgramIoArrays {
  return {
    request_metadata_params: paramsJsonStringToArray(requestMetadataParamsJson),
    request_message_body_params: paramsJsonStringToArray(requestMessageBodyParamsJson),
    response_message_body_params: paramsJsonStringToArray(responseMessageBodyParamsJson),
  };
}

export type BuildRuleChainFlowgramConfigurationInput = {
  description: string;
  requestMetadataParamsJson: string;
  requestMessageBodyParamsJson: string;
  responseMessageBodyParamsJson: string;
  editorScratchJson: string;
  skillDirName?: string;
};

/** 合并写入 ruleChain.configuration.flowgram，并移除历史 devpilot 键 */
export function buildRuleChainConfigurationWithFlowgram(
  existingConfiguration: Record<string, unknown> | undefined,
  input: BuildRuleChainFlowgramConfigurationInput
): Record<string, unknown> {
  const base =
    existingConfiguration &&
    typeof existingConfiguration === 'object' &&
    !Array.isArray(existingConfiguration)
      ? { ...existingConfiguration }
      : {};
  delete base[LEGACY_RULE_CHAIN_CONFIG_KEY];
  const io = paramsJsonStringsToIOArrays(
    input.requestMetadataParamsJson,
    input.requestMessageBodyParamsJson,
    input.responseMessageBodyParamsJson
  );
  base[RULE_CHAIN_FLOWGRAM_CONFIG_KEY] = {
    schema_version: FLOWGRAM_DSL_SCHEMA_VERSION,
    description: String(input.description ?? '').trim(),
    io,
    editor: {
      scratch_json: String(input.editorScratchJson ?? '').trim(),
    },
    skill: {
      dir_name: String(input.skillDirName ?? '').trim(),
    },
  };
  return base;
}
