import { describe, expect, it } from 'vitest';

import { getRuleChainSkillPresentation } from '../rule-chain-skill-ui';

describe('getRuleChainSkillPresentation', () => {
  it('returns create action for missing skill', () => {
    expect(getRuleChainSkillPresentation('missing')).toMatchObject({
      tagLabel: '未创建',
      actionLabel: '创建技能',
      actionable: true,
    });
  });

  it('returns update action for stale skill', () => {
    expect(getRuleChainSkillPresentation('stale')).toMatchObject({
      tagLabel: '待更新',
      actionLabel: '更新技能',
      actionable: true,
    });
  });

  it('keeps manual rebuild action for ready skill', () => {
    expect(getRuleChainSkillPresentation('ready')).toMatchObject({
      tagLabel: '已就绪',
      actionLabel: '更新技能',
      actionable: true,
    });
  });
});
