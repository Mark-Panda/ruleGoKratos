export type ManagedAgentOption = {
  value: number;
  label: string;
};

export type ManagedAgentSelection = {
  managedAgentId: number;
  needsPicker: boolean;
  shouldClearStoredId: boolean;
};

export function resolveManagedAgentSelection(
  storedManagedAgentId: number,
  options: ManagedAgentOption[] | null
): ManagedAgentSelection {
  if (storedManagedAgentId > 0 && options == null) {
    return {
      managedAgentId: storedManagedAgentId,
      needsPicker: false,
      shouldClearStoredId: false,
    };
  }

  const safeOptions = options ?? [];
  const firstManagedAgentId = safeOptions[0]?.value ?? 0;
  if (storedManagedAgentId > 0) {
    const exists = safeOptions.some((item) => item.value === storedManagedAgentId);
    if (exists) {
      return {
        managedAgentId: storedManagedAgentId,
        needsPicker: false,
        shouldClearStoredId: false,
      };
    }
    return {
      managedAgentId: firstManagedAgentId,
      needsPicker: safeOptions.length > 0,
      shouldClearStoredId: true,
    };
  }
  return {
    managedAgentId: firstManagedAgentId,
    needsPicker: safeOptions.length > 0,
    shouldClearStoredId: false,
  };
}

export function shouldTreatStatusErrorAsMissing(_error: unknown): boolean {
  return false;
}
