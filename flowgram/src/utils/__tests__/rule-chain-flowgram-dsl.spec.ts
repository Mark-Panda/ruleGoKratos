import { describe, expect, it } from 'vitest';

import { buildRuleChainConfigurationWithFlowgram } from '../rule-chain-flowgram-dsl';

describe('buildRuleChainConfigurationWithFlowgram', () => {
  it('preserves existing flowgram skill metadata when saving known fields', () => {
    const next = buildRuleChainConfigurationWithFlowgram(
      {
        flowgram: {
          schema_version: 1,
          description: 'old',
          io: {
            request_metadata_params: [{ name: 'tenant' }],
            request_message_body_params: [{ name: 'query' }],
            response_message_body_params: [{ name: 'answer' }],
          },
          editor: {
            scratch_json: '{"nodes":[]}',
          },
          skill: {
            dir_name: 'weather-agent',
            signature: 'sig-123',
            generated_at: '2026-04-24T10:00:00Z',
            generated_by_managed_agent_id: 7,
            skill_entry_file: 'SKILL.md',
            last_error: 'boom',
          },
        },
      },
      {
        description: 'new description',
        requestMetadataParamsJson: '[{"name":"tenant"}]',
        requestMessageBodyParamsJson: '[{"name":"city"}]',
        responseMessageBodyParamsJson: '[{"name":"forecast"}]',
        editorScratchJson: '{"nodes":[1]}',
        skillDirName: 'weather-agent-v2',
      }
    );

    expect(next.flowgram).toMatchObject({
      description: 'new description',
      skill: {
        dir_name: 'weather-agent-v2',
        signature: 'sig-123',
        generated_at: '2026-04-24T10:00:00Z',
        generated_by_managed_agent_id: 7,
        skill_entry_file: 'SKILL.md',
        last_error: 'boom',
      },
    });
  });
});
