import { ApiError } from "./api";

export type RequestFailureKind =
  | "auth"
  | "disabled"
  | "configuration"
  | "network"
  | "unavailable"
  | "unknown";

export type RequestFailure = {
  kind: RequestFailureKind;
  title: string;
  message: string;
  retryable: boolean;
};

export type ResourceState<T> =
  | { status: "idle"; data: null; failure: null }
  | { status: "loading"; data: T | null; failure: null }
  | { status: "ready"; data: T; failure: null }
  | { status: "error"; data: T | null; failure: RequestFailure };

export function idleResource<T>(): ResourceState<T> {
  return { status: "idle", data: null, failure: null };
}

export function loadingResource<T>(data: T | null = null): ResourceState<T> {
  return { status: "loading", data, failure: null };
}

export function readyResource<T>(data: T): ResourceState<T> {
  return { status: "ready", data, failure: null };
}

export function failedResource<T>(failure: RequestFailure, data: T | null = null): ResourceState<T> {
  return { status: "error", data, failure };
}

function looksLikeConfigurationError(message: string): boolean {
  return /config|configured|configuration|credential|price|provider|secret|disabled|enable/i.test(message);
}

export function describeRequestFailure(error: unknown, capability: string): RequestFailure {
  if (error instanceof ApiError) {
    if (error.status === 401 || error.code === 401) {
      return {
        kind: "auth",
        title: "Sign in again",
        message: "Your saved session is no longer accepted by the API. Sign in again to continue.",
        retryable: false
      };
    }

    if (error.status === 404 || error.code === 404) {
      return {
        kind: "disabled",
        title: `${capability} is not enabled`,
        message: "This backend does not expose the required endpoint. Enable the matching module and redeploy the API.",
        retryable: false
      };
    }

    if ((error.status === 503 || error.code === 503) && looksLikeConfigurationError(error.message)) {
      return {
        kind: "configuration",
        title: `${capability} needs configuration`,
        message: error.message,
        retryable: false
      };
    }

    if (error.status === 503 || error.code === 503) {
      return {
        kind: "unavailable",
        title: `${capability} is temporarily unavailable`,
        message: error.message || "The API reported that this capability is unavailable.",
        retryable: true
      };
    }

    if ((error.status === 400 || error.status === 422) && looksLikeConfigurationError(error.message)) {
      return {
        kind: "configuration",
        title: `${capability} needs configuration`,
        message: error.message,
        retryable: false
      };
    }

    return {
      kind: "unknown",
      title: `Could not load ${capability.toLowerCase()}`,
      message: error.message || `The API returned HTTP ${error.status}.`,
      retryable: error.status >= 500
    };
  }

  if (error instanceof TypeError) {
    return {
      kind: "network",
      title: "Cannot reach the API",
      message: "Check the API URL, CORS configuration, and network connection, then try again.",
      retryable: true
    };
  }

  return {
    kind: "unknown",
    title: `Could not load ${capability.toLowerCase()}`,
    message: error instanceof Error ? error.message : "An unexpected error occurred.",
    retryable: true
  };
}

export async function settleResource<T>(
  request: Promise<T>,
  capability: string
): Promise<ResourceState<T>> {
  try {
    return readyResource(await request);
  } catch (error) {
    return failedResource(describeRequestFailure(error, capability));
  }
}
