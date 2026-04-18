import { describe, expect, it } from 'vitest';

import { buildCanvasNodeMapsFromRuleDetail, summarizeRuleMsgLike } from '../run-log-display';

describe('summarizeRuleMsgLike', () => {
  it('summarizes RuleMsg-like object with type and data string', () => {
    const s = summarizeRuleMsgLike({
      type: 'CHAIN',
      data: '{"a":1}',
      id: 'x',
      ts: 1,
      dataType: 'JSON',
    });
    expect(s).toContain('[CHAIN]');
    expect(s).toContain('"a":1');
  });

  it('returns dash for empty object payload', () => {
    expect(summarizeRuleMsgLike(null)).toBe('—');
  });
});

describe('buildCanvasNodeMapsFromRuleDetail', () => {
  it('maps flowgramUI node id to data.title and type', () => {
    const { labels, types } = buildCanvasNodeMapsFromRuleDetail({
      metadata: {
        flowgramUI: {
          nodes: [
            { id: 'n1', type: 'start', data: { title: '开始' } },
            { id: 'n2', type: 'log', data: { title: '日志' } },
          ],
        },
        nodes: [{ id: 'n1', type: 'start', name: 'DslName' }],
      },
    });
    expect(labels.get('n1')).toBe('开始');
    expect(labels.get('n2')).toBe('日志');
    expect(types.get('n1')).toBe('start');
  });
});
