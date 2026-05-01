import {
  AlarmClock,
  Api,
  ApiApp,
  Command,
  Code,
  CompassOne,
  DataAll,
  DataSwitching,
  DatabasePoint,
  DatabaseSearch,
  Delete,
  Download,
  FileCodeOne,
  FileLock,
  FileDisplay,
  FileSearch,
  Filter,
  Formula,
  Gitlab,
  Histogram,
  LinkCloud,
  ListView,
  Lock,
  Log,
  LoopOnce,
  Merge,
  NetworkTree,
  Pause,
  PauseOne,
  Play,
  PlayOne,
  Power,
  Return,
  RobotOne,
  Router,
  Search,
  SendEmail,
  SettingConfig,
  SortTwo,
  SplitBranch,
  SplitTurnDownLeft,
  SplitTurnDownRight,
  Switch,
  Terminal,
  TextMessage,
  Timer,
  Upload,
  ViewGridCard,
  Write,
} from '@icon-park/react';

import { FlowNodeRegistry } from '../../typings';
import { WorkflowNodeType } from '../../nodes';

interface NodeIconOptions {
  size?: number;
  strokeWidth?: number;
}

interface RegistryNodeIconOptions extends NodeIconOptions {
  useFallbackImage?: boolean;
}

const NODE_COLOR_PALETTE = {
  lifecycle: ['#16a34a', '#22c55e', '#ef4444', '#dc2626'],
  logic: ['#f59e0b', '#d97706', '#b45309', '#0d9488', '#0f766e'],
  script: ['#4f46e5', '#6366f1', '#4338ca', '#3730a3'],
  data: ['#0ea5e9', '#0891b2', '#0284c7', '#0369a1'],
  storage: ['#f97316', '#dc2626', '#7c2d12', '#c2410c'],
  file: ['#0ea5e9', '#10b981', '#ef4444', '#8b5cf6'],
  external: ['#2563eb', '#1d4ed8', '#7c3aed'],
  git: ['#059669', '#16a34a', '#3b82f6'],
  runtime: ['#6b7280', '#4b5563', '#374151'],
  observability: ['#64748b', '#475569', '#06b6d4'],
  product: ['#7c3aed', '#a855f7', '#8b5cf6'],
  fallback: ['#334155'],
} as const;

function getNodeIconColor(type: string): string {
  switch (type) {
    case WorkflowNodeType.Start:
      return NODE_COLOR_PALETTE.lifecycle[0];
    case WorkflowNodeType.BlockStart:
      return NODE_COLOR_PALETTE.lifecycle[1];
    case WorkflowNodeType.End:
      return NODE_COLOR_PALETTE.lifecycle[2];
    case WorkflowNodeType.BlockEnd:
      return NODE_COLOR_PALETTE.lifecycle[3];
    case WorkflowNodeType.HTTP:
      return NODE_COLOR_PALETTE.external[0];
    case WorkflowNodeType.Yapi:
      return NODE_COLOR_PALETTE.external[1];
    case WorkflowNodeType.ServiceManagement:
      return NODE_COLOR_PALETTE.product[0];
    case WorkflowNodeType.ApiRouteTracerSourcegraph:
      return NODE_COLOR_PALETTE.external[0];
    case WorkflowNodeType.AgentHarness:
      return NODE_COLOR_PALETTE.external[2];
    case WorkflowNodeType.Code:
      return NODE_COLOR_PALETTE.script[0];
    case WorkflowNodeType.Transform:
      return NODE_COLOR_PALETTE.script[1];
    case WorkflowNodeType.JsFilter:
      return NODE_COLOR_PALETTE.script[2];
    case WorkflowNodeType.LuaTransform:
      return NODE_COLOR_PALETTE.script[3];
    case WorkflowNodeType.Variable:
      return NODE_COLOR_PALETTE.data[0];
    case WorkflowNodeType.JsonExtract:
      return NODE_COLOR_PALETTE.data[1];
    case WorkflowNodeType.Condition:
      return NODE_COLOR_PALETTE.logic[0];
    case WorkflowNodeType.CaseCondition:
      return NODE_COLOR_PALETTE.logic[1];
    case WorkflowNodeType.Inclusive:
      return NODE_COLOR_PALETTE.logic[2];
    case WorkflowNodeType.Loop:
      return NODE_COLOR_PALETTE.logic[3];
    case WorkflowNodeType.For:
      return NODE_COLOR_PALETTE.logic[4];
    case WorkflowNodeType.While:
      return NODE_COLOR_PALETTE.logic[4];
    case WorkflowNodeType.Continue:
      return NODE_COLOR_PALETTE.lifecycle[1];
    case WorkflowNodeType.Break:
      return NODE_COLOR_PALETTE.lifecycle[2];
    case WorkflowNodeType.Fork:
      return NODE_COLOR_PALETTE.logic[3];
    case WorkflowNodeType.Join:
      return NODE_COLOR_PALETTE.logic[4];
    case WorkflowNodeType.FetchNodeOutput:
      return NODE_COLOR_PALETTE.observability[2];
    case WorkflowNodeType.MultiNodeOutput:
      return NODE_COLOR_PALETTE.data[2];
    case WorkflowNodeType.LogString:
      return NODE_COLOR_PALETTE.observability[0];
    case WorkflowNodeType.Comment:
      return NODE_COLOR_PALETTE.observability[1];
    case WorkflowNodeType.DBClient:
      return NODE_COLOR_PALETTE.storage[0];
    case WorkflowNodeType.RedisClient:
      return NODE_COLOR_PALETTE.storage[1];
    case WorkflowNodeType.OpenSearchSearch:
      return NODE_COLOR_PALETTE.storage[2];
    case WorkflowNodeType.VolcTlsSearchLogs:
      return NODE_COLOR_PALETTE.storage[3];
    case WorkflowNodeType.Cron:
      return NODE_COLOR_PALETTE.logic[0];
    case WorkflowNodeType.Flow:
      return NODE_COLOR_PALETTE.data[3];
    case WorkflowNodeType.Exec:
      return NODE_COLOR_PALETTE.runtime[0];
    case WorkflowNodeType.CursorCli:
      return NODE_COLOR_PALETTE.runtime[1];
    case WorkflowNodeType.CursorCliAuth:
      return NODE_COLOR_PALETTE.runtime[2];
    case WorkflowNodeType.CursorAcp:
      return NODE_COLOR_PALETTE.external[1];
    case WorkflowNodeType.FeishuWebhook:
      return NODE_COLOR_PALETTE.external[0];
    case WorkflowNodeType.FeishuCliAuth:
      return NODE_COLOR_PALETTE.external[1];
    case WorkflowNodeType.FileRead:
      return NODE_COLOR_PALETTE.file[0];
    case WorkflowNodeType.FileWrite:
      return NODE_COLOR_PALETTE.file[1];
    case WorkflowNodeType.FileDelete:
      return NODE_COLOR_PALETTE.file[2];
    case WorkflowNodeType.FileList:
      return NODE_COLOR_PALETTE.file[3];
    case WorkflowNodeType.GitClone:
      return NODE_COLOR_PALETTE.git[0];
    case WorkflowNodeType.GitCommit:
      return NODE_COLOR_PALETTE.git[1];
    case WorkflowNodeType.GitPush:
      return NODE_COLOR_PALETTE.git[2];
    case WorkflowNodeType.TaskBoard:
      return NODE_COLOR_PALETTE.product[1];
    default:
      return NODE_COLOR_PALETTE.fallback[0];
  }
}

const getIconCommonProps = (type: string, options?: NodeIconOptions) => ({
  theme: 'outline' as const,
  size: options?.size ?? 14,
  strokeWidth: options?.strokeWidth ?? 3,
  fill: getNodeIconColor(type),
});

function getIconByNodeType(type: string, options?: NodeIconOptions): JSX.Element | null {
  const iconCommonProps = getIconCommonProps(type, options);
  switch (type) {
    case WorkflowNodeType.Start:
      return <PlayOne {...iconCommonProps} />;
    case WorkflowNodeType.BlockStart:
      return <Play {...iconCommonProps} />;
    case WorkflowNodeType.End:
      return <PauseOne {...iconCommonProps} />;
    case WorkflowNodeType.BlockEnd:
      return <Power {...iconCommonProps} />;
    case WorkflowNodeType.HTTP:
      return <Api {...iconCommonProps} />;
    case WorkflowNodeType.Yapi:
      return <ApiApp {...iconCommonProps} />;
    case WorkflowNodeType.ServiceManagement:
      return <SettingConfig {...iconCommonProps} />;
    case WorkflowNodeType.ApiRouteTracerSourcegraph:
      return <Router {...iconCommonProps} />;
    case WorkflowNodeType.AgentHarness:
      return <RobotOne {...iconCommonProps} />;
    case WorkflowNodeType.Code:
      return <Code {...iconCommonProps} />;
    case WorkflowNodeType.Transform:
      return <DataSwitching {...iconCommonProps} />;
    case WorkflowNodeType.JsFilter:
      return <Filter {...iconCommonProps} />;
    case WorkflowNodeType.LuaTransform:
      return <Formula {...iconCommonProps} />;
    case WorkflowNodeType.Variable:
      return <DataAll {...iconCommonProps} />;
    case WorkflowNodeType.JsonExtract:
      return <FileCodeOne {...iconCommonProps} />;
    case WorkflowNodeType.Condition:
      return <Switch {...iconCommonProps} />;
    case WorkflowNodeType.CaseCondition:
      return <SplitTurnDownRight {...iconCommonProps} />;
    case WorkflowNodeType.Inclusive:
      return <SplitTurnDownLeft {...iconCommonProps} />;
    case WorkflowNodeType.Loop:
      return <LoopOnce {...iconCommonProps} />;
    case WorkflowNodeType.For:
      return <SortTwo {...iconCommonProps} />;
    case WorkflowNodeType.While:
      return <Timer {...iconCommonProps} />;
    case WorkflowNodeType.Continue:
      return <Return {...iconCommonProps} />;
    case WorkflowNodeType.Break:
      return <Pause {...iconCommonProps} />;
    case WorkflowNodeType.Fork:
      return <SplitBranch {...iconCommonProps} />;
    case WorkflowNodeType.Join:
      return <Merge {...iconCommonProps} />;
    case WorkflowNodeType.FetchNodeOutput:
      return <Download {...iconCommonProps} />;
    case WorkflowNodeType.MultiNodeOutput:
      return <ViewGridCard {...iconCommonProps} />;
    case WorkflowNodeType.LogString:
      return <Log {...iconCommonProps} />;
    case WorkflowNodeType.Comment:
      return <TextMessage {...iconCommonProps} />;
    case WorkflowNodeType.DBClient:
      return <DatabaseSearch {...iconCommonProps} />;
    case WorkflowNodeType.RedisClient:
      return <DatabasePoint {...iconCommonProps} />;
    case WorkflowNodeType.OpenSearchSearch:
      return <Search {...iconCommonProps} />;
    case WorkflowNodeType.VolcTlsSearchLogs:
      return <Histogram {...iconCommonProps} />;
    case WorkflowNodeType.Cron:
      return <AlarmClock {...iconCommonProps} />;
    case WorkflowNodeType.Flow:
      return <NetworkTree {...iconCommonProps} />;
    case WorkflowNodeType.Exec:
      return <Command {...iconCommonProps} />;
    case WorkflowNodeType.CursorCli:
      return <Terminal {...iconCommonProps} />;
    case WorkflowNodeType.CursorAcp:
      return <CompassOne {...iconCommonProps} />;
    case WorkflowNodeType.FileRead:
      return <FileSearch {...iconCommonProps} />;
    case WorkflowNodeType.FileWrite:
      return <Write {...iconCommonProps} />;
    case WorkflowNodeType.FileDelete:
      return <Delete {...iconCommonProps} />;
    case WorkflowNodeType.FileList:
      return <FileDisplay {...iconCommonProps} />;
    case WorkflowNodeType.GitClone:
      return <LinkCloud {...iconCommonProps} />;
    case WorkflowNodeType.GitCommit:
      return <Gitlab {...iconCommonProps} />;
    case WorkflowNodeType.GitPush:
      return <Upload {...iconCommonProps} />;
    case WorkflowNodeType.TaskBoard:
      return <ListView {...iconCommonProps} />;
    case WorkflowNodeType.FeishuWebhook:
      return <SendEmail {...iconCommonProps} />;
    case WorkflowNodeType.FeishuCliAuth:
      return <Lock {...iconCommonProps} />;
    case WorkflowNodeType.CursorCliAuth:
      return <FileLock {...iconCommonProps} />;
    default:
      return null;
  }
}

export function renderNodePanelIcon(
  registry: FlowNodeRegistry,
  options?: RegistryNodeIconOptions
): JSX.Element {
  const mapped = getIconByNodeType(String(registry.type), options);
  if (mapped) {
    return mapped;
  }

  const useFallbackImage = options?.useFallbackImage ?? true;
  if (useFallbackImage) {
    const iconUrl = registry.info?.icon;
    if (iconUrl) {
      const size = options?.size ?? 14;
      return <img style={{ width: size, height: size, borderRadius: 4 }} src={iconUrl} alt="" />;
    }
  }

  return <Router {...getIconCommonProps(String(registry.type), options)} />;
}
