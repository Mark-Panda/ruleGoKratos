/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import {
  ASTFactory,
  definePluginCreator,
  GlobalScope,
  VariableDeclaration,
  WorkflowDocument,
} from '@flowgram.ai/free-layout-editor';
import { IJsonSchema, JsonSchemaUtils } from '@flowgram.ai/form-materials';

import iconVariable from '../../assets/icon-variable.png';
import { VariablePanelLayer } from './variable-panel-layer';

const fetchMockVariableFromRemote = async () => {
  await new Promise((resolve) => setTimeout(resolve, 1000));
  return {
    type: 'object',
    properties: {
      userId: { type: 'string' },
    },
  };
};

export type GetGlobalVariableSchema = () => IJsonSchema;
export const GetGlobalVariableSchema = Symbol('GlobalVariableSchemaGetter');

export const createVariablePanelPlugin = definePluginCreator<{ initialData?: IJsonSchema }>({
  onBind({ bind }) {
    bind(GetGlobalVariableSchema).toDynamicValue((ctx) => () => {
      const variable = ctx.container.get(GlobalScope).getVar() as VariableDeclaration;
      return JsonSchemaUtils.astToSchema(variable?.type);
    });
  },
  onInit(ctx, opts) {
    ctx.playground.registerLayer(VariablePanelLayer);

    const globalScope = ctx.get(GlobalScope);
    const document: any = ctx.get(WorkflowDocument);

    if (opts.initialData) {
      globalScope.setVar(
        ASTFactory.createVariableDeclaration({
          key: 'global',
          meta: {
            title: 'Global',
            icon: iconVariable,
          },
          type: JsonSchemaUtils.schemaToAST(opts.initialData),
        })
      );
    } else {
      fetchMockVariableFromRemote().then((v) => {
        globalScope.setVar(
          ASTFactory.createVariableDeclaration({
            key: 'global',
            meta: {
              title: 'Global',
              icon: iconVariable,
            },
            type: JsonSchemaUtils.schemaToAST(v),
          })
        );
        try {
          registerNodeVariables();
        } catch {}
      });
    }

    const registerNodeVariables = () => {
      const globalVar = globalScope.getVar() as VariableDeclaration;
      if (!globalVar) {
        return;
      }
      const schemaFromAst = globalVar?.type
        ? JsonSchemaUtils.astToSchema(globalVar.type)
        : undefined;
      const baseSchema =
        schemaFromAst ?? ({ type: 'object', properties: {}, required: [] } as IJsonSchema);
      const nodeProps: Record<string, IJsonSchema> = {};
      document.getAllNodes().forEach((n: any) => {
        const id = String(n.id);
        const title = document.toNodeJSON(n)?.data?.title || id;
        nodeProps[id] = {
          type: 'object',
          title,
          properties: {
            id: { type: 'string' },
            ts: { type: 'string' },
            data: { type: 'object' },
            msg: { type: 'object' },
            metadata: { type: 'object' },
            msgType: { type: 'string' },
            dataType: { type: 'string' },
          },
        } as any;
      });
      const nextSchema: IJsonSchema = {
        type: 'object',
        required: Array.isArray(baseSchema?.required) ? (baseSchema as any).required : [],
        properties: {
          ...((baseSchema && (baseSchema as any).properties) || {}),
          nodes: { type: 'object', properties: nodeProps },
        },
      } as any;
      globalVar.updateType(JsonSchemaUtils.schemaToAST(nextSchema));

      try {
        document.getAllNodes().forEach((n: any) => {
          const id = String(n.id);
          const title = document.toNodeJSON(n)?.data?.title || id;
          const field = globalVar.getByKeyPath(['nodes', id]);
          if (field) {
            field.updateMeta({ ...(field.meta || {}), title });
          }
        });
      } catch {}
    };

    try {
      registerNodeVariables();
    } catch {}

    try {
      document.output.onDocumentChange(registerNodeVariables);
    } catch {}

    try {
      // 当节点被创建时，立即合并到 global.nodes
      const disposeCreate = document.onNodeCreate(() => registerNodeVariables());
      // 当画布销毁时清理监听
      (ctx.playground as any).toDispose?.add?.(disposeCreate);
    } catch {}

    try {
      globalScope.output.onVariableListChange(registerNodeVariables);
    } catch {}
  },
});
