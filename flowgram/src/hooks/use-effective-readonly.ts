/**
 * 画布节点仅预览：在非侧栏渲染时等价只读；侧栏内仍可编辑（除非 playground 全局 readonly）。
 */
import { useNodeRenderContext } from './use-node-render-context';
import { useIsSidebar } from './use-is-sidebar';

export function useEffectiveReadonly(): boolean {
  const { readonly } = useNodeRenderContext();
  const isSidebar = useIsSidebar();
  return readonly || !isSidebar;
}
