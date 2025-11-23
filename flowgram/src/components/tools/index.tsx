/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useState, useEffect } from 'react';

import { useRefresh } from '@flowgram.ai/free-layout-editor';
import { useClientContext } from '@flowgram.ai/free-layout-editor';
import { Tooltip, IconButton, Divider } from '@douyinfe/semi-ui';
import { IconUndo, IconRedo } from '@douyinfe/semi-icons';

import { TestRunButton } from '../testrun/testrun-button';
import { AddNode } from '../add-node';
import { ZoomSelect } from './zoom-select';
import { VariablePanelToggle } from './variable-panel-toggle';
import { SwitchLine } from './switch-line';
import { ToolContainer, ToolSection } from './styles';
import { SaveButton } from './save-button';
import { Readonly } from './readonly';
import { MinimapSwitch } from './minimap-switch';
import { Minimap } from './minimap';
import { Interactive } from './interactive';
import { FitView } from './fit-view';
import { Comment } from './comment';
import { AutoLayout } from './auto-layout';
import { ProblemButton } from '../problem-panel';
import { ExportImport } from './export-import';

export const DemoTools = ({
  showBottomActions = true,
  allowedTools,
}: {
  showBottomActions?: boolean;
  allowedTools?: string[];
}) => {
  const { history, playground } = useClientContext();
  const [canUndo, setCanUndo] = useState(false);
  const [canRedo, setCanRedo] = useState(false);
  const [minimapVisible, setMinimapVisible] = useState(true);
  useEffect(() => {
    const disposable = history.undoRedoService.onChange(() => {
      setCanUndo(history.canUndo());
      setCanRedo(history.canRedo());
    });
    return () => disposable.dispose();
  }, [history]);
  const refresh = useRefresh();

  useEffect(() => {
    const disposable = playground.config.onReadonlyOrDisabledChange(() => refresh());
    return () => disposable.dispose();
  }, [playground]);

  return (
    <ToolContainer className="demo-free-layout-tools">
      <ToolSection>
        {!allowedTools || allowedTools.includes('interactive') ? <Interactive /> : null}
        {!allowedTools || allowedTools.includes('autoLayout') ? <AutoLayout /> : null}
        {!allowedTools || allowedTools.includes('switchLine') ? <SwitchLine /> : null}
        {!allowedTools || allowedTools.includes('zoomSelect') ? <ZoomSelect /> : null}
        {!allowedTools || allowedTools.includes('fitView') ? <FitView /> : null}
        {!allowedTools || allowedTools.includes('minimapSwitch') ? (
          <MinimapSwitch minimapVisible={minimapVisible} setMinimapVisible={setMinimapVisible} />
        ) : null}
        {!allowedTools || allowedTools.includes('minimap') ? (
          <Minimap visible={minimapVisible} />
        ) : null}
        {!allowedTools || allowedTools.includes('readonly') ? <Readonly /> : null}
        {!allowedTools || allowedTools.includes('comment') ? <Comment /> : null}
        {!allowedTools || allowedTools.includes('undo') ? (
          <Tooltip content="Undo">
            <IconButton
              type="tertiary"
              theme="borderless"
              icon={<IconUndo />}
              disabled={!canUndo || playground.config.readonly}
              onClick={() => history.undo()}
            />
          </Tooltip>
        ) : null}
        {!allowedTools || allowedTools.includes('redo') ? (
          <Tooltip content="Redo">
            <IconButton
              type="tertiary"
              theme="borderless"
              icon={<IconRedo />}
              disabled={!canRedo || playground.config.readonly}
              onClick={() => history.redo()}
            />
          </Tooltip>
        ) : null}
        {!allowedTools || allowedTools.includes('problems') ? <ProblemButton /> : null}
        <Divider layout="vertical" style={{ height: '16px' }} margin={3} />
        {!allowedTools || allowedTools.includes('addNode') ? (
          <AddNode disabled={playground.config.readonly} />
        ) : null}
        {showBottomActions && (
          <>
            <Divider layout="vertical" style={{ height: '16px' }} margin={3} />
            {!allowedTools || allowedTools.includes('exportImport') ? (
              <ExportImport disabled={playground.config.readonly} />
            ) : null}
            {!allowedTools || allowedTools.includes('variablePanelToggle') ? (
              <VariablePanelToggle />
            ) : null}
            <Divider layout="vertical" style={{ height: '16px' }} margin={3} />
            {!allowedTools || allowedTools.includes('save') ? (
              <SaveButton disabled={playground.config.readonly} />
            ) : null}
            <Divider layout="vertical" style={{ height: '16px' }} margin={3} />
            {!allowedTools || allowedTools.includes('testRun') ? (
              <TestRunButton disabled={playground.config.readonly} />
            ) : null}
          </>
        )}
      </ToolSection>
    </ToolContainer>
  );
};
