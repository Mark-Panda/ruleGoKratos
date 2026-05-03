export type ScheduleType =
  | 'every_minutes'
  | 'every_hours'
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'advanced';

export type ScheduleConfig =
  | { type: 'every_minutes'; minutes: number }
  | { type: 'every_hours'; hours: number }
  | { type: 'daily'; hour: number; minute: number }
  | { type: 'weekly'; dayOfWeek: number; hour: number; minute: number }
  | { type: 'monthly'; dayOfMonth: number; hour: number; minute: number }
  | { type: 'advanced'; cronExpr: string };

export interface ScheduledTaskFormValues {
  name?: unknown;
  description?: unknown;
  ruleChainId?: unknown;
  scheduleType?: unknown;
  minutes?: unknown;
  hours?: unknown;
  hour?: unknown;
  minute?: unknown;
  dayOfWeek?: unknown;
  dayOfMonth?: unknown;
  cronExpr?: unknown;
  payloadTemplate?: unknown;
}

export interface ScheduledTaskLike {
  id?: unknown;
  disabled?: unknown;
  name: string;
  description?: string;
  ruleChainId: string;
  cronExpr: string;
  scheduleType: string;
  scheduleConfig: string;
  payloadTemplate?: string;
}

export interface ScheduledTaskPayloadLike {
  name: string;
  description: string;
  ruleChainId: string;
  cronExpr: string;
  scheduleType: string;
  scheduleConfig: string;
  payloadTemplate?: string;
}

export type NormalizedScheduledTaskRunStatus = 'success' | 'failed' | 'unknown';

function assertIntegerRange(field: string, value: number, min: number, max: number): void {
  if (!Number.isInteger(value) || value < min || value > max) {
    throw new Error(`${field} must be an integer between ${min} and ${max}`);
  }
}

function formatTime(hour: unknown, minute: unknown): string {
  const h = Number(hour);
  const m = Number(minute);
  if (!Number.isInteger(h) || !Number.isInteger(m)) return '--:--';
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

export function buildCronExpr(config: ScheduleConfig): string {
  switch (config.type) {
    case 'every_minutes':
      assertIntegerRange('minutes', config.minutes, 1, 59);
      return `*/${config.minutes} * * * *`;
    case 'every_hours':
      assertIntegerRange('hours', config.hours, 1, 23);
      return `0 */${config.hours} * * *`;
    case 'daily':
      assertIntegerRange('hour', config.hour, 0, 23);
      assertIntegerRange('minute', config.minute, 0, 59);
      return `${config.minute} ${config.hour} * * *`;
    case 'weekly':
      assertIntegerRange('dayOfWeek', config.dayOfWeek, 0, 6);
      assertIntegerRange('hour', config.hour, 0, 23);
      assertIntegerRange('minute', config.minute, 0, 59);
      return `${config.minute} ${config.hour} * * ${config.dayOfWeek}`;
    case 'monthly':
      assertIntegerRange('dayOfMonth', config.dayOfMonth, 1, 31);
      assertIntegerRange('hour', config.hour, 0, 23);
      assertIntegerRange('minute', config.minute, 0, 59);
      return `${config.minute} ${config.hour} ${config.dayOfMonth} * *`;
    case 'advanced':
      return config.cronExpr.trim();
    default: {
      const exhaustive: never = config;
      return exhaustive;
    }
  }
}

function toScheduleType(value: unknown): ScheduleType {
  const type = String(value ?? 'daily') as ScheduleType;
  if (
    type === 'every_minutes' ||
    type === 'every_hours' ||
    type === 'daily' ||
    type === 'weekly' ||
    type === 'monthly' ||
    type === 'advanced'
  ) {
    return type;
  }
  return 'daily';
}

function toNumber(value: unknown, fallback: number): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : fallback;
}

export function buildScheduleConfigFromFormValues(values: ScheduledTaskFormValues): ScheduleConfig {
  const scheduleType = toScheduleType(values.scheduleType);
  switch (scheduleType) {
    case 'every_minutes':
      return { type: scheduleType, minutes: toNumber(values.minutes, 15) };
    case 'every_hours':
      return { type: scheduleType, hours: toNumber(values.hours, 1) };
    case 'daily':
      return {
        type: scheduleType,
        hour: toNumber(values.hour, 9),
        minute: toNumber(values.minute, 0),
      };
    case 'weekly':
      return {
        type: scheduleType,
        dayOfWeek: toNumber(values.dayOfWeek, 1),
        hour: toNumber(values.hour, 9),
        minute: toNumber(values.minute, 0),
      };
    case 'monthly':
      return {
        type: scheduleType,
        dayOfMonth: toNumber(values.dayOfMonth, 1),
        hour: toNumber(values.hour, 9),
        minute: toNumber(values.minute, 0),
      };
    case 'advanced':
      return { type: scheduleType, cronExpr: String(values.cronExpr ?? '').trim() };
    default: {
      const exhaustive: never = scheduleType;
      return exhaustive;
    }
  }
}

export function buildScheduledTaskPayload(
  values: ScheduledTaskFormValues
): ScheduledTaskPayloadLike {
  const config = buildScheduleConfigFromFormValues(values);
  const payload: ScheduledTaskPayloadLike = {
    name: String(values.name ?? '').trim(),
    description: String(values.description ?? ''),
    ruleChainId: String(values.ruleChainId ?? '').trim(),
    scheduleType: config.type,
    scheduleConfig: JSON.stringify(config),
    cronExpr: buildCronExpr(config),
  };
  if (typeof values.payloadTemplate === 'string') {
    payload.payloadTemplate = values.payloadTemplate;
  }
  return payload;
}

export function parseScheduleConfig(scheduleConfig: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(scheduleConfig);
    if (parsed !== null && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
    return {};
  } catch {
    return {};
  }
}

export function normalizeScheduledTaskRunStatus(status: unknown): NormalizedScheduledTaskRunStatus {
  const normalized = String(status ?? '')
    .trim()
    .toUpperCase();
  if (
    normalized === '1' ||
    normalized === 'SUCCESS' ||
    normalized === 'SCHEDULED_TASK_RUN_STATUS_SUCCESS'
  ) {
    return 'success';
  }
  if (
    normalized === '2' ||
    normalized === 'FAILED' ||
    normalized === 'SCHEDULED_TASK_RUN_STATUS_FAILED'
  ) {
    return 'failed';
  }
  return 'unknown';
}

export function describeScheduledTaskRunStatus(status: unknown): string {
  const normalized = normalizeScheduledTaskRunStatus(status);
  if (normalized === 'success') return '触发成功';
  if (normalized === 'failed') return '触发失败';
  return '—';
}

export function getScheduledTaskFormInitValues(
  task?: ScheduledTaskLike | null
): ScheduledTaskFormValues {
  if (!task) {
    return {
      name: '',
      description: '',
      ruleChainId: '',
      scheduleType: 'daily',
      minutes: 15,
      hours: 1,
      hour: 9,
      minute: 0,
      dayOfWeek: 1,
      dayOfMonth: 1,
      cronExpr: '',
    };
  }

  const scheduleType = toScheduleType(task.scheduleType);
  const config = parseScheduleConfig(task.scheduleConfig);
  const cronExpr =
    scheduleType === 'advanced' ? String(config.cronExpr ?? task.cronExpr ?? '').trim() : '';
  return {
    name: task.name ?? '',
    description: task.description ?? '',
    ruleChainId: task.ruleChainId ?? '',
    scheduleType,
    minutes: toNumber(config.minutes, 15),
    hours: toNumber(config.hours, 1),
    hour: toNumber(config.hour, 9),
    minute: toNumber(config.minute, 0),
    dayOfWeek: toNumber(config.dayOfWeek, 1),
    dayOfMonth: toNumber(config.dayOfMonth, 1),
    cronExpr,
    payloadTemplate: task?.payloadTemplate ?? '',
  };
}

export function describeSchedule(
  scheduleType: ScheduleType,
  scheduleConfig: string,
  cronExpr: string
): string {
  const config = parseScheduleConfig(scheduleConfig);

  switch (scheduleType) {
    case 'every_minutes':
      return `每 ${String(config.minutes ?? '?')} 分钟执行`;
    case 'every_hours':
      return `每 ${String(config.hours ?? '?')} 小时执行`;
    case 'daily':
      return `每天 ${formatTime(config.hour, config.minute)} 执行`;
    case 'weekly': {
      const dayNames: Record<number, string> = {
        0: '日',
        1: '一',
        2: '二',
        3: '三',
        4: '四',
        5: '五',
        6: '六',
        7: '日',
      };
      const day = Number(config.dayOfWeek);
      return `每周${dayNames[day] ?? '?'} ${formatTime(config.hour, config.minute)} 执行`;
    }
    case 'monthly':
      return `每月 ${String(config.dayOfMonth ?? '?')} 日 ${formatTime(
        config.hour,
        config.minute
      )} 执行`;
    case 'advanced':
      return cronExpr.trim();
    default: {
      const exhaustive: never = scheduleType;
      return exhaustive;
    }
  }
}
