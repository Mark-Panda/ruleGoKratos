import { FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <>
    <FormHeader />
    <FormContent>
      <FormInputs />
      <OutputsPeek />
    </FormContent>
  </>
);

export const sourcegraphTokenVerifyFormMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
