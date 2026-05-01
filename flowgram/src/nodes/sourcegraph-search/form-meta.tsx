import { Field, FormMeta, FormRenderProps } from '@flowgram.ai/free-layout-editor';

import { defaultFormMeta } from '../default-form-meta';
import { FlowNodeJSON } from '../../typings';
import { FormContent, FormHeader, FormInputs, OutputsPeek } from '../../form-components';

type RepoScope = '' | 'frontend' | 'backend';

function parseRepoScope(raw: unknown): RepoScope {
  if (raw === 'frontend' || raw === 'backend') return raw;
  return '';
}

function isPropertyVisible(repoScope: RepoScope, key: string): boolean {
  if (key === 'repoFrontend') return repoScope === 'frontend';
  if (key === 'repoBackend') return repoScope === 'backend';
  return true;
}

const renderForm = (_props: FormRenderProps<FlowNodeJSON>) => (
  <Field name="inputsValues.repoScope">
    {({ field }) => {
      const repoScope = parseRepoScope(
        (field.value as { content?: unknown } | undefined)?.content
      );
      return (
        <>
          <FormHeader />
          <FormContent>
            <FormInputs propertyFilter={(key) => isPropertyVisible(repoScope, key)} />
            <OutputsPeek />
          </FormContent>
        </>
      );
    }}
  </Field>
);

export const formMeta: FormMeta<FlowNodeJSON> = {
  ...defaultFormMeta,
  render: renderForm,
};
