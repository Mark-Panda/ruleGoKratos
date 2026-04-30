/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { pluginReact } from '@rsbuild/plugin-react';
import { pluginLess } from '@rsbuild/plugin-less';
import { defineConfig } from '@rsbuild/core';

export default defineConfig({
  plugins: [pluginReact(), pluginLess()],
  server: {
    proxy: {
      '/api': {
        target: process.env.RSBUILD_API_PROXY || 'http://127.0.0.1:8000',
        changeOrigin: true,
      },
    },
  },
  source: {
    entry: {
      index: './src/app.tsx',
    },
    /**
     * support inversify @injectable() and @inject decorators
     */
    decorators: {
      version: 'legacy',
    },
  },
  html: {
    title: 'BaBo Flow',
  },
  tools: {
    rspack: {
      /**
       * ignore warnings from @coze-editor/editor/language-typescript
       */
      ignoreWarnings: [/Critical dependency: the request of a dependency is an expression/],
    },
  },
});
