import type { TagColor } from '@douyinfe/semi-ui/lib/es/tag';

import { ServiceStatus } from '../../services/api-service';
import { TaskStatus, TaskType } from '../../services/api-task';

export function serviceStatusTagColor(status: ServiceStatus): TagColor {
  switch (status) {
    case ServiceStatus.RUNNING:
      return 'green';
    case ServiceStatus.STOPPED:
    default:
      return 'red';
  }
}

export function taskStatusTagColor(status: TaskStatus): TagColor {
  switch (status) {
    case TaskStatus.PENDING:
      return 'orange';
    case TaskStatus.PROCESSING:
      return 'blue';
    case TaskStatus.COMPLETED:
      return 'green';
    case TaskStatus.FAILED:
    default:
      return 'red';
  }
}

export function taskTypeTagColor(type: TaskType): TagColor {
  switch (type) {
    case TaskType.BUG:
      return 'red';
    case TaskType.REQUIRE:
      return 'purple';
    case TaskType.FEATURE:
      return 'blue';
    case TaskType.OTHER:
    default:
      return 'grey';
  }
}

export function priorityTagColor(priority: number): TagColor {
  if (priority <= 10) return 'red';
  if (priority <= 30) return 'orange';
  return 'grey';
}
