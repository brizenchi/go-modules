import assert from "node:assert/strict";
import { test } from "node:test";

type Listener = () => void;

class MemoryStorage {
  private readonly data = new Map<string, string>();

  get length() {
    return this.data.size;
  }

  clear() {
    this.data.clear();
  }

  getItem(key: string) {
    return this.data.has(key) ? this.data.get(key)! : null;
  }

  key(index: number) {
    return Array.from(this.data.keys())[index] ?? null;
  }

  removeItem(key: string) {
    this.data.delete(key);
  }

  setItem(key: string, value: string) {
    this.data.set(key, value);
  }
}

function installWindow() {
  const listeners = new Map<string, Set<Listener>>();
  const windowMock = {
    localStorage: new MemoryStorage(),
    addEventListener(name: string, listener: Listener) {
      if (!listeners.has(name)) {
        listeners.set(name, new Set());
      }
      listeners.get(name)!.add(listener);
    },
    removeEventListener(name: string, listener: Listener) {
      listeners.get(name)?.delete(listener);
    },
    dispatchEvent(event: Event) {
      for (const listener of listeners.get(event.type) ?? []) {
        listener();
      }
      return true;
    }
  };

  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: windowMock
  });

  return windowMock;
}

function clearWindow() {
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: undefined
  });
}

function loadAuthModule() {
  const modPath = require.resolve("../lib/auth");
  delete require.cache[modPath];
  return require("../lib/auth") as typeof import("../lib/auth");
}

test("auth helpers are no-ops during SSR", () => {
  clearWindow();
  const auth = loadAuthModule();

  assert.equal(auth.readSession(), null);
  assert.equal(auth.readReferralCode(), "");
  assert.equal(auth.writeReferralCode(" ABC "), "ABC");
});

test("writeSession persists valid sessions and emits a session event", () => {
  const windowMock = installWindow();
  const auth = loadAuthModule();
  let emitted = 0;

  windowMock.addEventListener(auth.SESSION_EVENT, () => {
    emitted += 1;
  });

  const session = {
    token: "jwt-token",
    expires_at: new Date(Date.now() + 60_000).toISOString(),
    user: { id: "user_1", email: "user@example.com" }
  };

  auth.writeSession(session);

  assert.deepEqual(auth.readSession(), session);
  assert.equal(windowMock.localStorage.length, 1);
  assert.equal(emitted, 1);
});

test("readSession clears expired sessions", () => {
  const windowMock = installWindow();
  const auth = loadAuthModule();

  auth.writeSession({
    token: "expired",
    expires_at: new Date(Date.now() - 60_000).toISOString(),
    user: { id: "user_2", email: "expired@example.com" }
  });

  assert.equal(auth.readSession(), null);
  assert.equal(windowMock.localStorage.length, 0);
});

test("referral helpers trim, persist, clear, and emit", () => {
  const windowMock = installWindow();
  const auth = loadAuthModule();
  let emitted = 0;

  windowMock.addEventListener(auth.REFERRAL_EVENT, () => {
    emitted += 1;
  });

  assert.equal(auth.writeReferralCode(" REF123 "), "REF123");
  assert.equal(auth.readReferralCode(), "REF123");

  auth.clearReferralCode();

  assert.equal(auth.readReferralCode(), "");
  assert.equal(windowMock.localStorage.length, 0);
  assert.equal(emitted, 2);
});
