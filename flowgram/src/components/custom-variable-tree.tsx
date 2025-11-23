import { useCallback, useEffect, useState } from 'react';

import { debounce } from 'lodash-es';
import { useVariableTree } from '@flowgram.ai/form-materials';
import {
  Mention,
  MentionOpenChangeEvent,
  getCurrentMentionReplaceRange,
  useEditor,
  PositionMirror,
} from '@flowgram.ai/coze-editor/react';
import { EditorAPI } from '@flowgram.ai/coze-editor/preset-prompt';
import { Popover, Tree } from '@douyinfe/semi-ui';

const DEFAULT_TRIGGER_CHARACTER = ['{', '{}', '@', '${'] as const;

export function CustomVariableTree({
  triggerCharacters = DEFAULT_TRIGGER_CHARACTER as unknown as string[],
}: {
  triggerCharacters?: string[];
}) {
  const [posKey, setPosKey] = useState('');
  const [visible, setVisible] = useState(false);
  const [position, setPosition] = useState(-1);
  const editor = useEditor<EditorAPI>();

  function insert(variablePath: string) {
    const range = getCurrentMentionReplaceRange(editor.$view.state);
    if (!range) return;

    let { from, to } = range;
    while (editor.$view.state.doc.sliceString(from - 1, from) === '{') from--;
    while (editor.$view.state.doc.sliceString(from - 1, from) === '$') from--;
    while (editor.$view.state.doc.sliceString(to, to + 1) === '}') to++;

    editor.replaceText({ from, to, text: '${' + variablePath + '}' });
    setVisible(false);
  }

  function handleOpenChange(e: MentionOpenChangeEvent) {
    setPosition(e.state.selection.main.head);
    setVisible(e.value);
  }

  useEffect(() => {
    if (!editor) return;
  }, [editor, visible]);

  const treeData = useVariableTree({});

  const debounceUpdatePosKey = useCallback(
    debounce(() => setPosKey(String(Math.random())), 100),
    []
  );

  return (
    <>
      <Mention triggerCharacters={triggerCharacters} onOpenChange={handleOpenChange} />
      <Popover
        visible={visible}
        trigger="custom"
        position="topLeft"
        rePosKey={posKey}
        content={
          <div style={{ width: 300, maxHeight: 300, overflowY: 'auto' }}>
            <Tree
              treeData={treeData as any}
              onExpand={() => debounceUpdatePosKey()}
              onSelect={(v) => insert(String(Array.isArray(v) ? v[0] : v))}
            />
          </div>
        }
      >
        <PositionMirror position={position} onChange={() => setPosKey(String(Math.random()))} />
      </Popover>
    </>
  );
}
