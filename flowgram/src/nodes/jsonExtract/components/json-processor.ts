/**
 * JSON 提取与纠错组件
 * 从文本中提取 JSON 并做格式纠错与补全
 */

export interface JsonExtractResult {
  success: boolean;
  result?: string;
  extractedJson?: unknown;
  error?: string;
}

/**
 * 从 markdown 文本中提取 JSON
 */
export function extractJsonFromMarkdown(text: string): string | null {
  // 尝试提取 ```json ... ``` 代码块
  const jsonBlockMatch = text.match(/```(?:json)?\s*([\s\S]*?)```/i);
  if (jsonBlockMatch) {
    return jsonBlockMatch[1].trim();
  }

  // 尝试提取 ``` ... ``` 代码块
  const codeBlockMatch = text.match(/```\s*([\s\S]*?)```/);
  if (codeBlockMatch) {
    return codeBlockMatch[1].trim();
  }

  // 直接尝试解析整个文本
  return null;
}

/**
 * 修复常见 JSON 格式错误
 */
export function fixJsonFormat(jsonStr: string): string {
  let fixed = jsonStr;

  // 移除行内注释
  fixed = fixed.replace(/\/\/.*$/gm, '');

  // 修复单引号为双引号（排除已经在双引号内的）
  fixed = fixed.replace(/'/g, '"');

  // 移除尾部逗号
  fixed = fixed.replace(/,(\s*[}\]])/g, '$1');

  // 修复缺失的引号（key 未加引号的情况）
  fixed = fixed.replace(/([{,]\s*)([a-zA-Z_][a-zA-Z0-9_]*)\s*:/g, '$1"$2":');

  // 处理 undefined（转换为 null）
  fixed = fixed.replace(/:\s*undefined/g, ': null');

  // 处理单引号包裹的字符串内部的双引号问题
  // 这是一个很复杂的场景，暂时不处理

  return fixed;
}

/**
 * 智能补全不完整的 JSON
 */
export function completeJson(jsonStr: string): string {
  let fixed = jsonStr.trim();

  // 检查是否是 JSON 数组片段
  if (fixed.startsWith('[') && !fixed.endsWith(']')) {
    // 找到匹配的 ]
    let depth = 0;
    let endIndex = -1;
    for (let i = 0; i < fixed.length; i++) {
      if (fixed[i] === '[') depth++;
      if (fixed[i] === ']') {
        depth--;
        if (depth === 0) {
          endIndex = i;
          break;
        }
      }
    }
    if (endIndex !== -1) {
      fixed = fixed.substring(0, endIndex + 1);
    }
  }

  // 检查是否是 JSON 对象片段
  if (fixed.startsWith('{') && !fixed.endsWith('}')) {
    // 简单处理：找到最后一个完整的键值对
    // 移除可能不完整的部分
    const lastComplete = fixed.lastIndexOf(',');
    if (lastComplete !== -1) {
      fixed = fixed.substring(0, lastComplete) + '}';
    }
  }

  return fixed;
}

/**
 * 解析 JSON，尝试多种修复策略
 */
export function parseJsonWithFixes(text: string, mode: string = 'auto'): JsonExtractResult {
  const trimText = text.trim();

  // 1. 尝试直接解析
  try {
    const parsed = JSON.parse(trimText);
    return {
      success: true,
      result: JSON.stringify(parsed, null, 2),
      extractedJson: parsed,
    };
  } catch (e) {
    // continue to next step
  }

  // 2. 根据模式提取 JSON
  let jsonStr = trimText;
  if (mode === 'md' || mode === 'auto') {
    const extracted = extractJsonFromMarkdown(trimText);
    if (extracted) {
      jsonStr = extracted;
    }
  }

  // 3. 尝试修复格式后解析
  const strategies = [
    () => extractJsonFromMarkdown(jsonStr) ?? jsonStr,
    () => fixJsonFormat(jsonStr),
    () => completeJson(fixJsonFormat(jsonStr)),
    () => fixJsonFormat(completeJson(jsonStr)),
    () => completeJson(jsonStr),
    () => fixJsonFormat(completeJson(jsonStr)),
  ];

  for (let i = 0; i < strategies.length; i++) {
    try {
      const fixed = strategies[i]();
      const parsed = JSON.parse(fixed);
      return {
        success: true,
        result: JSON.stringify(parsed, null, 2),
        extractedJson: parsed,
      };
    } catch (e) {
      // try next strategy
    }
  }

  // 4. 所有策略都失败
  return {
    success: false,
    error: `无法解析 JSON，请检查输入格式是否正确。最后尝试的格式: ${jsonStr.substring(0, 200)}...`,
  };
}

/**
 * 主处理函数
 */
export function processJsonExtract(source: string, mode: string = 'auto'): JsonExtractResult {
  if (!source || source.trim() === '') {
    return {
      success: false,
      error: '输入文本不能为空',
    };
  }

  return parseJsonWithFixes(source, mode);
}
