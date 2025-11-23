/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { useEffect } from 'react';

import {
  useRefresh,
  useService,
  GlobalScope,
  WorkflowDocument,
  WorkflowNodeEntity,
  BaseVariableField,
} from '@flowgram.ai/free-layout-editor';
import { useVariableTree, JsonSchemaUtils } from '@flowgram.ai/form-materials';
import { Tree } from '@douyinfe/semi-ui';

import { getNodeTypeName } from '../../../nodes/node-type-names';

export function FullVariableList() {
  const refresh = useRefresh();
  const document = useService(WorkflowDocument);
  const globalScope = useService(GlobalScope);

  useEffect(() => {
    let disposables: any[] = [];
    try {
      const d = (document as any).output?.onDocumentChange?.(() => refresh());
      if (d) disposables.push(d);
    } catch {}
    try {
      const d = (globalScope as any).output?.onVariableListChange?.(() => refresh());
      if (d) disposables.push(d);
    } catch {}
    return () => {
      disposables.forEach((d) => d?.dispose?.());
    };
  }, []);

  // useVariableTree will return the structure based on globalVar schema
  // Since registerNodeVariables already updates the globalVar schema with all nodes and their properties,
  // we should rely on useVariableTree to get the full correct tree structure.
  // The manual nodeItems construction below was creating a duplicate/mock structure that didn't reflect the actual registered schema.
  const treeData = useVariableTree({});
  const filteredTreeData = Array.isArray(treeData)
    ? treeData.filter((item) => item.label === 'Global')
    : treeData;

  return (
    <Tree
      treeData={filteredTreeData}
      onSelect={(keys, info: any) => {
        try {
          const label: string = String(info?.node?.label ?? '');
          if (!label) return;
          const globalVar = globalScope.getVar() as BaseVariableField;
          if (!globalVar) return;
          const schema = globalVar.type
            ? JsonSchemaUtils.astToSchema(globalVar.type)
            : ({ type: 'object', properties: {} } as any);
          const props = (schema as any).properties || ((schema as any).properties = {});
          if (!props[label]) {
            props[label] = { type: 'string' } as any;
            globalVar.updateType(JsonSchemaUtils.schemaToAST(schema));
          }
        } catch {}
      }}
    />
  );
}
