import assert from "node:assert/strict";
import { test } from "node:test";
import { ApiError, logout } from "../lib/api";
import { readSession, writeSession } from "../lib/auth";

test("logout removes only the confirmed session and preserves retryable failures", async (suite) => {
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const originalFetch = globalThis.fetch;
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      localStorage: {
        getItem: (key: string) => storage.get(key) ?? null,
        setItem: (key: string, value: string) => storage.set(key, value),
        removeItem: (key: string) => storage.delete(key)
      },
      dispatchEvent: () => true
    }
  });
  suite.after(() => {
    globalThis.fetch = originalFetch;
    if (originalWindow) Object.defineProperty(globalThis, "window", originalWindow);
    else Reflect.deleteProperty(globalThis, "window");
  });
  const session = {
    token: "logout-test-token",
    expires_at: new Date(Date.now() + 3600000).toISOString(),
    user: { id: "user", email: "user@example.test", role: "user" }
  };
  for (const status of [200, 401, 503]) {
    await suite.test(`HTTP ${status}`, async () => {
      writeSession(session);
      globalThis.fetch = async (_input, options) => {
        assert.equal(readSession()?.token, session.token);
        assert.equal(options?.method, "POST");
        assert.equal(new Headers(options?.headers).get("Authorization"), `Bearer ${session.token}`);
        return Response.json({ code: status, data: { ok: status === 200 }, msg: "test" }, { status });
      };
      if (status === 503) {
        await assert.rejects(logout(session.token), ApiError);
        assert.equal(readSession()?.token, session.token);
      } else {
        await logout(session.token);
        assert.equal(readSession(), null);
      }
    });
  }
  await suite.test("network error keeps credentials for retry", async () => {
    writeSession(session);
    globalThis.fetch = async () => { throw new Error("offline"); };
    await assert.rejects(logout(session.token), /offline/);
    assert.equal(readSession()?.token, session.token);
  });
  await suite.test("unconfirmed response keeps credentials", async () => {
    writeSession(session);
    globalThis.fetch = async () => Response.json({ code: 200, data: { ok: false } });
    await assert.rejects(logout(session.token), /not confirmed/);
    assert.equal(readSession()?.token, session.token);
  });
  for (const status of [200, 401]) {
    await suite.test(`late HTTP ${status} does not clear a newer login`, async () => {
      writeSession(session);
      globalThis.fetch = async () => {
        writeSession({ ...session, token: "newer-session" });
        return Response.json({ code: status, data: { ok: true } }, { status });
      };
      await logout(session.token);
      assert.equal(readSession()?.token, "newer-session");
    });
  }
});
