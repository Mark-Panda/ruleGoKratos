/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useEffect, useState } from 'react';

import {
  AlarmClock,
  Api,
  ApplicationOne,
  CompassOne,
  ConnectionPoint,
  DataSheet,
  Histogram,
  ListView,
  PlayCycle,
  RobotOne,
  SettingConfig,
  Terminal,
} from '@icon-park/react';
import { Nav, Typography, Breadcrumb, Tabs, TabPane } from '@douyinfe/semi-ui';
import { IconUser, IconChevronLeft, IconChevronRight } from '@douyinfe/semi-icons';

import { WorkspacesSection } from './sections/WorkspacesSection';
import { WorkflowSection } from './sections/WorkflowSection';
import { WorkflowRunLogsSection } from './sections/WorkflowRunLogsSection';
import { WorkflowExecuteSection } from './sections/WorkflowExecuteSection';
import { TerminalSection } from './sections/TerminalSection';
import { TaskBoardSection } from './sections/TaskBoardSection';
import { ServiceManagementSection } from './sections/ServiceManagementSection';
import { ScheduledTaskSection } from './sections/ScheduledTaskSection';
import { OverviewChatSection } from './sections/OverviewChatSection';
import { ManagedAgentsSection } from './sections/ManagedAgentsSection';
import { LlmTokenStatsSection } from './sections/LlmTokenStatsSection';
import { LarkCliSection } from './sections/LarkCliSection';
import { CursorCliSection } from './sections/CursorCliSection';
import { ComponentsSection } from './sections/ComponentsSection';
import { AgentSection } from './sections/AgentSection';
import { AgentPlaygroundPage } from '../agent-playground';

const navIconProps = {
  theme: 'outline' as const,
  size: 16,
  strokeWidth: 2.4,
};

const subNavIconProps = {
  ...navIconProps,
  size: 14,
  strokeWidth: 2.2,
};

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
  | 'admin-cli'
  | 'admin-lark-cli'
  | 'admin-cursor-cli'
  | 'engine'
  | 'component'
  | 'agent'
  | 'task-board'
  | 'service-management'
  | 'scheduled-tasks'
  | 'llm-token-stats'
  | 'business';

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
  'admin-cursor-cli',
  'task-board',
  'service-management',
  'scheduled-tasks',
  'llm-token-stats',
];

function getMenuFromHash(h: string): MenuKey {
  if (h === '#/' || h === '' || h === '#') return 'intro';
  if (h.startsWith('#/terminal')) return 'admin-terminal';
  if (h.startsWith('#/lark-cli')) return 'admin-lark-cli';
  if (h.startsWith('#/cursor-cli')) return 'admin-cursor-cli';
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
  if (h.startsWith('#/task-board')) return 'task-board';
  if (h.startsWith('#/service-management')) return 'service-management';
  if (h.startsWith('#/scheduled-tasks')) return 'scheduled-tasks';
  if (h.startsWith('#/llm-token-stats')) return 'llm-token-stats';
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
  else if (key === 'admin-cursor-cli') window.location.hash = '#/cursor-cli';
  else if (key === 'task-board') window.location.hash = '#/task-board';
  else if (key === 'service-management') window.location.hash = '#/service-management';
  else if (key === 'scheduled-tasks') window.location.hash = '#/scheduled-tasks';
  else if (key === 'llm-token-stats') window.location.hash = '#/llm-token-stats';
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
    if (key === 'admin-cursor-cli') return <CursorCliSection />;
    if (key === 'component-rules') return <ComponentsSection view="rules" />;
    if (key === 'task-board') return <TaskBoardSection />;
    if (key === 'service-management') return <ServiceManagementSection />;
    if (key === 'scheduled-tasks') return <ScheduledTaskSection />;
    if (key === 'llm-token-stats') return <LlmTokenStatsSection />;
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
        return 'Playground';
      case 'agent-profiles':
        return 'Agent 配置';
      case 'workspace-manager':
        return '工作区管理';
      case 'admin-terminal':
        return '终端';
      case 'admin-lark-cli':
        return '飞书';
      case 'admin-cursor-cli':
        return 'Cursor';
      case 'task-board':
        return '任务看板';
      case 'service-management':
        return '服务列表';
      case 'scheduled-tasks':
        return '定时任务';
      case 'llm-token-stats':
        return 'Token 统计';
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
      activeMenu === 'agent-models' ||
      activeMenu === 'agent-profiles' ||
      activeMenu === 'workspace-manager' ||
      activeMenu === 'llm-token-stats'
    )
      return '模型与工具';
    if (activeMenu === 'task-board' || activeMenu === 'service-management') return '业务管理';
    if (activeMenu === 'component-installed' || activeMenu === 'component-rules') return '组件管理';
    if (activeMenu === 'intro') return '工作台';
    if (activeMenu === 'agent-playground') return '智能体';
    if (activeMenu === 'admin-lark-cli' || activeMenu === 'admin-cursor-cli') return 'CLI 配置';
    if (activeMenu === 'admin-terminal' || activeMenu === 'scheduled-tasks') return '运维';
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
              BaBo Flow
            </Typography.Title>
          )}
        </div>

        {/* Nav */}
        <div style={{ flex: 1, padding: '12px 0', overflowY: 'auto' }}>
          <Nav
            mode="vertical"
            isCollapsed={isCollapsed}
            items={[
              { itemKey: 'intro', text: 'Code 助手', icon: <ApplicationOne {...navIconProps} /> },
              {
                itemKey: 'agent-playground',
                text: 'Playground',
                icon: <PlayCycle {...navIconProps} />,
              },
              {
                itemKey: 'scheduled-tasks',
                text: '定时任务',
                icon: <AlarmClock {...navIconProps} />,
              },
              { itemKey: 'admin-terminal', text: '终端', icon: <Terminal {...navIconProps} /> },
              {
                text: 'CLI 配置',
                itemKey: 'admin-cli',
                items: [
                  { itemKey: 'admin-lark-cli', text: '飞书', icon: <Api {...subNavIconProps} /> },
                  {
                    itemKey: 'admin-cursor-cli',
                    text: 'Cursor',
                    icon: <CompassOne {...subNavIconProps} />,
                  },
                ],
              },
              {
                text: '工作流引擎',
                itemKey: 'engine',
                // 组件管理（已安装组件 / 组件规则）已藏起入口，路由与 renderPage 仍保留，可通过 URL 访问
                items: [
                  {
                    itemKey: 'workflow',
                    text: '流程管理',
                    icon: <ConnectionPoint {...subNavIconProps} />,
                  },
                  {
                    itemKey: 'workflow-run',
                    text: '工作流执行',
                    icon: <PlayCycle {...subNavIconProps} />,
                  },
                  {
                    itemKey: 'workflow-logs',
                    text: '执行日志',
                    icon: <Histogram {...subNavIconProps} />,
                  },
                ],
              },
              {
                text: '业务管理',
                itemKey: 'business',
                items: [
                  {
                    itemKey: 'task-board',
                    text: '任务看板',
                    icon: <ListView {...subNavIconProps} />,
                  },
                  {
                    itemKey: 'service-management',
                    text: '服务列表',
                    icon: <SettingConfig {...subNavIconProps} />,
                  },
                ],
              },
              {
                text: '模型与工具',
                itemKey: 'agent',
                items: [
                  {
                    itemKey: 'agent-skills',
                    text: 'SKILL 管理',
                    icon: <DataSheet {...subNavIconProps} />,
                  },
                  {
                    itemKey: 'agent-models',
                    text: '模型管理',
                    icon: <ApplicationOne {...subNavIconProps} />,
                  },
                  { itemKey: 'agent-mcp', text: 'MCP 配置', icon: <Api {...subNavIconProps} /> },
                  {
                    itemKey: 'agent-profiles',
                    text: 'Agent 配置',
                    icon: <RobotOne {...subNavIconProps} />,
                  },
                  {
                    itemKey: 'workspace-manager',
                    text: '工作区管理',
                    icon: <DataSheet {...subNavIconProps} />,
                  },
                  {
                    itemKey: 'llm-token-stats',
                    text: 'Token 统计',
                    icon: <Histogram {...subNavIconProps} />,
                  },
                ],
              },
            ]}
            selectedKeys={[activeMenu]}
            defaultOpenKeys={['engine', 'agent', 'admin-cli', 'business']}
            onSelect={(data) => {
              const key = data.itemKey as MenuKey;
              if (
                key === 'engine' ||
                key === 'component' ||
                key === 'agent' ||
                key === 'admin-cli' ||
                key === 'business'
              )
                return;
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
            {/* <Typography.Text strong>BaBo Flow</Typography.Text> */}
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
            className="admin-shell-tabs"
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
              height: '100%',
              minHeight: 0,
              overflow: 'hidden',
              padding: 0,
              background: '#F7F8FA',
              display: 'flex',
              flexDirection: 'column',
            }}
            keepDOM
          >
            {openTabs.map((tabKey) => (
              <TabPane
                key={tabKey}
                itemKey={tabKey}
                tab={getPageTitle(tabKey)}
                closable={openTabs.length > 1}
                style={{
                  // 勿在此写 display:flex：inline 会压过 Semi 的 .semi-tabs-pane-inactive{display:none}，导致 keepDOM 下所有标签页同时铺开
                  flex: '1 1 0%',
                  minHeight: 0,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    minHeight: 0,
                    display: 'flex',
                    flexDirection: 'column',
                    padding: 0,
                  }}
                >
                  <div
                    style={{ flex: 1, minHeight: 0, minWidth: 0, width: '100%', display: 'flex' }}
                  >
                    {renderPage(tabKey)}
                  </div>
                </div>
              </TabPane>
            ))}
          </Tabs>
        </div>
      </div>
    </div>
  );
};

export default AdminPanel;
