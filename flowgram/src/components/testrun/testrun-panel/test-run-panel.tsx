/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FC, useState, useEffect } from 'react';

import classnames from 'classnames';
import { WorkflowInputs, WorkflowOutputs } from '@flowgram.ai/runtime-interface';
import { type PanelFactory, usePanelManager } from '@flowgram.ai/panel-manager-plugin';
import { useService } from '@flowgram.ai/free-layout-editor';
import { Button } from '@douyinfe/semi-ui';
import { IconClose, IconPlay, IconSpin } from '@douyinfe/semi-icons';

import { TestRunJsonInput } from '../testrun-json-input';
import { TestRunForm } from '../testrun-form';
import { NodeStatusGroup } from '../node-status-bar/group';
import { alphaNanoid } from '../../../utils';
import { executeTestRun, fetchRunLogs } from '../../../services/test-run-http';
import { getRuleBaseInfo } from '../../../services/rule-base-info';
import { WorkflowRuntimeService } from '../../../plugins/runtime-plugin/runtime-service';
import { IconCancel } from '../../../assets/icon-cancel';

import styles from './index.module.less';

interface TestRunSidePanelProps {}

export const TestRunSidePanel: FC<TestRunSidePanelProps> = () => {
  const runtimeService = useService(WorkflowRuntimeService);

  const panelManager = usePanelManager();
  const [isRunning, setRunning] = useState(false);
  const [values, setValues] = useState<Record<string, unknown>>({
    headers: { 'Content-Type': 'application/json' },
    body: { temperature: 68 },
    metadata: 'key1=value1&key2=value2',
  });
  const [errors, setErrors] = useState<string[]>();
  const [result, setResult] = useState<
    | {
        inputs: WorkflowInputs;
        outputs: WorkflowOutputs;
      }
    | undefined
  >();
  const [logsData, setLogsData] = useState<any | undefined>();

  const onTestRun = async () => {
    if (isRunning) {
      await runtimeService.taskCancel();
      return;
    }
    setResult(undefined);
    setErrors(undefined);
    const base0 = getRuleBaseInfo();
    if (base0?.disabled) {
      setErrors(['当前规则链未部署']);
      return;
    }
    const mt = values.msgType;
    if (typeof mt !== 'string' || mt.trim().length === 0) {
      setErrors(['消息类型为必填']);
      return;
    }
    const base = getRuleBaseInfo();
    const ruleId = String(base?.id ?? '');
    if (!ruleId) {
      setErrors(['缺少规则链ID']);
      return;
    }
    const msgType = String(mt);
    setRunning(true);
    try {
      const msgId = alphaNanoid(24) + '11';
      const resp = await executeTestRun({
        ruleId,
        msgType,
        metadata: String(values.metadata ?? '').trim(),
        headers: (values.headers || {}) as any,
        body: (values.body as any) ?? {},
        debugMode: true,
        msgId,
      });
      setResult({ inputs: (values as any) ?? {}, outputs: (resp.data as any) ?? {} });
      if (resp.ok) {
        let timer: any;
        const poll = async () => {
          const logs = await fetchRunLogs(msgId);
          const list = logs?.logs;
          setLogsData(list);
          try {
            runtimeService.injectLogs(list || [], logs?.startTs, logs?.endTs);
          } catch {}
          if (Array.isArray(list) && list.length > 0 && timer) {
            clearInterval(timer);
            timer = null;
          }
        };
        await poll();
        if (!timer) timer = setInterval(poll, 1500);
        const stop = () => {
          if (timer) {
            clearInterval(timer);
            timer = null;
          }
        };
        runtimeService.onReset(() => stop());
      }
    } catch (e) {
      setErrors([String((e as Error)?.message ?? e)]);
    } finally {
      setRunning(false);
    }
  };

  const onClose = async () => {
    await runtimeService.taskCancel();
    setValues({});
    setRunning(false);
    panelManager.close(testRunPanelFactory.key);
  };

  const renderRunning = (
    <div className={styles['testrun-panel-running']}>
      <IconSpin spin size="large" />
      <div className={styles.text}>Running...</div>
    </div>
  );

  const renderForm = (
    <div className={styles['testrun-panel-form']}>
      <TestRunForm values={values} setValues={setValues} />
      {errors?.map((e) => (
        <div className={styles.error} key={e}>
          {e}
        </div>
      ))}
      <NodeStatusGroup title="Inputs Result" data={result?.inputs} optional disableCollapse />
      <NodeStatusGroup title="Outputs Result" data={result?.outputs} optional disableCollapse />
      <NodeStatusGroup title="Logs" data={formatLogs(logsData)} optional disableCollapse />
    </div>
  );

  const renderButton = (
    <Button
      onClick={onTestRun}
      icon={isRunning ? <IconCancel /> : <IconPlay size="small" />}
      className={classnames(styles.button, {
        [styles.running]: isRunning,
        [styles.default]: !isRunning,
      })}
    >
      {isRunning ? 'Cancel' : '试运行'}
    </Button>
  );

  useEffect(() => {
    const disposer = runtimeService.onResultChanged(({ result, errors }) => {
      setRunning(false);
      setResult(result);
      if (errors) {
        setErrors(errors);
      } else {
        setErrors(undefined);
      }
    });
    return () => disposer.dispose();
  }, []);

  useEffect(
    () => () => {
      runtimeService.taskCancel();
    },
    [runtimeService]
  );

  return (
    <div className={styles['testrun-panel-container']}>
      <div className={styles['testrun-panel-header']}>
        <div className={styles['testrun-panel-title']}>试运行</div>
        <Button
          className={styles['testrun-panel-title']}
          type="tertiary"
          icon={<IconClose />}
          size="small"
          theme="borderless"
          onClick={onClose}
        />
      </div>
      <div className={styles['testrun-panel-content']}>
        {isRunning ? renderRunning : renderForm}
      </div>
      <div className={styles['testrun-panel-footer']}>{renderButton}</div>
    </div>
  );
};

function formatLogs(logs: any[]): any {
  if (!Array.isArray(logs)) return undefined;
  const byNode: Record<string, any> = {};
  logs.forEach((item) => {
    const nodeId = item?.nodeId;
    if (!nodeId) return;
    byNode[nodeId] = {
      inMsg: item?.inMsg,
      outMsg: item?.outMsg,
      relationType: item?.relationType,
      err: item?.err,
    };
  });
  return byNode;
}

export const testRunPanelFactory: PanelFactory<TestRunSidePanelProps> = {
  key: 'test-run-panel',
  defaultSize: 400,
  render: () => <TestRunSidePanel />,
};
