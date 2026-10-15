import { ApiError } from "./api";

export function accountLogoutFailure(error: unknown, locale: "en" | "zh"): string {
  const zh = locale === "zh";
  const message = error instanceof ApiError && error.status === 403
    ? (zh ? "暂时无法退出，请重试。" : "Unable to sign out. Please retry.")
    : (zh ? "退出失败，请检查网络后重试。" : "Sign-out failed. Check your connection and retry.");
  return error instanceof ApiError && error.status >= 500 && error.requestId
    ? `${message} ${zh ? "请求编号" : "Request ID"}: ${error.requestId}` : message;
}
