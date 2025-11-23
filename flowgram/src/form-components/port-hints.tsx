import styled from 'styled-components';

import { OutPutPortType } from '../nodes/constants';
import { useIsSidebar, useNodeRenderContext } from '../hooks';

const Hint = styled.div`
  position: absolute;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 10px;
  border: 1px solid rgba(6, 7, 9, 0.15);
  background: #fff;
  color: rgba(6, 7, 9, 0.65);
  font-size: 12px;
  pointer-events: none;
`;

const RightHint = styled(Hint)`
  right: -8px;
  top: 50%;
  transform: translate(100%, -50%);
`;

const BottomHint = styled(Hint)`
  left: 50%;
  bottom: -8px;
  transform: translate(-50%, 100%);
`;

export function PortHints() {
  const isSidebar = useIsSidebar();
  const { node } = useNodeRenderContext();
  if (isSidebar) return null;
  const ports = node.getNodeRegistry<any>()?.meta?.defaultPorts || [];
  const hasRightOutput = ports.some((p: any) => p?.type === 'output' && p?.location === 'right');
  const hasBottomOutput = ports.some((p: any) => p?.type === 'output' && p?.location === 'bottom');
  return (
    <>
      {hasRightOutput && <RightHint>{OutPutPortType.SuccessPort}</RightHint>}
      {hasBottomOutput && <BottomHint>{OutPutPortType.FailurePort}</BottomHint>}
    </>
  );
}
