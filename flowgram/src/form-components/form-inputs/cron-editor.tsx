/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import React, { useState, useEffect } from 'react';

import { Input, InputNumber, Radio, RadioGroup, Typography } from '@douyinfe/semi-ui';

interface CronEditorProps {
  value?: any;
  onChange?: (value: any) => void;
  readonly?: boolean;
  hasError?: boolean;
}

type CronType = 'every-second' | 'every-minute' | 'every-hour' | 'every-day' | 'custom';

export const CronEditor: React.FC<CronEditorProps> = ({ value, onChange, readonly, hasError }) => {
  const content = typeof value?.content === 'string' ? value.content : '*/10 * * * * *';
  const [cronType, setCronType] = useState<CronType>('custom');
  const [customCron, setCustomCron] = useState(content);

  // 间隔值
  const [secondInterval, setSecondInterval] = useState(10);
  const [minuteInterval, setMinuteInterval] = useState(10);
  const [hourInterval, setHourInterval] = useState(1);

  // 解析 Cron 表达式
  useEffect(() => {
    const parts = content.split(' ');
    if (parts.length === 6) {
      // 判断类型
      if (content.match(/^\*\/\d+ \* \* \* \* \*$/)) {
        setCronType('every-second');
        const match = content.match(/^\*\/(\d+)/);
        if (match) setSecondInterval(parseInt(match[1]));
      } else if (content.match(/^\* \*\/\d+ \* \* \* \*$/)) {
        setCronType('every-minute');
        const match = content.match(/^\* \*\/(\d+)/);
        if (match) setMinuteInterval(parseInt(match[1]));
      } else if (content.match(/^\* \* \*\/\d+ \* \* \*$/)) {
        setCronType('every-hour');
        const match = content.match(/^\* \* \*\/(\d+)/);
        if (match) setHourInterval(parseInt(match[1]));
      } else if (content === '0 0 0 * * *') {
        setCronType('every-day');
      } else {
        setCronType('custom');
        setCustomCron(content);
      }
    }
  }, [content]);

  const generateCron = (type: CronType) => {
    let cron = '';
    switch (type) {
      case 'every-second':
        cron = `*/${secondInterval} * * * * *`;
        break;
      case 'every-minute':
        cron = `* */${minuteInterval} * * * *`;
        break;
      case 'every-hour':
        cron = `* * */${hourInterval} * * *`;
        break;
      case 'every-day':
        cron = '0 0 0 * * *';
        break;
      case 'custom':
        cron = customCron;
        break;
    }
    return cron;
  };

  const handleTypeChange = (type: CronType) => {
    setCronType(type);
    const cron = generateCron(type);
    onChange?.({ type: 'constant', content: cron });
  };

  const handleIntervalChange = (
    intervalType: 'second' | 'minute' | 'hour',
    val: number | string
  ) => {
    const numVal = typeof val === 'number' ? val : parseInt(val) || 1;
    if (intervalType === 'second') {
      setSecondInterval(numVal);
      if (cronType === 'every-second') {
        onChange?.({ type: 'constant', content: `*/${numVal} * * * * *` });
      }
    } else if (intervalType === 'minute') {
      setMinuteInterval(numVal);
      if (cronType === 'every-minute') {
        onChange?.({ type: 'constant', content: `* */${numVal} * * * *` });
      }
    } else if (intervalType === 'hour') {
      setHourInterval(numVal);
      if (cronType === 'every-hour') {
        onChange?.({ type: 'constant', content: `* * */${numVal} * * *` });
      }
    }
  };

  const handleCustomChange = (val: string) => {
    setCustomCron(val);
    if (cronType === 'custom') {
      onChange?.({ type: 'constant', content: val });
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <RadioGroup
        type="button"
        value={cronType}
        onChange={(e) => handleTypeChange(e.target.value as CronType)}
        disabled={readonly}
        style={{ width: '100%' }}
      >
        <Radio value="every-second" style={{ flex: 1 }}>
          每N秒
        </Radio>
        <Radio value="every-minute" style={{ flex: 1 }}>
          每N分钟
        </Radio>
        <Radio value="every-hour" style={{ flex: 1 }}>
          每N小时
        </Radio>
        <Radio value="every-day" style={{ flex: 1 }}>
          每天
        </Radio>
        <Radio value="custom" style={{ flex: 1 }}>
          自定义
        </Radio>
      </RadioGroup>

      {cronType === 'every-second' && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '12px',
            background: '#f5f5f5',
            borderRadius: 8,
          }}
        >
          <Typography.Text>每</Typography.Text>
          <InputNumber
            value={secondInterval}
            onChange={(val) => handleIntervalChange('second', val as number)}
            min={1}
            max={59}
            disabled={readonly}
            style={{ width: 80 }}
          />
          <Typography.Text>秒执行一次</Typography.Text>
        </div>
      )}

      {cronType === 'every-minute' && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '12px',
            background: '#f5f5f5',
            borderRadius: 8,
          }}
        >
          <Typography.Text>每</Typography.Text>
          <InputNumber
            value={minuteInterval}
            onChange={(val) => handleIntervalChange('minute', val as number)}
            min={1}
            max={59}
            disabled={readonly}
            style={{ width: 80 }}
          />
          <Typography.Text>分钟执行一次</Typography.Text>
        </div>
      )}

      {cronType === 'every-hour' && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '12px',
            background: '#f5f5f5',
            borderRadius: 8,
          }}
        >
          <Typography.Text>每</Typography.Text>
          <InputNumber
            value={hourInterval}
            onChange={(val) => handleIntervalChange('hour', val as number)}
            min={1}
            max={23}
            disabled={readonly}
            style={{ width: 80 }}
          />
          <Typography.Text>小时执行一次</Typography.Text>
        </div>
      )}

      {cronType === 'every-day' && (
        <div style={{ padding: '12px', background: '#f5f5f5', borderRadius: 8 }}>
          <Typography.Text>每天 00:00:00 执行一次</Typography.Text>
        </div>
      )}

      {cronType === 'custom' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <Input
            value={customCron}
            onChange={handleCustomChange}
            placeholder="秒 分 时 日 月 周"
            disabled={readonly}
            style={{
              fontFamily: 'monospace',
              borderColor: hasError ? '#f53f3f' : undefined,
            }}
          />
          <Typography.Text type="tertiary" size="small">
            格式: 秒(0-59) 分(0-59) 时(0-23) 日(1-31) 周(0-6) 月(1-12)
          </Typography.Text>
        </div>
      )}

      <div
        style={{
          padding: '10px 12px',
          background: '#e8f4ff',
          borderRadius: 6,
          border: '1px solid #b3d8ff',
        }}
      >
        <Typography.Text strong style={{ fontSize: 12, color: '#0077cc' }}>
          当前表达式:{' '}
          <code
            style={{
              background: '#fff',
              padding: '2px 6px',
              borderRadius: 4,
              fontFamily: 'monospace',
              color: '#0077cc',
            }}
          >
            {generateCron(cronType)}
          </code>
        </Typography.Text>
      </div>

      <div
        style={{
          padding: '8px 12px',
          background: '#fff7e6',
          borderRadius: 6,
          border: '1px solid #ffd591',
        }}
      >
        <Typography.Text size="small" style={{ color: '#ad6800' }}>
          💡 提示: 使用 * 表示任意值, */N 表示每N个单位
        </Typography.Text>
      </div>
    </div>
  );
};
