import assert from "node:assert/strict";
import { test } from "node:test";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "../lib/request-generation";

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

test("a slower same-session response cannot overwrite a newer generation", async () => {
  const generationRef = { current: 0 };
  const slow = deferred<string>();
  const fast = deferred<string>();
  const committed: string[] = [];

  async function run(request: Promise<string>) {
    const generation = beginRequestGeneration(generationRef);
    const value = await request;
    if (isCurrentRequestGeneration(generationRef, generation)) {
      committed.push(value);
    }
  }

  const slowRun = run(slow.promise);
  const fastRun = run(fast.promise);
  fast.resolve("new response");
  await fastRun;
  slow.resolve("stale response");
  await slowRun;

  assert.deepEqual(committed, ["new response"]);
});

test("invalidating a generation prevents an unmounted or signed-out request from committing", () => {
  const generationRef = { current: 0 };
  const generation = beginRequestGeneration(generationRef);
  invalidateRequestGeneration(generationRef);
  assert.equal(isCurrentRequestGeneration(generationRef, generation), false);
});

test("an older completion cannot clear the busy state of a newer request", async () => {
  const generationRef = { current: 0 };
  const slow = deferred<string>();
  const fast = deferred<string>();
  let busy = "";
  const committed: string[] = [];

  async function run(label: string, request: Promise<string>) {
    const generation = beginRequestGeneration(generationRef);
    busy = label;
    try {
      const value = await request;
      if (isCurrentRequestGeneration(generationRef, generation)) {
        committed.push(value);
      }
    } finally {
      if (isCurrentRequestGeneration(generationRef, generation)) {
        busy = "";
      }
    }
  }

  const slowRun = run("send-one", slow.promise);
  const fastRun = run("send-two", fast.promise);
  slow.resolve("stale code");
  await slowRun;

  assert.equal(busy, "send-two");
  assert.deepEqual(committed, []);

  fast.resolve("latest code");
  await fastRun;
  assert.equal(busy, "");
  assert.deepEqual(committed, ["latest code"]);
});

test("a request only commits while mounted and on its initial session", async () => {
  const generationRef = { current: 0 };
  let mounted = true;
  let currentSessionToken = "session-a";
  let writtenSession = "";

  async function verify(request: Promise<string>) {
    const generation = beginRequestGeneration(generationRef);
    const initialSessionToken = currentSessionToken;
    const nextSession = await request;
    if (
      mounted
      && isCurrentRequestGeneration(generationRef, generation)
      && currentSessionToken === initialSessionToken
    ) {
      writtenSession = nextSession;
    }
  }

  const changedSession = deferred<string>();
  const changedSessionRun = verify(changedSession.promise);
  currentSessionToken = "session-b";
  changedSession.resolve("stale verification session");
  await changedSessionRun;
  assert.equal(writtenSession, "");

  const unmounted = deferred<string>();
  const unmountedRun = verify(unmounted.promise);
  mounted = false;
  invalidateRequestGeneration(generationRef);
  unmounted.resolve("unmounted verification session");
  await unmountedRun;
  assert.equal(writtenSession, "");
});

test("a capabilities retry wins over a slower earlier failure", async () => {
  const generationRef = { current: 0 };
  let mounted = true;
  let state = "idle";
  const slowFailure = deferred<string>();
  const fastSuccess = deferred<string>();

  async function load(request: Promise<string>) {
    const generation = beginRequestGeneration(generationRef);
    state = "loading";
    const nextState = await request;
    if (mounted && isCurrentRequestGeneration(generationRef, generation)) {
      state = nextState;
    }
  }

  const initialLoad = load(slowFailure.promise);
  const retryLoad = load(fastSuccess.promise);
  fastSuccess.resolve("ready");
  await retryLoad;
  slowFailure.resolve("error");
  await initialLoad;
  assert.equal(state, "ready");

  const afterUnmount = deferred<string>();
  const afterUnmountLoad = load(afterUnmount.promise);
  mounted = false;
  invalidateRequestGeneration(generationRef);
  afterUnmount.resolve("error-after-unmount");
  await afterUnmountLoad;
  assert.equal(state, "loading");
});
