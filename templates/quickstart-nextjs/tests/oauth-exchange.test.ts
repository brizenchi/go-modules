import assert from "node:assert/strict";
import { test } from "node:test";
import { createOAuthExchangeCoordinator } from "../lib/oauth-exchange";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((next, fail) => {
    resolve = next;
    reject = fail;
  });
  return { promise, resolve, reject };
}

test("OAuth exchange is single-flight per code and only the latest code may commit", async () => {
  const requests = new Map<string, ReturnType<typeof deferred<string>>>();
  const committed: string[] = [];
  let exchangeCalls = 0;
  let sessionToken = "";
  const coordinator = createOAuthExchangeCoordinator<string>({
    exchange: (code) => {
      exchangeCalls += 1;
      const request = deferred<string>();
      requests.set(code, request);
      return request.promise;
    },
    readSessionToken: () => sessionToken,
    commit: (session) => {
      committed.push(session);
      sessionToken = session;
    }
  });

  const first = coordinator.exchange("code-a");
  const strictModeReplay = coordinator.exchange("code-a");
  assert.equal(first, strictModeReplay);
  assert.equal(exchangeCalls, 1);

  const latest = coordinator.exchange("code-b");
  assert.equal(exchangeCalls, 2);
  requests.get("code-b")!.resolve("token-b");
  assert.deepEqual(await latest, { session: "token-b", committed: true });
  requests.get("code-a")!.resolve("token-a");
  assert.deepEqual(await first, { session: "token-a", committed: false });
  assert.deepEqual(committed, ["token-b"]);
});

test("OAuth exchange does not overwrite a session created while it was pending", async () => {
  const request = deferred<string>();
  const committed: string[] = [];
  let sessionToken = "old-token";
  const coordinator = createOAuthExchangeCoordinator<string>({
    exchange: () => request.promise,
    readSessionToken: () => sessionToken,
    commit: (session) => committed.push(session)
  });

  const exchange = coordinator.exchange("code-a");
  sessionToken = "newer-token";
  request.resolve("oauth-token");

  assert.deepEqual(await exchange, { session: "oauth-token", committed: false });
  assert.deepEqual(committed, []);
});

test("a failed current exchange is evicted so the same code can retry", async () => {
  const requests: Array<ReturnType<typeof deferred<string>>> = [];
  let exchangeCalls = 0;
  const coordinator = createOAuthExchangeCoordinator<string>({
    exchange: () => {
      exchangeCalls += 1;
      const request = deferred<string>();
      requests.push(request);
      return request.promise;
    },
    readSessionToken: () => "",
    commit: () => undefined
  });

  const failed = coordinator.exchange("retryable-code");
  requests[0].reject(new Error("temporary network failure"));
  await assert.rejects(failed, /temporary network failure/);

  const retry = coordinator.exchange("retryable-code");
  assert.equal(exchangeCalls, 2);
  requests[1].resolve("token-after-retry");
  assert.deepEqual(await retry, { session: "token-after-retry", committed: true });
});
