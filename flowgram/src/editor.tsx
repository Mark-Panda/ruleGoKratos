/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import {
  EditorRenderer,
  FreeLayoutEditorProvider,
  useService,
} from '@flowgram.ai/free-layout-editor';

import '@flowgram.ai/free-layout-editor/index.css';
import './styles/index.css';
import { FlowDocumentJSON } from './typings';
import { WorkflowRuntimeService } from './plugins/runtime-plugin/runtime-service';
import { nodeRegistries } from './nodes';
import { initialData } from './initial-data';
import { useEditorProps } from './hooks';

import { useEffect } from 'react';

import { usePanelManager } from '@flowgram.ai/panel-manager-plugin';

import { TopToolbar } from './components/tools/top-toolbar';
import { DemoTools } from './components/tools';
import { testRunPanelFactory } from './components/testrun/testrun-panel';

export const Editor: React.FC<{
  initialDoc?: FlowDocumentJSON;
  showTopToolbar?: boolean;
  readonly?: boolean;
  initialLogs?: { list: any[]; startTs?: number; endTs?: number };
  openRunPanel?: boolean;
}> = ({
  initialDoc,
  showTopToolbar = false,
  readonly = false,
  initialLogs,
  openRunPanel = false,
}) => {
  const data = initialDoc ?? initialData;
  const editorProps = useEditorProps(data, nodeRegistries, readonly, initialLogs);
  return (
    <div
      className="doc-free-feature-overview"
      style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
    >
      <FreeLayoutEditorProvider {...editorProps}>
        {showTopToolbar && <TopToolbar />}
        <div className="demo-container">
          <EditorRenderer className="demo-editor" />
        </div>
        <DemoTools
          showBottomActions={!showTopToolbar}
          allowedTools={readonly ? ['interactive', 'zoomSelect'] : undefined}
        />
      </FreeLayoutEditorProvider>
    </div>
  );
};
