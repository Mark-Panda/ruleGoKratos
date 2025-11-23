/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React from 'react';

import { usePanelManager } from '@flowgram.ai/panel-manager-plugin';
import { useService, WorkflowDocument, useClientContext } from '@flowgram.ai/free-layout-editor';
import { Button, Toast, Tooltip } from '@douyinfe/semi-ui';
import { IconSave } from '@douyinfe/semi-icons';

import { problemPanelFactory } from '../problem-panel';
import { collectWorkflowProblems } from '../../utils/workflow-validation';
import { buildRuleChainJSONFromDocument } from '../../utils/rulechain-builder';
import { getRuleBaseInfo } from '../../services/rule-base-info';
import { DirtyService } from '../../services/dirty-service';
import { updateRule } from '../../services/api-rules';

export const SaveButton: React.FC<{ disabled?: boolean }> = ({ disabled }) => {
  const wfDocument = useService(WorkflowDocument);
  const { playground, document } = useClientContext();
  const panelManager = usePanelManager();
  const dirtyService = useService(DirtyService);
  const [isDirty, setDirty] = React.useState<boolean>(dirtyService.dirty);
  React.useEffect(() => {
    const disposer = dirtyService.onChange((d) => setDirty(d));
    return () => disposer.dispose();
  }, [dirtyService]);

  const onClick = async () => {
    try {
      const problems = await collectWorkflowProblems(document);
      if (problems.length > 0) {
        panelManager.open(problemPanelFactory.key, 'bottom', { props: { problems } });
        Toast.error({ content: '存在未填写的必填项' });
        return;
      }
      const baseInfo = getRuleBaseInfo();
      const text = buildRuleChainJSONFromDocument(wfDocument, baseInfo);
      const payload = JSON.parse(text);
      const id = String(payload?.ruleChain?.id || baseInfo?.id || '');
      if (!id) {
        Toast.error({ content: '保存失败：缺少规则链ID' });
        return;
      }
      await updateRule(id, payload);
      Toast.success({ content: '保存成功' });
      try {
        dirtyService.setDirty(false);
      } catch {}
    } catch (e) {
      console.error(e);
      Toast.error({ content: `保存失败：${String((e as Error)?.message ?? e)}` });
    }
  };

  return (
    <Tooltip content="保存画布">
      <Button
        icon={<IconSave size="default" />}
        type="primary"
        theme="solid"
        disabled={disabled || playground.config.readonly || !isDirty}
        onClick={onClick}
      >
        保存
      </Button>
    </Tooltip>
  );
};
