/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { domUtils, injectable, Layer } from '@flowgram.ai/free-layout-editor';

import { VariablePanel } from './components/variable-panel';

@injectable()
export class VariablePanelLayer extends Layer {
  onReady(): void {
    // 固定在画布右上角，不随滚动偏移
    domUtils.setStyle(this.node, {
      position: 'absolute',
      right: 290,
      top: 0,
      zIndex: 100,
      pointerEvents: 'auto',
    });
  }

  render(): JSX.Element {
    return <VariablePanel />;
  }
}
