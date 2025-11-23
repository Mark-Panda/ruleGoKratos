/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect, useRef, useState } from 'react';

import { usePlayground } from '@flowgram.ai/free-layout-editor';
import { Collapsible, Tabs } from '@douyinfe/semi-ui';

import { GlobalVariableEditor } from './global-variable-editor';
import { FullVariableList } from './full-variable-list';

import styles from './index.module.less';

export function VariablePanel() {
  const playground = usePlayground();
  const isReadonly = playground?.config?.readonly;
  if (isReadonly) return null;
  const [isOpen, setOpen] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const open = () => setOpen(true);
    const close = () => setOpen(false);
    const toggle = () => setOpen((_open) => !_open);
    window.addEventListener('openVariablePanel', open as any);
    window.addEventListener('closeVariablePanel', close as any);
    window.addEventListener('toggleVariablePanel', toggle as any);
    return () => {
      window.removeEventListener('openVariablePanel', open as any);
      window.removeEventListener('closeVariablePanel', close as any);
      window.removeEventListener('toggleVariablePanel', toggle as any);
    };
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    const onDown = (e: MouseEvent) => {
      const el = containerRef.current;
      if (el && !el.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', onDown, true);
    return () => document.removeEventListener('mousedown', onDown, true);
  }, [isOpen]);

  return (
    <div className={styles['panel-wrapper']}>
      <Collapsible isOpen={isOpen}>
        <div ref={containerRef} className={styles['panel-container']}>
          <Tabs>
            <Tabs.TabPane itemKey="variables" tab="变量列表">
              <FullVariableList />
            </Tabs.TabPane>
            <Tabs.TabPane itemKey="global" tab="全局编辑器">
              <GlobalVariableEditor />
            </Tabs.TabPane>
          </Tabs>
        </div>
      </Collapsible>
    </div>
  );
}
