/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FC, MouseEvent, useState } from 'react';

import classNames from 'classnames';
import { Tag, Toast } from '@douyinfe/semi-ui';
import { IconCopy, IconSmallTriangleDown } from '@douyinfe/semi-icons';

import { DataStructureViewer } from '../viewer';

import styles from './index.module.less';

interface NodeStatusGroupProps {
  title: string;
  data: unknown;
  optional?: boolean;
  disableCollapse?: boolean;
}

const isObjectHasContent = (obj: any = {}): boolean => obj && Object.keys(obj).length > 0;

export const NodeStatusGroup: FC<NodeStatusGroupProps> = ({
  title,
  data,
  optional = false,
  disableCollapse = false,
}) => {
  const hasContent = isObjectHasContent(data);
  const [isExpanded, setIsExpanded] = useState(true);

  const handleCopyGroupData = async (event: MouseEvent) => {
    event.stopPropagation();
    if (!hasContent) {
      return;
    }
    try {
      const text =
        typeof data === 'string' ? data : JSON.stringify(data, null, 2) ?? String(data ?? '');
      await navigator.clipboard.writeText(text);
      Toast.success({ content: `已复制${title}`, duration: 2 });
    } catch {
      Toast.warning({ content: '复制失败，请手动复制', duration: 3 });
    }
  };

  if (optional && !hasContent) {
    return null;
  }

  return (
    <>
      <div
        className={styles['node-status-group']}
        onClick={() => hasContent && !disableCollapse && setIsExpanded(!isExpanded)}
      >
        {!disableCollapse && (
          <IconSmallTriangleDown
            className={classNames(styles['node-status-group-icon'], {
              [styles['node-status-group-icon-expanded']]: isExpanded && hasContent,
            })}
          />
        )}
        <span>{title}:</span>
        {hasContent ? (
          <span className={styles['node-status-group-copy']} onClick={handleCopyGroupData}>
            <IconCopy className={styles['node-status-group-copy-icon']} />
            复制
          </span>
        ) : null}
        {!hasContent && (
          <Tag size="small" className={styles['node-status-group-tag']}>
            null
          </Tag>
        )}
      </div>
      {hasContent && isExpanded ? <DataStructureViewer data={data} /> : null}
    </>
  );
};
