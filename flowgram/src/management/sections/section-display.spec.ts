import { describe, expect, it } from 'vitest';

import { TaskStatus, TaskType } from '../../services/api-task';
import { ServiceStatus } from '../../services/api-service';
import {
  priorityTagColor,
  serviceStatusTagColor,
  taskStatusTagColor,
  taskTypeTagColor,
} from './section-display';

describe('section-display helpers', () => {
  it('maps service status to Semi tag colors', () => {
    expect(serviceStatusTagColor(ServiceStatus.RUNNING)).toBe('green');
    expect(serviceStatusTagColor(ServiceStatus.STOPPED)).toBe('red');
  });

  it('maps task status to Semi tag colors', () => {
    expect(taskStatusTagColor(TaskStatus.PENDING)).toBe('orange');
    expect(taskStatusTagColor(TaskStatus.PROCESSING)).toBe('blue');
    expect(taskStatusTagColor(TaskStatus.COMPLETED)).toBe('green');
    expect(taskStatusTagColor(TaskStatus.FAILED)).toBe('red');
  });

  it('maps task type to Semi tag colors', () => {
    expect(taskTypeTagColor(TaskType.BUG)).toBe('red');
    expect(taskTypeTagColor(TaskType.REQUIRE)).toBe('purple');
    expect(taskTypeTagColor(TaskType.FEATURE)).toBe('blue');
    expect(taskTypeTagColor(TaskType.OTHER)).toBe('grey');
  });

  it('maps priority to Semi tag colors', () => {
    expect(priorityTagColor(0)).toBe('red');
    expect(priorityTagColor(10)).toBe('red');
    expect(priorityTagColor(11)).toBe('orange');
    expect(priorityTagColor(30)).toBe('orange');
    expect(priorityTagColor(31)).toBe('grey');
  });
});
