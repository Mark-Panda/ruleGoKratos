import { useState } from 'react';

import { useVariableTree } from '@flowgram.ai/form-materials';
import { Button, Modal, Tree } from '@douyinfe/semi-ui';

import iconVariable from '../assets/icon-variable.png';

interface VariablePickerProps {
  disabled?: boolean;
  onInsert: (text: string) => void;
  size?: 'small' | 'default';
}

export function VariablePicker({ disabled, onInsert, size }: VariablePickerProps) {
  const treeData = useVariableTree({});

  const [open, setOpen] = useState(false);

  const normalize = (key: string) => {
    if (key.startsWith('global.nodes.')) {
      return key.replace(/^global\.nodes\./, '');
    }
    return key;
  };
  const toPath = (key: string) =>
    `
${'${' + normalize(key) + '}'}
`.trim();

  return (
    <>
      <Button
        size={size || 'small'}
        type="tertiary"
        theme="light"
        icon={<img src={iconVariable} width={16} height={16} />}
        disabled={disabled}
        onClick={() => setOpen(true)}
      />
      <Modal visible={open} footer={null} onCancel={() => setOpen(false)} title="选择变量">
        <Tree
          treeData={treeData as any}
          onSelect={(keys) => {
            const k = String(Array.isArray(keys) ? keys[0] : keys);
            if (!k || k === 'global') return;
            onInsert(toPath(k));
            setOpen(false);
          }}
        />
      </Modal>
    </>
  );
}
