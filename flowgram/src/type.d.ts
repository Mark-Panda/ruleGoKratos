/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

declare module '*.svg'
declare module '*.png'
declare module '*.jpg'
declare module '*.module.less'

declare module 'sanitize-html' {
  export interface IOptions {
    allowedTags?: string[];
    allowedAttributes?: Record<string, string[]>;
    allowedSchemes?: string[];
    allowedSchemesAppliedToAttributes?: string[];
  }

  export interface SanitizeHtml {
    (dirty: string, options?: IOptions): string;
    defaults: {
      allowedTags: string[];
      allowedAttributes: Record<string, string[]>;
    };
  }

  const sanitizeHtml: SanitizeHtml;
  export default sanitizeHtml;
}
