/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React from 'react';

import { createRoot } from 'react-dom/client';
import { unstableSetCreateRoot } from '@flowgram.ai/form-materials';

import { RuleDetailPage } from './management/rule-detail-page';
import { AdminPanel } from './management/admin-panel';

/**
 * React 18/19 polyfill for form-materials
 */
unstableSetCreateRoot(createRoot);

/**
 * 忽略 ResizeObserver loop 警告
 * 这是一个已知的浏览器问题，不会影响功能
 * 参考：https://github.com/WICG/resize-observer/issues/38
 */
const resizeObserverErrorHandler = (e: ErrorEvent) => {
  if (
    e.message === 'ResizeObserver loop completed with undelivered notifications.' ||
    e.message === 'ResizeObserver loop limit exceeded'
  ) {
    e.stopImmediatePropagation();
    return true;
  }
  return false;
};

window.addEventListener('error', resizeObserverErrorHandler);

const app = createRoot(document.getElementById('root')!);

function Router() {
  const [hash, setHash] = React.useState<string>(() => window.location.hash || '#/');
  React.useEffect(() => {
    const onHash = () => setHash(window.location.hash || '#/');
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const match = hash.match(/^#\/workflow\/([^/]+)(?:\/(design))?$/);
  if (match) {
    const id = decodeURIComponent(match[1]);
    const tab = match[2] as 'design' | undefined;
    return <RuleDetailPage id={id} tab={tab} />;
  }
  return <AdminPanel />;
}

app.render(<Router />);
