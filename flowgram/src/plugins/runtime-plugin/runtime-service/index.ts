/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import {
  IReport,
  NodeReport,
  WorkflowInputs,
  WorkflowOutputs,
  WorkflowStatus,
} from '@flowgram.ai/runtime-interface';
import {
  injectable,
  inject,
  WorkflowDocument,
  Playground,
  WorkflowLineEntity,
  WorkflowNodeEntity,
  Emitter,
} from '@flowgram.ai/free-layout-editor';

import { WorkflowRuntimeClient } from '../client';
import { GetGlobalVariableSchema } from '../../variable-panel-plugin';
import { WorkflowNodeType } from '../../../nodes/constants';

const SYNC_TASK_REPORT_INTERVAL = 500;

interface NodeRunningStatus {
  nodeID: string;
  status: WorkflowStatus;
  nodeResultLength: number;
}

@injectable()
export class WorkflowRuntimeService {
  @inject(Playground) playground: Playground;

  @inject(WorkflowDocument) document: WorkflowDocument;

  @inject(WorkflowRuntimeClient) runtimeClient: WorkflowRuntimeClient;

  @inject(GetGlobalVariableSchema) getGlobalVariableSchema: GetGlobalVariableSchema;

  private runningNodes: WorkflowNodeEntity[] = [];

  private taskID?: string;

  private syncTaskReportIntervalID?: ReturnType<typeof setInterval>;

  private reportEmitter = new Emitter<NodeReport>();

  private resetEmitter = new Emitter<{}>();

  private resultEmitter = new Emitter<{
    errors?: string[];
    result?: {
      inputs: WorkflowInputs;
      outputs: WorkflowOutputs;
    };
  }>();

  private nodeRunningStatus: Map<string, NodeRunningStatus>;

  public onNodeReportChange = this.reportEmitter.event;

  public onReset = this.resetEmitter.event;

  public onResultChanged = this.resultEmitter.event;

  public isFlowingLine(line: WorkflowLineEntity) {
    return this.runningNodes.some((node) => node.lines.inputLines.includes(line));
  }

  public async taskRun(inputs: WorkflowInputs): Promise<string | undefined> {
    if (this.taskID) {
      await this.taskCancel();
    }
    const isFormValid = await this.validateForm();
    if (!isFormValid) {
      this.resultEmitter.fire({
        errors: ['Form validation failed'],
      });
      return;
    }
    const schema = {
      ...this.document.toJSON(),
      globalVariable: this.getGlobalVariableSchema(),
    };

    const validateResult = await this.runtimeClient.TaskValidate({
      schema: JSON.stringify(schema),
      inputs,
    });
    if (!validateResult?.valid) {
      this.resultEmitter.fire({
        errors: validateResult?.errors ?? ['Internal Server Error'],
      });
      return;
    }
    this.reset();
    let taskID: string | undefined;
    try {
      const output = await this.runtimeClient.TaskRun({
        schema: JSON.stringify(schema),
        inputs,
      });
      taskID = output?.taskID;
    } catch (e) {
      this.resultEmitter.fire({
        errors: [(e as Error)?.message],
      });
      return;
    }
    if (!taskID) {
      this.resultEmitter.fire({
        errors: ['Task run failed'],
      });
      return;
    }
    this.taskID = taskID;
    this.syncTaskReportIntervalID = setInterval(() => {
      this.syncTaskReport();
    }, SYNC_TASK_REPORT_INTERVAL);
    return this.taskID;
  }

  public async taskCancel(): Promise<void> {
    if (!this.taskID) {
      return;
    }
    await this.runtimeClient.TaskCancel({
      taskID: this.taskID,
    });
  }

  private async validateForm(): Promise<boolean> {
    const allForms = this.document.getAllNodes().map((node) => node.form);
    const formValidations = await Promise.all(allForms.map(async (form) => form?.validate()));
    const validations = formValidations.filter((validation) => validation !== undefined);
    const isValid = validations.every((validation) => validation);
    if (!isValid) return false;
    const nodes = this.document.getAllNodes();
    const toJSONList = nodes.map((n) => this.document.toNodeJSON(n));
    const getVal = (v: any) => {
      if (!v) return undefined;
      if (typeof v.content !== 'undefined') return v.content;
      return v;
    };
    const isEmpty = (schema: any, val: any) => {
      const t = schema?.type;
      if (t === 'string') return !(typeof val === 'string' && val.trim().length > 0);
      if (t === 'number') return !(typeof val === 'number');
      if (t === 'boolean') return typeof val === 'boolean' ? false : true;
      if (t === 'array') return Array.isArray(val) ? false : true;
      if (t === 'object') return typeof val === 'object' && val !== null ? false : true;
      return val === undefined || val === null;
    };
    let requiredOk = true;
    const validateNode = (json: any) => {
      const inputs = json?.data?.inputs;
      const values = json?.data?.inputsValues;
      const requiredKeys: string[] = Array.isArray(inputs?.required) ? inputs.required : [];
      requiredKeys.forEach((k) => {
        const schema = inputs?.properties?.[k];
        const v = getVal(values?.[k]);
        if (isEmpty(schema, v)) requiredOk = false;
      });
      const blocks = Array.isArray(json?.blocks) ? json.blocks : [];
      blocks.forEach((b: any) => validateNode(b));
    };
    toJSONList.forEach(validateNode);
    return requiredOk;
  }

  private reset(): void {
    this.taskID = undefined;
    this.nodeRunningStatus = new Map();
    this.runningNodes = [];
    if (this.syncTaskReportIntervalID) {
      clearInterval(this.syncTaskReportIntervalID);
    }
    this.resetEmitter.fire({});
  }

  private async syncTaskReport(): Promise<void> {
    if (!this.taskID) {
      return;
    }
    const report = await this.runtimeClient.TaskReport({
      taskID: this.taskID,
    });
    if (!report) {
      clearInterval(this.syncTaskReportIntervalID);
      console.error('Sync task report failed');
      return;
    }
    const { workflowStatus, inputs, outputs, messages } = report;
    if (workflowStatus.terminated) {
      clearInterval(this.syncTaskReportIntervalID);
      if (Object.keys(outputs).length > 0) {
        this.resultEmitter.fire({ result: { inputs, outputs } });
      } else {
        this.resultEmitter.fire({
          errors: messages?.error?.map((message) =>
            message.nodeID ? `${message.nodeID}: ${message.message}` : message.message
          ),
        });
      }
    }
    this.updateReport(report);
  }

  public injectLogs(logs: any[], startTs?: number, endTs?: number): void {
    const timeCost = typeof startTs === 'number' && typeof endTs === 'number' ? endTs - startTs : 0;
    const toVal = (v: any) => {
      if (v === null || v === undefined) return v;
      if (typeof v === 'string') {
        try {
          const parsed = JSON.parse(v);
          return parsed;
        } catch {
          return v;
        }
      }
      return v;
    };
    (logs || []).forEach((item: any) => {
      const id = String(item?.nodeId ?? '');
      if (!id) return;
      const inputs = item?.inMsg ? { ...item.inMsg, data: toVal(item.inMsg?.data) } : undefined;
      const outputs = item?.outMsg ? { ...item.outMsg, data: toVal(item.outMsg?.data) } : undefined;
      const branch = item?.relationType;
      const error = item?.err;
      const nodeReport: NodeReport = {
        id,
        status: error ? WorkflowStatus.Failed : WorkflowStatus.Succeeded,
        timeCost,
        snapshots: [
          {
            inputs,
            outputs,
            branch,
            data: undefined,
            error,
          } as any,
        ],
      } as any;
      this.reportEmitter.fire(nodeReport);
    });
    try {
      this.document.linesManager.forceUpdate();
    } catch {}
  }

  private updateReport(report: IReport): void {
    const { reports } = report;
    this.runningNodes = [];
    this.document
      .getAllNodes()
      .filter(
        (node) =>
          ![WorkflowNodeType.BlockStart, WorkflowNodeType.BlockEnd].includes(
            node.flowNodeType as WorkflowNodeType
          )
      )
      .forEach((node) => {
        const nodeID = node.id;
        const nodeReport = reports[nodeID];
        if (!nodeReport) {
          return;
        }
        if (nodeReport.status === WorkflowStatus.Processing) {
          this.runningNodes.push(node);
        }
        const runningStatus = this.nodeRunningStatus.get(nodeID);
        if (
          !runningStatus ||
          nodeReport.status !== runningStatus.status ||
          nodeReport.snapshots.length !== runningStatus.nodeResultLength
        ) {
          this.nodeRunningStatus.set(nodeID, {
            nodeID,
            status: nodeReport.status,
            nodeResultLength: nodeReport.snapshots.length,
          });
          this.reportEmitter.fire(nodeReport);
          this.document.linesManager.forceUpdate();
        } else if (nodeReport.status === WorkflowStatus.Processing) {
          this.reportEmitter.fire(nodeReport);
        }
      });
  }
}
