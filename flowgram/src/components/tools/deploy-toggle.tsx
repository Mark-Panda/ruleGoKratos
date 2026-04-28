import React from 'react';

import { Button, Toast, Tooltip } from '@douyinfe/semi-ui';
import { IconUpload, IconStop } from '@douyinfe/semi-icons';

import { getRuleBaseInfo, setRuleBaseInfo } from '../../services/rule-base-info';
import { getRuleDetail, startRuleChain, stopRuleChain } from '../../services/api-rules';
import { RULE_CHAIN_DEPLOY_STATUS_EVENT } from '../../constants/deploy-status-event';

export const DeployToggle: React.FC<{ disabled?: boolean }> = ({ disabled }) => {
  const [submitting, setSubmitting] = React.useState(false);
  const [deployed, setDeployed] = React.useState<boolean>(() => {
    const base = getRuleBaseInfo();
    return !Boolean(base?.disabled);
  });

  const refreshStatus = React.useCallback(async (id: string) => {
    const latest = await getRuleDetail(id);
    const latestRule = latest?.ruleChain;
    let nextDeployed = false;
    if (latestRule && typeof latestRule === 'object') {
      setRuleBaseInfo(latestRule as any);
      nextDeployed = !Boolean((latestRule as any)?.disabled);
    } else {
      const base = getRuleBaseInfo();
      nextDeployed = !Boolean(base?.disabled);
    }
    setDeployed(nextDeployed);
    window.dispatchEvent(
      new CustomEvent(RULE_CHAIN_DEPLOY_STATUS_EVENT, {
        detail: { id, deployed: nextDeployed },
      })
    );
  }, []);

  const onToggle = React.useCallback(async () => {
    const base = getRuleBaseInfo();
    const id = String(base?.id ?? '');
    if (!id) {
      Toast.error({ content: '缺少规则链ID，无法操作部署状态' });
      return;
    }
    setSubmitting(true);
    try {
      if (deployed) {
        await stopRuleChain(id);
        Toast.success({ content: '已下线' });
      } else {
        await startRuleChain(id);
        Toast.success({ content: '已部署' });
      }
      await refreshStatus(id);
    } catch (e) {
      Toast.error({ content: String((e as Error)?.message ?? e) || '操作失败' });
      await refreshStatus(id);
    } finally {
      setSubmitting(false);
    }
  }, [deployed, refreshStatus]);

  React.useEffect(() => {
    const base = getRuleBaseInfo();
    const id = String(base?.id ?? '');
    if (!id) return;
    refreshStatus(id).catch(() => {
      const latestBase = getRuleBaseInfo();
      setDeployed(!Boolean(latestBase?.disabled));
    });
  }, [refreshStatus]);

  const label = deployed ? '下线' : '部署';
  const icon = deployed ? <IconStop size="default" /> : <IconUpload size="default" />;

  return (
    <Tooltip content={deployed ? '将当前规则链下线' : '部署当前规则链'}>
      <Button
        icon={icon}
        theme={deployed ? 'light' : 'solid'}
        type={deployed ? 'danger' : 'primary'}
        loading={submitting}
        disabled={Boolean(disabled) || submitting}
        onClick={onToggle}
      >
        {label}
      </Button>
    </Tooltip>
  );
};
