/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

/* eslint-disable import/no-extraneous-dependencies */
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE';

/** 未设置时：浏览器内用当前站点 origin（与 Docker/Nginx 同端口）；非浏览器回退 8000 便于本地脚本 */
export function getApiOrigin(): string {
  const fromEnv = (import.meta as any).env?.PUBLIC_API_ORIGIN as string | undefined;
  if (fromEnv != null && String(fromEnv).trim() !== '') {
    return String(fromEnv).replace(/\/$/, '');
  }
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin;
  }
  return 'http://127.0.0.1:8000';
}

const API_ORIGIN = getApiOrigin();
let BASE_URL = `${API_ORIGIN}/api/v1`;

const getToken = (): string => {
  if (typeof window === 'undefined') return '';
  return window.localStorage.getItem('AUTH_TOKEN') || window.localStorage.getItem('token') || '';
};

/** 供 WebSocket 等无法带 Authorization 头时使用（可选 query）。 */
export function getAuthToken(): string {
  return getToken();
}

export interface RequestOptions {
  method?: HttpMethod;
  headers?: Record<string, string>;
  body?: any;
  params?: Record<string, any>;
}

// 创建 axios 实例并设置拦截器
const client: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  headers: {
    Accept: 'application/json, text/plain, */*',
  },
});

client.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers = config.headers || {};
    (config.headers as any).Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (resp) => resp,
  async (error) => Promise.reject(error)
);

export const requestJSON = async <T = any>(path: string, opts: RequestOptions = {}): Promise<T> => {
  const config: AxiosRequestConfig = {
    url: path,
    method: (opts.method || 'GET') as any,
    params: opts.params,
    headers: { ...(opts.headers || {}) },
    data: opts.body,
  };
  try {
    const resp = await client.request<T>(config);
    return resp.data as T;
  } catch (error: any) {
    const r = error?.response;
    if (r) {
      const status = r.status;
      const text = typeof r.data === 'string' ? r.data : JSON.stringify(r.data);
      throw new Error(`HTTP ${status}: ${text || 'Request failed'}`);
    }
    throw error;
  }
};

export const requestRaw = async (
  path: string,
  opts: RequestOptions = {}
): Promise<{ ok: boolean; status: number; data: any }> => {
  const config: AxiosRequestConfig = {
    url: path,
    method: (opts.method || 'GET') as any,
    params: opts.params,
    headers: { ...(opts.headers || {}) },
    data: opts.body,
  };
  try {
    const resp = await client.request(config);
    return { ok: true, status: resp.status, data: resp.data };
  } catch (error: any) {
    const r = error?.response;
    if (r) {
      return { ok: false, status: r.status, data: r.data };
    }
    return { ok: false, status: 0, data: {} };
  }
};

export const setBaseURL = (url: string) => {
  BASE_URL = url;
  client.defaults.baseURL = url;
};

export const get = <T = any>(
  path: string,
  params?: RequestOptions['params'],
  headers?: Record<string, string>
) => requestJSON<T>(path, { method: 'GET', params, headers });

export const post = <T = any>(
  path: string,
  body?: any,
  headers?: Record<string, string>,
  params?: RequestOptions['params']
) => requestJSON<T>(path, { method: 'POST', body, headers, params });

export const put = <T = any>(
  path: string,
  body?: any,
  headers?: Record<string, string>,
  params?: RequestOptions['params']
) => requestJSON<T>(path, { method: 'PUT', body, headers, params });

export const del = <T = any>(
  path: string,
  params?: RequestOptions['params'],
  headers?: Record<string, string>
) => requestJSON<T>(path, { method: 'DELETE', params, headers });
