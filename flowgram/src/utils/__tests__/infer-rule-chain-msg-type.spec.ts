import { describe, expect, it } from 'vitest';

import { parseRuleChainFlowgramFromConfiguration } from '../rule-chain-flowgram-dsl';
import {
  DEFAULT_INFERRED_NOTIFY_MSG_TYPE,
  inferMsgTypeFromRuleDetail,
  looksLikeCronExpression,
} from '../infer-rule-chain-msg-type';

describe('inferMsgTypeFromRuleDetail', () => {
  it('uses flowgram entry_msg_type from configuration when parsed', () => {
    const configuration = {
      flowgram: {
        io: {
          request_metadata_params: [],
          request_message_body_params: [],
          response_message_body_params: [],
        },
        entry_msg_type: 'fromFlowgram',
      },
    };
    const fg = parseRuleChainFlowgramFromConfiguration(configuration);
    expect(
      inferMsgTypeFromRuleDetail({ ruleChain: { configuration }, metadata: {} }, fg.entryMsgType)
    ).toBe('fromFlowgram');
  });

  it('uses flowgram entryMsgType when set', () => {
    expect(
      inferMsgTypeFromRuleDetail(
        { ruleChain: { configuration: {} }, metadata: {} },
        'myEvent'
      )
    ).toBe('myEvent');
  });

  it('uses additionalInfo.msgType', () => {
    expect(
      inferMsgTypeFromRuleDetail({
        ruleChain: { additionalInfo: { msgType: 'fromAdd' } },
        metadata: {},
      })
    ).toBe('fromAdd');
  });

  it('infers last segment from HTTP endpoint router path', () => {
    expect(
      inferMsgTypeFromRuleDetail({
        ruleChain: {},
        metadata: {
          endpoints: [
            {
              type: 'endpoint/rest',
              routers: [{ from: { path: '/api/v1/hooks/github' }, to: {} }],
            },
          ],
        },
      })
    ).toBe('github');
  });

  it('skips schedule endpoint cron path', () => {
    expect(
      inferMsgTypeFromRuleDetail({
        ruleChain: {},
        metadata: {
          endpoints: [
            {
              type: 'endpoint/schedule',
              routers: [{ from: { path: '*/10 * * * * *' }, to: {} }],
            },
          ],
        },
      })
    ).toBe(DEFAULT_INFERRED_NOTIFY_MSG_TYPE);
  });

  it('defaults to CHAIN', () => {
    expect(inferMsgTypeFromRuleDetail({ ruleChain: {}, metadata: {} })).toBe(
      DEFAULT_INFERRED_NOTIFY_MSG_TYPE
    );
  });
});

describe('looksLikeCronExpression', () => {
  it('detects six-field cron', () => {
    expect(looksLikeCronExpression('*/10 * * * * *')).toBe(true);
  });

  it('does not flag URL path', () => {
    expect(looksLikeCronExpression('/webhook/feishu')).toBe(false);
  });
});
