export type RuleChainSkillStatus = 'missing' | 'stale' | 'ready';

export type RuleChainSkillPresentation = {
  tagColor: 'grey' | 'orange' | 'green';
  tagLabel: string;
  actionLabel: string;
  actionable: boolean;
};

export function getRuleChainSkillPresentation(status: string): RuleChainSkillPresentation {
  switch (status) {
    case 'ready':
      return {
        tagColor: 'green',
        tagLabel: '已就绪',
        actionLabel: '更新技能',
        actionable: true,
      };
    case 'stale':
      return {
        tagColor: 'orange',
        tagLabel: '待更新',
        actionLabel: '更新技能',
        actionable: true,
      };
    case 'missing':
    default:
      return {
        tagColor: 'grey',
        tagLabel: '未创建',
        actionLabel: '创建技能',
        actionable: true,
      };
  }
}
