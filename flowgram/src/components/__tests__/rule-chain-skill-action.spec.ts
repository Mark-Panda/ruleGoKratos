import { describe, expect, it } from 'vitest';

import {
  resolveManagedAgentSelection,
  shouldTreatStatusErrorAsMissing,
} from '../../utils/rule-chain-skill-action-helpers';

describe('resolveManagedAgentSelection', () => {
  it('accepts cached managed agent when still enabled', () => {
    const result = resolveManagedAgentSelection(12, [
      { value: 12, label: 'Agent A' },
      { value: 18, label: 'Agent B' },
    ]);

    expect(result).toEqual({
      managedAgentId: 12,
      needsPicker: false,
      shouldClearStoredId: false,
    });
  });

  it('clears invalid cached managed agent and opens picker', () => {
    const result = resolveManagedAgentSelection(99, [
      { value: 12, label: 'Agent A' },
      { value: 18, label: 'Agent B' },
    ]);

    expect(result).toEqual({
      managedAgentId: 12,
      needsPicker: true,
      shouldClearStoredId: true,
    });
  });

  it('keeps stored managed agent when options are temporarily unavailable', () => {
    const result = resolveManagedAgentSelection(12, null);

    expect(result).toEqual({
      managedAgentId: 12,
      needsPicker: false,
      shouldClearStoredId: false,
    });
  });

  it('does not treat generic 404 as skill missing', () => {
    expect(shouldTreatStatusErrorAsMissing(new Error('HTTP 404: {"error":"not found"}'))).toBe(
      false
    );
  });
});
