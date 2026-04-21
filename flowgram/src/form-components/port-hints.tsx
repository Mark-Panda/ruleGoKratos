import styled from 'styled-components';

import { InputPortLabel, OutPutPortType } from '../nodes/constants';
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

/** 输入端：与出口 Success/Failure 区分色调 */
const InputHint = styled(Hint)`
  border-color: rgba(77, 83, 232, 0.35);
  background: rgba(77, 83, 232, 0.06);
  color: rgba(52, 56, 143, 0.92);
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

const LeftHint = styled(InputHint)`
  left: -8px;
  top: 50%;
  transform: translate(-100%, -50%);
`;

const TopHint = styled(InputHint)`
  left: 50%;
  top: -8px;
  transform: translate(-50%, -100%);
`;

export function PortHints() {
  const isSidebar = useIsSidebar();
  const { node } = useNodeRenderContext();
  if (isSidebar) return null;
  const ports = node.getNodeRegistry<any>()?.meta?.defaultPorts || [];
  const hasRightOutput = ports.some((p: any) => p?.type === 'output' && p?.location === 'right');
  const hasBottomOutput = ports.some((p: any) => p?.type === 'output' && p?.location === 'bottom');
  const hasLeftInput = ports.some((p: any) => p?.type === 'input' && p?.location === 'left');
  const hasTopInput = ports.some((p: any) => p?.type === 'input' && p?.location === 'top');
  return (
    <>
      {hasLeftInput && <LeftHint>{InputPortLabel}</LeftHint>}
      {hasTopInput && <TopHint>{InputPortLabel}</TopHint>}
      {hasRightOutput && <RightHint>{OutPutPortType.SuccessPort}</RightHint>}
      {hasBottomOutput && <BottomHint>{OutPutPortType.FailurePort}</BottomHint>}
    </>
  );
}
