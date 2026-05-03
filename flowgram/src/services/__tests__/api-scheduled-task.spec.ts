import { beforeEach, describe, expect, it, vi } from 'vitest';

import { requestJSON } from '../http';
import {
  createScheduledTask,
  deleteScheduledTask,
  disableScheduledTask,
  enableScheduledTask,
  getScheduledTask,
  listScheduledTaskRuns,
  listScheduledTasks,
  updateScheduledTask,
} from '../api-scheduled-task';

vi.mock('../http', () => ({
  requestJSON: vi.fn(),
}));

describe('scheduled task api service', () => {
  beforeEach(() => {
    vi.mocked(requestJSON).mockReset();
  });

  it('lists scheduled tasks with query params', async () => {
    const reply = { tasks: [], total: 0 };
    const params = { name: 'daily', ruleChainId: 'rc-1', disabled: false, page: 2, pageSize: 20 };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(listScheduledTasks(params)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks', { method: 'GET', params });
  });

  it('gets scheduled task detail by id', async () => {
    const reply = {
      task: {
        id: 7,
        name: 'daily task',
        ruleChainId: 'rc-1',
        cronExpr: '0 9 * * *',
        scheduleType: 'daily',
        scheduleConfig: '{"hour":9,"minute":0}',
        disabled: false,
      },
    };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(getScheduledTask(7)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks/7', { method: 'GET' });
  });

  it('creates scheduled task with request body', async () => {
    const payload = {
      name: 'daily task',
      description: 'run daily',
      ruleChainId: 'rc-1',
      cronExpr: '0 9 * * *',
      scheduleType: 'daily',
      scheduleConfig: '{"hour":9,"minute":0}',
    };
    const reply = { task: { id: 7, ...payload, disabled: false } };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(createScheduledTask(payload)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks', { method: 'POST', body: payload });
  });

  it('updates scheduled task with request body', async () => {
    const payload = {
      name: 'updated task',
      description: 'updated description',
      ruleChainId: 'rc-1',
      cronExpr: '0 10 * * *',
      scheduleType: 'daily',
      scheduleConfig: '{"hour":10,"minute":0}',
    };
    const reply = { task: { id: 7, disabled: false, ...payload } };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(updateScheduledTask(7, payload)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks/7', {
      method: 'PUT',
      body: payload,
    });
  });

  it('deletes scheduled task by id', async () => {
    const reply = { success: true };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(deleteScheduledTask(7)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks/7', { method: 'DELETE' });
  });

  it('enables scheduled task by id', async () => {
    const reply = { task: { id: 7, disabled: false } };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(enableScheduledTask(7)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks/7/enable', {
      method: 'POST',
      body: {},
    });
  });

  it('disables scheduled task by id', async () => {
    const reply = { task: { id: 7, disabled: true } };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(disableScheduledTask(7)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks/7/disable', {
      method: 'POST',
      body: {},
    });
  });

  it('lists scheduled task runs with query params', async () => {
    const reply = { runs: [], total: 0 };
    const params = { page: 3, pageSize: 30 };
    vi.mocked(requestJSON).mockResolvedValue(reply);

    await expect(listScheduledTaskRuns(7, params)).resolves.toBe(reply);

    expect(requestJSON).toHaveBeenCalledWith('/scheduled-tasks/7/runs', { method: 'GET', params });
  });
});
