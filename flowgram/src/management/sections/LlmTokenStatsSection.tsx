import React, { useState, useEffect, useMemo } from 'react';

import {
  Card,
  Table,
  Select,
  DatePicker,
  Typography,
  Spin,
  Toast,
  Space,
  Button,
} from '@douyinfe/semi-ui';
import { IconRefresh } from '@douyinfe/semi-icons';

const { Text } = Typography;

interface TokenStatItem {
  period: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  requestCount: number;
}

interface TokenUsageItem {
  id: number;
  configId: number;
  modelEntryId: number;
  sessionId: string;
  requestId: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  modelName: string;
  actionType: string;
  userId: string;
  projectPath: string;
  createdAt: string;
}

interface StatsResponse {
  items: TokenStatItem[];
  totalPromptTokens: number;
  totalCompletionTokens: number;
  totalTokens: number;
  totalRequests: number;
}

interface UsageResponse {
  items: TokenUsageItem[];
  total: number;
}

export const LlmTokenStatsSection: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [usageList, setUsageList] = useState<TokenUsageItem[]>([]);
  const [total, setTotal] = useState(0);
  const [dateRange, setDateRange] = useState<[string, string] | null>(null);
  const [groupBy, setGroupBy] = useState<string>('day');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('group_by', groupBy);
      // 不传日期，让它返回所有数据
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.append('start_date', dateRange[0]);
        params.append('end_date', dateRange[1]);
      }

      const url = `/api/v1/admin/llm-token/stats?${params.toString()}`;
      console.log('Fetching stats:', url);
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
      });
      console.log('Stats response status:', resp.status);
      if (resp.ok) {
        const data: StatsResponse = await resp.json();
        console.log('Stats response data:', JSON.stringify(data));
        setStats(data);
      } else {
        const errText = await resp.text();
        console.error('Stats error:', errText);
        Toast.error('获取统计数据失败');
      }
    } catch (e) {
      console.error('Stats exception:', e);
      console.error('Stats exception:', e);
      Toast.error('获取统计数据失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchUsageList = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.append('page', String(page));
      params.append('page_size', String(pageSize));
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.append('start_date', dateRange[0]);
        params.append('end_date', dateRange[1]);
      }

      const url = `/api/v1/admin/llm-token/usage?${params.toString()}`;
      console.log('Fetching usage:', url);
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
      });
      if (resp.ok) {
        const data: UsageResponse = await resp.json();
        console.log('Usage response:', data);
        setUsageList(data.items || []);
        setTotal(data.total || 0);
      } else {
        const errText = await resp.text();
        console.error('Usage error:', errText);
        Toast.error('获取使用明细失败');
      }
    } catch (e) {
      console.error('Usage exception:', e);
      Toast.error('获取使用明细失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStats();
    fetchUsageList();
  }, [dateRange, groupBy, page, pageSize]);

  const columns = useMemo(
    () => [
      {
        title: '时间',
        dataIndex: 'period',
        width: 120,
      },
      {
        title: '模型',
        dataIndex: 'modelName',
        width: 150,
      },
      {
        title: 'Prompt Tokens',
        dataIndex: 'promptTokens',
        width: 120,
        render: (v: number) => v.toLocaleString(),
      },
      {
        title: 'Completion Tokens',
        dataIndex: 'completionTokens',
        width: 140,
        render: (v: number) => v.toLocaleString(),
      },
      {
        title: 'Total Tokens',
        dataIndex: 'totalTokens',
        width: 120,
        render: (v: number) => v.toLocaleString(),
      },
      {
        title: '请求次数',
        dataIndex: 'requestCount',
        width: 100,
        render: (v: number) => v.toLocaleString(),
      },
    ],
    []
  );

  const usageColumns = useMemo(
    () => [
      {
        title: '时间',
        dataIndex: 'createdAt',
        width: 160,
        render: (v: string) => new Date(v).toLocaleString(),
      },
      {
        title: '模型',
        dataIndex: 'modelName',
        width: 150,
      },
      {
        title: 'Prompt',
        dataIndex: 'promptTokens',
        width: 100,
        render: (v: number) => v.toLocaleString(),
      },
      {
        title: 'Completion',
        dataIndex: 'completionTokens',
        width: 100,
        render: (v: number) => v.toLocaleString(),
      },
      {
        title: 'Total',
        dataIndex: 'totalTokens',
        width: 100,
        render: (v: number) => v.toLocaleString(),
      },
      {
        title: '用户',
        dataIndex: 'userId',
        width: 120,
      },
      {
        title: '项目',
        dataIndex: 'projectPath',
        ellipsis: true,
      },
    ],
    []
  );

  return (
    <div style={{ padding: 16, height: '100%', overflowY: 'auto' }}>
      <Space vertical align="start" style={{ width: '100%' }}>
        <Card style={{ width: '100%' }}>
          <Space vertical align="start" style={{ width: '100%' }}>
            <Text strong style={{ fontSize: 16 }}>
              Token 使用统计
            </Text>
            <Space>
              <DatePicker
                onChange={(date: any) => {
                  if (date && Array.isArray(date) && date.length === 2) {
                    setDateRange([date[0].toString(), date[1].toString()]);
                  } else {
                    setDateRange(null);
                  }
                }}
                style={{ width: 240 }}
                type="dateRange"
              />
              <Select
                value={groupBy}
                onChange={(v) => setGroupBy(String(v))}
                style={{ width: 120 }}
              >
                <Select.Option value="day">按天</Select.Option>
                <Select.Option value="model">按模型</Select.Option>
              </Select>
              <Button
                icon={<IconRefresh />}
                onClick={() => {
                  setDateRange(null);
                  setGroupBy('day');
                  setPage(1);
                }}
              >
                重置
              </Button>
              <Button
                icon={<IconRefresh />}
                onClick={() => {
                  fetchStats();
                  fetchUsageList();
                }}
              >
                刷新
              </Button>
            </Space>
          </Space>
        </Card>

        {stats && (
          <Card style={{ width: '100%' }}>
            <Space spacing="loose">
              <div>
                <Text type="tertiary" style={{ fontSize: 12 }}>
                  Total Prompt Tokens
                </Text>
                <div style={{ fontSize: 24, fontWeight: 500 }}>
                  {stats.totalPromptTokens.toLocaleString()}
                </div>
              </div>
              <div>
                <Text type="tertiary" style={{ fontSize: 12 }}>
                  Total Completion Tokens
                </Text>
                <div style={{ fontSize: 24, fontWeight: 500 }}>
                  {stats.totalCompletionTokens.toLocaleString()}
                </div>
              </div>
              <div>
                <Text type="tertiary" style={{ fontSize: 12 }}>
                  Total Tokens
                </Text>
                <div style={{ fontSize: 24, fontWeight: 500 }}>
                  {stats.totalTokens.toLocaleString()}
                </div>
              </div>
              <div>
                <Text type="tertiary" style={{ fontSize: 12 }}>
                  总请求数
                </Text>
                <div style={{ fontSize: 24, fontWeight: 500 }}>
                  {stats.totalRequests.toLocaleString()}
                </div>
              </div>
            </Space>
          </Card>
        )}

        <Card style={{ width: '100%' }}>
          <Text strong style={{ fontSize: 14, marginBottom: 12, display: 'block' }}>
            统计图表
          </Text>
          <Spin spinning={loading}>
            <Table
              columns={columns}
              dataSource={stats?.items || []}
              rowKey="period"
              pagination={false}
              scroll={{ y: 200 }}
              style={{ width: '100%' }}
            />
          </Spin>
        </Card>

        <Card style={{ width: '100%' }}>
          <Text strong style={{ fontSize: 14, marginBottom: 12, display: 'block' }}>
            使用明细
          </Text>
          <Spin spinning={loading}>
            <Table
              columns={usageColumns}
              dataSource={usageList}
              rowKey="id"
              pagination={{
                pageSize,
                total,
                onPageChange: (p) => setPage(p),
                onPageSizeChange: (ps) => setPageSize(ps),
              }}
              scroll={{ y: 300 }}
              style={{ width: '100%' }}
            />
          </Spin>
        </Card>
      </Space>
    </div>
  );
};
