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
  IconFile,
  IconChevronLeft,
  IconChevronRight,
} from '@douyinfe/semi-icons';

import { WorkflowSection } from './sections/WorkflowSection';
import { DocsSection } from './sections/DocsSection';
import { ComponentsSection } from './sections/ComponentsSection';
import { IntroPage } from '../landing/IntroPage';

type MenuKey =
  | 'intro'
  | 'workflow'
  | 'component-installed'
  | 'component-rules'
  | 'docs'
  | 'engine'
  | 'component';

export const AdminPanel: React.FC = () => {
  const [activeMenu, setActiveMenu] = useState<MenuKey>(() => {
    try {
      const h = typeof window !== 'undefined' ? window.location.hash : '';
      if (h === '#/' || h === '' || h === '#') return 'intro';
      if (h.startsWith('#/components/rules')) return 'component-rules';
      if (h.startsWith('#/components')) return 'component-installed';
      if (h.startsWith('#/docs')) return 'docs';
      return 'workflow';
    } catch {
      return 'intro';
    }
  });

  useEffect(() => {
    const getMenu = (h: string): MenuKey => {
      if (h === '#/' || h === '' || h === '#') return 'intro';
      if (h.startsWith('#/components/rules')) return 'component-rules';
      if (h.startsWith('#/components')) return 'component-installed';
      if (h.startsWith('#/docs')) return 'docs';
      return 'workflow';
    };
    const onHash = () => {
      try {
        const h = typeof window !== 'undefined' ? window.location.hash : '';
        setActiveMenu(getMenu(h || '#/'));
      } catch {}
    };
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const renderContent = () => {
    if (activeMenu === 'intro') return <IntroPage />;
    if (activeMenu === 'workflow') return <WorkflowSection />;
    if (activeMenu === 'docs') return <DocsSection />;
    if (activeMenu === 'component-rules') return <ComponentsSection view="rules" />;
    return <ComponentsSection view="installed" />;
  };

  const getPageTitle = () => {
    switch (activeMenu) {
      case 'intro':
        return '概览';
      case 'workflow':
        return '流程管理';
      case 'component-installed':
        return '已安装组件';
      case 'component-rules':
        return '组件规则';
      case 'docs':
        return '开发文档';
      default:
        return '概览';
    }
  };

  const getParentTitle = () => {
    if (activeMenu === 'workflow') return '工作流引擎';
    if (activeMenu === 'component-installed' || activeMenu === 'component-rules') return '组件管理';
    return '系统';
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
              { itemKey: 'intro', text: '概览', icon: <IconHome /> },
              {
                text: '工作流引擎',
                itemKey: 'engine',
                icon: <IconList />,
                items: [
                  { itemKey: 'workflow', text: '流程管理' },
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
              { itemKey: 'docs', text: '开发文档', icon: <IconFile /> },
            ]}
            selectedKeys={[activeMenu]}
            defaultOpenKeys={['engine', 'component']}
            onSelect={(data) => {
              const key = data.itemKey as MenuKey;
              if (key === 'engine' || key === 'component') return;
              setActiveMenu(key);
              if (key === 'intro') window.location.hash = '#/';
              if (key === 'workflow') window.location.hash = '#/admin';
              if (key === 'component-installed') window.location.hash = '#/components';
              if (key === 'component-rules') window.location.hash = '#/components/rules';
              if (key === 'docs') window.location.hash = '#/docs';
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

        {/* Tabs */}
        <div
          style={{
            padding: '6px 12px 0',
            background: '#fff',
            borderBottom: '1px solid rgba(6,7,9,0.08)',
          }}
        >
          <Tabs
            type="card"
            activeKey={activeMenu}
            onChange={(key) => {
              // Only handle tab clicks if they correspond to actual routes
              if (
                ['intro', 'workflow', 'component-installed', 'component-rules', 'docs'].includes(
                  key
                )
              ) {
                setActiveMenu(key as MenuKey);
                if (key === 'intro') window.location.hash = '#/';
                if (key === 'workflow') window.location.hash = '#/admin';
                if (key === 'component-installed') window.location.hash = '#/components';
                if (key === 'component-rules') window.location.hash = '#/components/rules';
                if (key === 'docs') window.location.hash = '#/docs';
              }
            }}
            tabBarExtraContent={null}
          >
            <TabPane tab="概览" itemKey="intro" />
            {activeMenu !== 'intro' && (
              <TabPane tab={getPageTitle()} itemKey={activeMenu} closable />
            )}
          </Tabs>
        </div>

        {/* Content Body */}
        <div style={{ flex: 1, overflow: 'auto', padding: 0 }}>{renderContent()}</div>
      </div>
    </div>
  );
};

export default AdminPanel;
