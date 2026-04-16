/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

const { defineConfig } = require('@flowgram.ai/eslint-config');

module.exports = defineConfig({
  preset: 'web',
  packageRoot: __dirname,
  rules: {
    'no-console': 'off',
    'react/prop-types': 'off',
  },
  settings: {
    react: {
      version: 'detect', // 自动检测 React 版本
    },
  },
  overrides: [
    {
      files: [
        '**/__tests__/**/*.ts',
        '**/__tests__/**/*.tsx',
        '**/*.spec.ts',
        '**/*.spec.tsx',
        '**/*.test.ts',
        '**/*.test.tsx',
        'vitest.config.ts',
      ],
      rules: {
        // vitest 等在 devDependencies；单测与 vitest 配置允许从 dev 依赖导入
        'import/no-extraneous-dependencies': [
          'error',
          { devDependencies: true, peerDependencies: true, optionalDependencies: false },
        ],
      },
    },
  ],
});
