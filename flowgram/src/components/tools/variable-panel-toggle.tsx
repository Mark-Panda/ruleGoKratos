/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useClientContext } from '@flowgram.ai/free-layout-editor';
import { Tooltip, Button } from '@douyinfe/semi-ui';

import iconVariable from '../../assets/icon-variable.png';

export function VariablePanelToggle() {
  const { playground } = useClientContext();
  const disabled = playground.config.readonly;

  return (
    <Tooltip content="变量面板">
      <Button
        theme="light"
        type="tertiary"
        disabled={disabled}
        icon={<img src={iconVariable} width={20} height={20} />}
        onClick={() => window.dispatchEvent(new Event('toggleVariablePanel'))}
      >
        变量
      </Button>
    </Tooltip>
  );
}
