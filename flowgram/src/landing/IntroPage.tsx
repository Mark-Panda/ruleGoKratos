/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React from 'react';

import { Typography, Card } from '@douyinfe/semi-ui';

export const IntroPage: React.FC = () => (
  <div
    style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(180deg, #ffffff 0%, #F7F8FA 100%)',
      padding: '48px 24px',
    }}
  >
    <Card
      style={{ maxWidth: 960, width: '100%', boxShadow: '0 8px 24px rgba(0,0,0,0.06)' }}
      header={<Typography.Title heading={3}>Flowgram 工作流服务</Typography.Title>}
    >
      <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: 16 }}>
        <Typography.Paragraph spacing="extended">
          这是一个可视化工作流编排与运行的演示服务，支持节点式设计、表单化配置、运行态管理与结果查看。
        </Typography.Paragraph>
        <Typography.Title heading={5} style={{ marginTop: 8 }}>
          核心能力
        </Typography.Title>
        <ul style={{ margin: 0, paddingLeft: 18, color: 'rgba(6,7,9,0.75)' }}>
          <li>自由布局编辑器：拖拽节点、连线、分组与吸附</li>
          <li>运行态：工作流部署、启动、下线与结果查询</li>
          <li>组件化：常用节点与业务组件的统一管理</li>
          <li>文档中心：业务流程说明与使用指引</li>
        </ul>
      </div>
    </Card>
  </div>
);

export default IntroPage;
