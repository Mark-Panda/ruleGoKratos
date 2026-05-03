import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { Button, Modal, Select, Tag, Toast, Typography } from '@douyinfe/semi-ui';

import { getRuleChainSkillPresentation } from '../utils/rule-chain-skill-ui';
import {
  resolveManagedAgentSelection,
  shouldTreatStatusErrorAsMissing,
  type ManagedAgentOption,
} from '../utils/rule-chain-skill-action-helpers';
import { loadStoredManagedAgentId, saveStoredManagedAgentId } from '../utils/managed-agent-storage';
import { listEnabledManagedAgents } from '../services/api-managed-agents';
import {
  generateRuleChainSkill,
  getRuleChainSkillStatus,
  type RuleChainSkillStatusReply,
} from '../services/api-agent';

export interface RuleChainSkillActionProps {
  ruleChainId: string;
  isRoot?: boolean;
  size?: 'small' | 'default';
  showStatusText?: boolean;
  refreshToken?: number;
  onStatusLoaded?: (status: RuleChainSkillStatusReply) => void;
  onGenerated?: () => void;
}

const missingStatus: RuleChainSkillStatusReply = { status: 'missing' };

export const RuleChainSkillAction: React.FC<RuleChainSkillActionProps> = ({
  ruleChainId,
  isRoot = true,
  size = 'default',
  showStatusText = false,
  refreshToken = 0,
  onStatusLoaded,
  onGenerated,
}) => {
  const [status, setStatus] = useState<RuleChainSkillStatusReply>(missingStatus);
  const [statusError, setStatusError] = useState('');
  const [hasLoadedStatus, setHasLoadedStatus] = useState(false);
  const [loadingStatus, setLoadingStatus] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [pickerVisible, setPickerVisible] = useState(false);
  const [pickerLoading, setPickerLoading] = useState(false);
  const [agentOptions, setAgentOptions] = useState<ManagedAgentOption[]>([]);
  const [selectedManagedAgentId, setSelectedManagedAgentId] = useState<number>(
    () => loadStoredManagedAgentId() || 0
  );

  const refreshStatus = useCallback(async () => {
    if (!ruleChainId || !isRoot) return;
    setLoadingStatus(true);
    setStatusError('');
    try {
      const next = await getRuleChainSkillStatus(ruleChainId);
      setStatus(next);
      setHasLoadedStatus(true);
      onStatusLoaded?.(next);
    } catch (err) {
      if (shouldTreatStatusErrorAsMissing(err)) {
        setStatus(missingStatus);
        setHasLoadedStatus(true);
        onStatusLoaded?.(missingStatus);
      } else {
        const message = String((err as Error)?.message ?? err);
        setStatusError(message);
        Toast.error({ content: `加载技能状态失败：${message}` });
      }
    } finally {
      setLoadingStatus(false);
    }
  }, [isRoot, onStatusLoaded, ruleChainId]);

  useEffect(() => {
    refreshStatus();
  }, [refreshStatus, refreshToken]);

  const presentation = useMemo(
    () => getRuleChainSkillPresentation(status.status || 'missing'),
    [status.status]
  );

  const loadManagedAgentOptions = useCallback(async () => {
    setPickerLoading(true);
    try {
      const rows = await listEnabledManagedAgents();
      const options = rows.map((item) => ({
        value: item.id,
        label: `${item.name} (#${item.id})`,
      }));
      setAgentOptions(options);
      if (!selectedManagedAgentId && options.length > 0) {
        setSelectedManagedAgentId(options[0].value);
      }
      return options;
    } finally {
      setPickerLoading(false);
    }
  }, [selectedManagedAgentId]);

  const runGenerate = useCallback(
    async (managedAgentId: number) => {
      if (!ruleChainId || managedAgentId <= 0) return;
      setGenerating(true);
      try {
        await generateRuleChainSkill(ruleChainId, { managedAgentId });
        saveStoredManagedAgentId(managedAgentId);
        setSelectedManagedAgentId(managedAgentId);
        Toast.success({
          content: presentation.actionLabel === '更新技能' ? '技能已更新' : '技能已创建',
        });
        await refreshStatus();
        onGenerated?.();
      } catch (err) {
        Toast.error({ content: String((err as Error)?.message ?? err) });
        await refreshStatus();
      } finally {
        setGenerating(false);
        setPickerVisible(false);
      }
    },
    [onGenerated, presentation.actionLabel, refreshStatus, ruleChainId]
  );

  const handleGenerateClick = useCallback(async () => {
    const storedManagedAgentId = loadStoredManagedAgentId();
    let options: ManagedAgentOption[] | null = null;
    try {
      options = await loadManagedAgentOptions();
    } catch (err) {
      if (storedManagedAgentId > 0) {
        await runGenerate(storedManagedAgentId);
        return;
      }
      Toast.error({ content: String((err as Error)?.message ?? err) });
      return;
    }
    const resolved = resolveManagedAgentSelection(storedManagedAgentId, options);
    setSelectedManagedAgentId(resolved.managedAgentId);
    if (resolved.shouldClearStoredId) {
      saveStoredManagedAgentId(0);
    }
    if (!resolved.needsPicker && resolved.managedAgentId > 0) {
      await runGenerate(resolved.managedAgentId);
      return;
    }
    if (!options || options.length === 0) {
      Toast.warning({ content: '暂无可用托管 Agent，请先在 Agent 管理中启用一个配置' });
      return;
    }
    setPickerVisible(true);
  }, [loadManagedAgentOptions, runGenerate]);

  if (!isRoot) {
    return showStatusText ? (
      <Typography.Text type="tertiary" size="small">
        仅主规则链支持
      </Typography.Text>
    ) : null;
  }

  return (
    <>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
        {!hasLoadedStatus && statusError && !loadingStatus ? (
          <Tag color="red" type="ghost">
            状态异常
          </Tag>
        ) : (
          <Tag color={presentation.tagColor} type="ghost">
            {loadingStatus ? '加载中...' : presentation.tagLabel}
          </Tag>
        )}
        {statusError ? (
          <Typography.Text type="danger" size="small">
            {statusError}
          </Typography.Text>
        ) : null}
        {showStatusText && status.lastError ? (
          <Typography.Text type="tertiary" size="small">
            {status.lastError}
          </Typography.Text>
        ) : null}
        {!hasLoadedStatus && statusError && !loadingStatus ? (
          <Button size={size} type="tertiary" onClick={refreshStatus}>
            重试
          </Button>
        ) : null}
        {presentation.actionable && !(!hasLoadedStatus && statusError) ? (
          <Button
            size={size}
            type="primary"
            theme="solid"
            loading={generating}
            disabled={loadingStatus || generating}
            onClick={handleGenerateClick}
          >
            {presentation.actionLabel}
          </Button>
        ) : null}
      </div>

      <Modal
        title="选择托管 Agent"
        visible={pickerVisible}
        onCancel={() => setPickerVisible(false)}
        onOk={() => runGenerate(selectedManagedAgentId)}
        confirmLoading={generating}
        okButtonProps={{ disabled: pickerLoading || selectedManagedAgentId <= 0 }}
      >
        <Typography.Paragraph type="tertiary" style={{ marginBottom: 12 }}>
          首次生成规则链技能前，请先选择一个托管 Agent。后续会默认复用当前选择。
        </Typography.Paragraph>
        <Select
          style={{ width: '100%' }}
          loading={pickerLoading}
          value={selectedManagedAgentId > 0 ? String(selectedManagedAgentId) : undefined}
          optionList={agentOptions.map((item) => ({
            value: String(item.value),
            label: item.label,
          }))}
          placeholder="请选择托管 Agent"
          onChange={(value) => {
            const next = Number(value ?? 0);
            setSelectedManagedAgentId(Number.isFinite(next) ? next : 0);
          }}
        />
      </Modal>
    </>
  );
};

export default RuleChainSkillAction;
