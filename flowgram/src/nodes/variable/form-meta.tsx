/**
 * Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
 * SPDX-License-Identifier: MIT
 */

import { FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';
import { AssignRows, createInferAssignPlugin } from '@flowgram.ai/form-materials';

import { FormHeader, FormContent, OutputsPeek } from '../../form-components';
import { VariableNodeJSON } from './types';
import { defaultFormMeta } from '../default-form-meta';
import { useIsSidebar } from '../../hooks';

export const FormRender = ({ form }: FormRenderProps<VariableNodeJSON>) => {
  const isSidebar = useIsSidebar();

  return (
    <>
      <FormHeader />
      <FormContent>
        {isSidebar ? <AssignRows name="assign" /> : <OutputsPeek />}
      </FormContent>
    </>
  );
};

export const formMeta: FormMeta = {
  render: (props) => <FormRender {...props} />,
  effect: defaultFormMeta.effect,
  plugins: [
    createInferAssignPlugin({
      assignKey: 'assign',
      outputKey: 'outputs',
    }),
  ],
};
