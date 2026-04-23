/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useState } from 'react';

import { Nav, Typography, Breadcrumb, Tabs, TabPane } from '@douyinfe/semi-ui';
import {
  IconUser,
  IconHome,
  IconList,
  IconChevronLeft,
  IconChevronRight,
  IconBranch,
  IconDesktop,
  IconSetting,
} from '@douyinfe/semi-icons';

import { WorkflowSection } from './sections/WorkflowSection';
import { WorkflowRunLogsSection } from './sections/WorkflowRunLogsSection';
import { WorkflowExecuteSection } from './sections/WorkflowExecuteSection';
import { ComponentsSection } from './sections/ComponentsSection';
import { AgentSection } from './sections/AgentSection';
import { OverviewChatSection } from './sections/OverviewChatSection';
import { ManagedAgentsSection } from './sections/ManagedAgentsSection';
import { AgentPlaygroundPage } from '../agent-playground';
import { TerminalSection } from './sections/TerminalSection';
import { LarkCliSection } from './sections/LarkCliSection';
import { WorkspacesSection } from './sections/WorkspacesSection';

type MenuKey =
  | 'intro'
  | 'workflow'
  | 'workflow-run'
  | 'workflow-logs'
  | 'component-installed'
  | 'component-rules'
  | 'agent-skills'
  | 'agent-mcp'
  | 'agent-models'
  | 'agent-playground'
  | 'agent-profiles'
  | 'workspace-manager'
  | 'admin-terminal'
  | 'admin-lark-cli'
  | 'engine'
  | 'component'
  | 'agent';

/** 与路由对应的菜单页 itemKey（不含 Nav 分组占位 key） */
const MENU_KEYS: MenuKey[] = [
  'intro',
  'workflow',
  'workflow-run',
  'workflow-logs',
  'component-installed',
  'component-rules',
  'agent-skills',
  'agent-mcp',
  'agent-models',
  'agent-playground',
  'agent-profiles',
  'workspace-manager',
  'admin-terminal',
  'admin-lark-cli',
];

function getMenuFromHash(h: string): MenuKey {
  if (h === '#/' || h === '' || h === '#') return 'intro';
  if (h.startsWith('#/terminal')) return 'admin-terminal';
  if (h.startsWith('#/lark-cli')) return 'admin-lark-cli';
  if (h.startsWith('#/workspaces')) return 'workspace-manager';
  if (h.startsWith('#/agent/profiles')) return 'agent-profiles';
  if (h.startsWith('#/agent/skills')) return 'agent-skills';
  if (h.startsWith('#/agent/models')) return 'agent-models';
  if (h.startsWith('#/agent/mcp')) return 'agent-mcp';
  if (h.startsWith('#/playground')) return 'agent-playground';
  if (h.startsWith('#/components/rules')) return 'component-rules';
  if (h.startsWith('#/components')) return 'component-installed';
  if (h.startsWith('#/workflow/run')) return 'workflow-run';
  if (h.startsWith('#/workflow/logs')) return 'workflow-logs';
  return 'workflow';
}

function setHashForMenu(key: MenuKey) {
  if (key === 'intro') window.location.hash = '#/';
  else if (key === 'workflow') window.location.hash = '#/admin';
  else if (key === 'workflow-run') window.location.hash = '#/workflow/run';
  else if (key === 'workflow-logs') window.location.hash = '#/workflow/logs';
  else if (key === 'component-installed') window.location.hash = '#/components';
  else if (key === 'component-rules') window.location.hash = '#/components/rules';
  else if (key === 'agent-skills') window.location.hash = '#/agent/skills';
  else if (key === 'agent-models') window.location.hash = '#/agent/models';
  else if (key === 'agent-mcp') window.location.hash = '#/agent/mcp';
  else if (key === 'agent-playground') window.location.hash = '#/playground';
  else if (key === 'agent-profiles') window.location.hash = '#/agent/profiles';
  else if (key === 'workspace-manager') window.location.hash = '#/workspaces';
  else if (key === 'admin-terminal') window.location.hash = '#/terminal';
  else if (key === 'admin-lark-cli') window.location.hash = '#/lark-cli';
}

export const AdminPanel: React.FC = () => {
  const [activeMenu, setActiveMenu] = useState<MenuKey>(() => {
    try {
      const h = typeof window !== 'undefined' ? window.location.hash : '';
      return getMenuFromHash(h || '#/');
    } catch {
      return 'intro';
    }
  });

  const [openTabs, setOpenTabs] = useState<MenuKey[]>(() => {
    try {
      const h = typeof window !== 'undefined' ? window.location.hash : '';
      const menu = getMenuFromHash(h || '#/');
      return menu === 'intro' ? ['intro'] : ['intro', menu];
    } catch {
      return ['intro'];
    }
  });

  useEffect(() => {
    const onHash = () => {
      try {
        const h = typeof window !== 'undefined' ? window.location.hash : '';
        const menu = getMenuFromHash(h || '#/');
        setActiveMenu(menu);
        setOpenTabs((prev) => (prev.includes(menu) ? prev : [...prev, menu]));
      } catch {}
    };
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const renderPage = (key: MenuKey) => {
    if (key === 'intro') return <OverviewChatSection />;
    if (key === 'agent-profiles') return <ManagedAgentsSection />;
    if (key === 'workflow') return <WorkflowSection />;
    if (key === 'workflow-run') return <WorkflowExecuteSection />;
    if (key === 'workflow-logs') return <WorkflowRunLogsSection />;
    if (key === 'agent-skills') return <AgentSection view="skills" />;
    if (key === 'agent-models') return <AgentSection view="models" />;
    if (key === 'agent-mcp') return <AgentSection view="mcps" />;
    if (key === 'agent-playground') return <AgentPlaygroundPage />;
    if (key === 'workspace-manager') return <WorkspacesSection />;
    if (key === 'admin-terminal') return <TerminalSection />;
    if (key === 'admin-lark-cli') return <LarkCliSection />;
    if (key === 'component-rules') return <ComponentsSection view="rules" />;
    return <ComponentsSection view="installed" />;
  };

  const getPageTitle = (menu: MenuKey = activeMenu) => {
    switch (menu) {
      case 'intro':
        return 'Code 助手';
      case 'workflow':
        return '流程管理';
      case 'workflow-run':
        return '工作流执行';
      case 'workflow-logs':
        return '执行日志';
      case 'component-installed':
        return '已安装组件';
      case 'component-rules':
        return '组件规则';
      case 'agent-skills':
        return 'SKILL 管理';
      case 'agent-mcp':
        return 'MCP 配置';
      case 'agent-models':
        return '模型管理';
      case 'agent-playground':
        return 'Agent Playground';
      case 'agent-profiles':
        return 'Agent 配置';
      case 'workspace-manager':
        return '工作区管理';
      case 'admin-terminal':
        return '终端';
      case 'admin-lark-cli':
        return '飞书 CLI 配置';
      default:
        return 'Code 助手';
    }
  };

  const getParentTitle = () => {
    if (
      activeMenu === 'workflow' ||
      activeMenu === 'workflow-run' ||
      activeMenu === 'workflow-logs'
    )
      return '工作流引擎';
    if (
      activeMenu === 'agent-skills' ||
      activeMenu === 'agent-mcp' ||
      activeMenu === 'agent-models'
    )
      return '模型与工具';
    if (activeMenu === 'component-installed' || activeMenu === 'component-rules') return '组件管理';
    if (activeMenu === 'intro') return '工作台';
    if (activeMenu === 'agent-playground' || activeMenu === 'agent-profiles' || activeMenu === 'workspace-manager') return '智能体';
    if (activeMenu === 'admin-terminal' || activeMenu === 'admin-lark-cli') return '运维';
    return '系统';
  };

  const openMenu = (key: MenuKey) => {
    setActiveMenu(key);
    setHashForMenu(key);
    setOpenTabs((prev) => (prev.includes(key) ? prev : [...prev, key]));
  };

  const handleTabClose = (tabKey: string) => {
    const key = tabKey as MenuKey;
    setOpenTabs((prev) => {
      if (prev.length <= 1) return prev;
      const idx = prev.indexOf(key);
      if (idx < 0) return prev;
      const nextTabs = prev.filter((k) => k !== key);
      setActiveMenu((am) => {
        if (am !== key) return am;
        const fallback = nextTabs[Math.max(0, idx - 1)] ?? nextTabs[0];
        setHashForMenu(fallback);
        return fallback;
      });
      return nextTabs;
    });
  };

  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <div style={{ display: 'flex', height: '100vh', background: '#F7F8FA' }}>
      {/* Sidebar */}
      <div
        style={{
          width: isCollapsed ? 60 : 240,
          background: '#fff',
          borderRight: '1px solid rgba(6,7,9,0.08)',
          display: 'flex',
          flexDirection: 'column',
          transition: 'width 0.2s',
        }}
      >
        {/* Logo Area */}
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            padding: isCollapsed ? '0 16px' : '0 24px',
            borderBottom: '1px solid rgba(6,7,9,0.08)',
            justifyContent: isCollapsed ? 'center' : 'flex-start',
            overflow: 'hidden',
          }}
        >
          {isCollapsed ? (
            <span style={{ fontSize: 24 }}>⚡</span>
          ) : (
            <Typography.Title
              heading={5}
              style={{ margin: 0, color: '#1C2029', whiteSpace: 'nowrap' }}
            >
              Flowgram
            </Typography.Title>
          )}
        </div>

        {/* Nav */}
        <div style={{ flex: 1, padding: '12px 0', overflowY: 'auto' }}>
          <Nav
            mode="vertical"
            isCollapsed={isCollapsed}
            items={[
              { itemKey: 'intro', text: 'Code 助手', icon: <IconHome /> },
              { itemKey: 'agent-profiles', text: 'Agent 配置', icon: <IconUser /> },
              { itemKey: 'agent-playground', text: 'Agent Playground', icon: <IconBranch /> },
              { itemKey: 'workspace-manager', text: '工作区管理', icon: <IconList /> },
              { itemKey: 'admin-terminal', text: '终端', icon: <IconDesktop /> },
              { itemKey: 'admin-lark-cli', text: '飞书 CLI 配置', icon: <IconSetting /> },
              {
                text: '工作流引擎',
                itemKey: 'engine',
                icon: <IconList />,
                items: [
                  { itemKey: 'workflow', text: '流程管理' },
                  { itemKey: 'workflow-run', text: '工作流执行' },
                  { itemKey: 'workflow-logs', text: '执行日志' },
                  {
                    text: '组件管理',
                    itemKey: 'component',
                    items: [
                      { itemKey: 'component-installed', text: '已安装组件' },
                      { itemKey: 'component-rules', text: '组件规则' },
                    ],
                  },
                ],
              },
              {
                text: '模型与工具',
                itemKey: 'agent',
                icon: <IconList />,
                items: [
                  { itemKey: 'agent-skills', text: 'SKILL 管理' },
                  { itemKey: 'agent-models', text: '模型管理' },
                  { itemKey: 'agent-mcp', text: 'MCP 配置' },
                ],
              },
            ]}
            selectedKeys={[activeMenu]}
            defaultOpenKeys={['engine', 'component', 'agent']}
            onSelect={(data) => {
              const key = data.itemKey as MenuKey;
              if (key === 'engine' || key === 'component' || key === 'agent') return;
              openMenu(key);
            }}
            style={{ background: 'transparent' }}
          />
        </div>

        {/* Footer Collapse Button */}
        <div
          style={{
            padding: '12px 0',
            borderTop: '1px solid rgba(6,7,9,0.08)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: isCollapsed ? 'center' : 'flex-start',
            paddingLeft: isCollapsed ? 0 : 24,
            cursor: 'pointer',
            color: '#1C2029',
            gap: 12,
          }}
          onClick={() => setIsCollapsed(!isCollapsed)}
        >
          {isCollapsed ? <IconChevronRight /> : <IconChevronLeft />}
          {!isCollapsed && (
            <Typography.Text style={{ userSelect: 'none' }}>收起导航</Typography.Text>
          )}
        </div>
      </div>

      {/* Main Content Area */}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          background: '#F7F8FA',
          overflow: 'hidden',
        }}
      >
        {/* Header */}
        <div
          style={{
            height: 64,
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: '1px solid rgba(6,7,9,0.08)',
          }}
        >
          <Breadcrumb>
            <Breadcrumb.Item>{getParentTitle()}</Breadcrumb.Item>
            <Breadcrumb.Item>{getPageTitle()}</Breadcrumb.Item>
          </Breadcrumb>

          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Typography.Text strong>Flowgram Team</Typography.Text>
            <div style={{ height: 16, width: 1, background: '#E5E6EB' }} />
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
              <IconUser />
              <Typography.Text>Admin</Typography.Text>
            </div>
          </div>
        </div>

        {/* Tabs + 各页内容（keepDOM 保留未焦点页状态） */}
        <div
          style={{
            flex: 1,
            minHeight: 0,
            display: 'flex',
            flexDirection: 'column',
            background: '#fff',
            padding: '6px 12px 0',
          }}
        >
          <Tabs
            type="card"
            collapsible
            activeKey={activeMenu}
            onChange={(key) => {
              if (MENU_KEYS.includes(key as MenuKey)) openMenu(key as MenuKey);
            }}
            onTabClose={handleTabClose}
            tabBarExtraContent={null}
            style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
            contentStyle={{
              flex: 1,
              minHeight: 0,
              overflow: 'auto',
              padding: 0,
              background: '#F7F8FA',
            }}
            keepDOM
          >
            {openTabs.map((tabKey) => (
              <TabPane
                key={tabKey}
                itemKey={tabKey}
                tab={getPageTitle(tabKey)}
                closable={openTabs.length > 1}
              >
                <div style={{ flex: 1, minHeight: '100%', padding: 0 }}>{renderPage(tabKey)}</div>
              </TabPane>
            ))}
          </Tabs>
        </div>
      </div>
    </div>
  );
};

export default AdminPanel;
