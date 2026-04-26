import { describe, expect, it } from 'vitest';

import {
  buildCronExpr,
  buildScheduledTaskPayload,
  describeScheduledTaskRunStatus,
  describeSchedule,
  getScheduledTaskFormInitValues,
  normalizeScheduledTaskRunStatus,
  parseScheduleConfig,
} from './scheduled-task-cron';

describe('buildCronExpr', () => {
  it('builds cron for every N minutes', () => {
    expect(buildCronExpr({ type: 'every_minutes', minutes: 15 })).toBe('*/15 * * * *');
  });

  it('builds cron for every N hours', () => {
    expect(buildCronExpr({ type: 'every_hours', hours: 6 })).toBe('0 */6 * * *');
  });

  it('builds cron for daily schedule', () => {
    expect(buildCronExpr({ type: 'daily', hour: 9, minute: 30 })).toBe('30 9 * * *');
  });

  it('builds cron for weekly schedule', () => {
    expect(buildCronExpr({ type: 'weekly', dayOfWeek: 1, hour: 8, minute: 5 })).toBe('5 8 * * 1');
  });

  it('builds cron for monthly schedule', () => {
    expect(buildCronExpr({ type: 'monthly', dayOfMonth: 20, hour: 23, minute: 45 })).toBe('45 23 20 * *');
  });

  it('uses advanced cron directly', () => {
    expect(buildCronExpr({ type: 'advanced', cronExpr: '0 2 * * *' })).toBe('0 2 * * *');
  });

  it('trims advanced cron', () => {
    expect(buildCronExpr({ type: 'advanced', cronExpr: '  0 2 * * *  ' })).toBe('0 2 * * *');
  });

  it('throws when minutes is invalid', () => {
    expect(() => buildCronExpr({ type: 'every_minutes', minutes: 0 })).toThrow(/minutes/);
  });

  it('throws when hours is invalid', () => {
    expect(() => buildCronExpr({ type: 'every_hours', hours: 24 })).toThrow(/hours/);
  });

  it('throws when hour is invalid', () => {
    expect(() => buildCronExpr({ type: 'daily', hour: 24, minute: 0 })).toThrow(/hour/);
  });

  it('throws when minute is invalid', () => {
    expect(() => buildCronExpr({ type: 'daily', hour: 0, minute: 60 })).toThrow(/minute/);
  });

  it('throws when dayOfWeek is invalid', () => {
    expect(() => buildCronExpr({ type: 'weekly', dayOfWeek: 8, hour: 0, minute: 0 })).toThrow(/dayOfWeek/);
  });

  it('throws when dayOfWeek is 7', () => {
    expect(() => buildCronExpr({ type: 'weekly', dayOfWeek: 7, hour: 0, minute: 0 })).toThrow(/dayOfWeek/);
  });

  it('throws when dayOfMonth is invalid', () => {
    expect(() => buildCronExpr({ type: 'monthly', dayOfMonth: 32, hour: 0, minute: 0 })).toThrow(/dayOfMonth/);
  });
});

describe('parseScheduleConfig', () => {
  it('returns parsed JSON object', () => {
    expect(parseScheduleConfig('{"minutes":15}')).toEqual({ minutes: 15 });
  });

  it('returns empty object for invalid JSON', () => {
    expect(parseScheduleConfig('{invalid json')).toEqual({});
  });
});

describe('describeSchedule', () => {
  it('describes every N minutes schedule in Chinese', () => {
    expect(describeSchedule('every_minutes', '{"minutes":15}', '*/15 * * * *')).toBe('每 15 分钟执行');
  });

  it('describes every N hours schedule in Chinese', () => {
    expect(describeSchedule('every_hours', '{"hours":6}', '0 */6 * * *')).toBe('每 6 小时执行');
  });

  it('describes daily schedule in Chinese', () => {
    expect(describeSchedule('daily', '{"hour":9,"minute":5}', '5 9 * * *')).toBe('每天 09:05 执行');
  });

  it('describes weekly schedule in Chinese', () => {
    expect(describeSchedule('weekly', '{"dayOfWeek":1,"hour":8,"minute":5}', '5 8 * * 1')).toBe('每周一 08:05 执行');
  });

  it('describes monthly schedule in Chinese', () => {
    expect(describeSchedule('monthly', '{"dayOfMonth":20,"hour":23,"minute":45}', '45 23 20 * *')).toBe('每月 20 日 23:45 执行');
  });

  it('returns cron expression for advanced schedule', () => {
    expect(describeSchedule('advanced', '{"cronExpr":"0 2 * * *"}', '0 2 * * *')).toBe('0 2 * * *');
  });
});

describe('buildScheduledTaskPayload', () => {
  it('builds payload and schedule config for simple schedule form values', () => {
    expect(
      buildScheduledTaskPayload({
        name: '  daily job  ',
        description: 'run every morning',
        ruleChainId: ' rc-1 ',
        scheduleType: 'daily',
        hour: 9,
        minute: 30,
      })
    ).toEqual({
      name: 'daily job',
      description: 'run every morning',
      ruleChainId: 'rc-1',
      scheduleType: 'daily',
      scheduleConfig: '{"type":"daily","hour":9,"minute":30}',
      cronExpr: '30 9 * * *',
    });
  });

  it('builds payload for advanced cron form values', () => {
    expect(
      buildScheduledTaskPayload({
        name: 'nightly',
        ruleChainId: 'rc-2',
        scheduleType: 'advanced',
        cronExpr: ' 0 2 * * * ',
      })
    ).toMatchObject({
      name: 'nightly',
      description: '',
      ruleChainId: 'rc-2',
      scheduleType: 'advanced',
      scheduleConfig: '{"type":"advanced","cronExpr":"0 2 * * *"}',
      cronExpr: '0 2 * * *',
    });
  });
});

describe('getScheduledTaskFormInitValues', () => {
  it('uses schedule type and config when editing a preset schedule task', () => {
    expect(
      getScheduledTaskFormInitValues({
        id: 1,
        name: 'weekly',
        description: 'every monday',
        ruleChainId: 'rc-1',
        scheduleType: 'weekly',
        scheduleConfig: '{"type":"weekly","dayOfWeek":1,"hour":8,"minute":5}',
        cronExpr: '5 8 * * 1',
        disabled: false,
      })
    ).toMatchObject({
      name: 'weekly',
      description: 'every monday',
      ruleChainId: 'rc-1',
      scheduleType: 'weekly',
      dayOfWeek: 1,
      hour: 8,
      minute: 5,
      cronExpr: '',
    });
  });

  it('uses cron expression for advanced schedule task editing', () => {
    expect(
      getScheduledTaskFormInitValues({
        id: 2,
        name: 'custom',
        ruleChainId: 'rc-2',
        scheduleType: 'advanced',
        scheduleConfig: '{"type":"advanced","cronExpr":"15 3 * * *"}',
        cronExpr: '15 3 * * *',
        disabled: true,
      })
    ).toMatchObject({
      name: 'custom',
      ruleChainId: 'rc-2',
      scheduleType: 'advanced',
      cronExpr: '15 3 * * *',
    });
  });
});

describe('normalizeScheduledTaskRunStatus', () => {
  it('normalizes success statuses from number, short string, and protobuf enum string', () => {
    expect(normalizeScheduledTaskRunStatus(1)).toBe('success');
    expect(normalizeScheduledTaskRunStatus('SUCCESS')).toBe('success');
    expect(normalizeScheduledTaskRunStatus('SCHEDULED_TASK_RUN_STATUS_SUCCESS')).toBe('success');
  });

  it('normalizes failed statuses from number, short string, and protobuf enum string', () => {
    expect(normalizeScheduledTaskRunStatus(2)).toBe('failed');
    expect(normalizeScheduledTaskRunStatus('FAILED')).toBe('failed');
    expect(normalizeScheduledTaskRunStatus('SCHEDULED_TASK_RUN_STATUS_FAILED')).toBe('failed');
  });

  it('keeps unknown statuses distinguishable', () => {
    expect(normalizeScheduledTaskRunStatus('SCHEDULED_TASK_RUN_STATUS_UNSPECIFIED')).toBe('unknown');
    expect(normalizeScheduledTaskRunStatus(undefined)).toBe('unknown');
  });
});

describe('describeScheduledTaskRunStatus', () => {
  it('describes run status as trigger result', () => {
    expect(describeScheduledTaskRunStatus(1)).toBe('触发成功');
    expect(describeScheduledTaskRunStatus('FAILED')).toBe('触发失败');
    expect(describeScheduledTaskRunStatus(undefined)).toBe('—');
  });
});
