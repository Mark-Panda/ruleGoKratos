/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useState, useEffect, useCallback } from 'react';

import { useClientContext, FlowNodeEntity } from '@flowgram.ai/free-layout-editor';
import { Button, Badge, Toast } from '@douyinfe/semi-ui';

import { buildRuleChainJSONFromDocument } from '../../utils/rulechain-builder';
import { getRuleBaseInfo } from '../../services/rule-base-info';
import { updateRule } from '../../services/api-rules';

export function Save(props: { disabled: boolean }) {
  const [errorCount, setErrorCount] = useState(0);
  const clientContext = useClientContext();

  const updateValidateData = useCallback(() => {
    const allForms = clientContext.document.getAllNodes().map((node) => node.form);
    const count = allForms.filter((form) => form?.state.invalid).length;
    setErrorCount(count);
  }, [clientContext]);

  /**
   * Validate all node and Save
   */
  const onSave = useCallback(async () => {
    const allForms = clientContext.document.getAllNodes().map((node) => node.form);
    await Promise.all(allForms.map(async (form) => form?.validate()));
    try {
      const baseInfo = getRuleBaseInfo();
      const text = buildRuleChainJSONFromDocument(clientContext.document, baseInfo);
      const payload = JSON.parse(text);
      const id = String(payload?.ruleChain?.id || baseInfo?.id || '');
      if (!id) {
        Toast.error('保存失败：缺少规则链ID');
        return;
      }
      await updateRule(id, payload);
      Toast.success('保存成功');
    } catch (e) {
      console.error(e);
      Toast.error(`保存失败：${String((e as Error)?.message ?? e)}`);
    }
  }, [clientContext]);

  /**
   * Listen single node validate
   */
  useEffect(() => {
    const listenSingleNodeValidate = (node: FlowNodeEntity) => {
      const { form } = node;
      if (form) {
        const formValidateDispose = form.onValidate(() => updateValidateData());
        node.onDispose(() => formValidateDispose.dispose());
      }
    };
    clientContext.document.getAllNodes().map((node) => listenSingleNodeValidate(node));
    const dispose = clientContext.document.onNodeCreate(({ node }) =>
      listenSingleNodeValidate(node)
    );
    return () => dispose.dispose();
  }, [clientContext]);

  if (errorCount === 0) {
    return (
      <Button
        disabled={props.disabled}
        onClick={onSave}
        style={{ backgroundColor: 'rgba(171,181,255,0.3)', borderRadius: '8px' }}
      >
        Save
      </Button>
    );
  }
  return (
    <Badge count={errorCount} position="rightTop" type="danger">
      <Button
        type="danger"
        disabled={props.disabled}
        onClick={onSave}
        style={{ backgroundColor: 'rgba(255, 179, 171, 0.3)', borderRadius: '8px' }}
      >
          Save
      </Button>
    </Badge>
  );
}
