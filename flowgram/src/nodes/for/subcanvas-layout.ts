/**
 * SubCanvasRender 高度 = 容器 bounds.height + offsetY（见 @flowgram.ai/free-container-plugin）。
 * For 画布上 SubCanvas 之上还有「处理节点ID / 迭代表达式 / 执行模式」等表单项，若不设负 offsetY，
 * 子画布会占满整高，表单体与 transform 底边不齐，底部 Failure 等端口会显示错位。
 * 与 loop/form-meta 中 formHeight=115 同理；数值与 for/index.ts 中容器 padding.top 保持一致便于维护。
 */
export const FOR_SUBCANVAS_TOP_FORM_RESERVE_PX = 248;
