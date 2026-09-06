import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { test } from "node:test";

const panelSource = readFileSync(
  path.join(process.cwd(), "components", "sign-in-panel.tsx"),
  "utf8"
);

test("email verification is bound to the email returned by send-code", () => {
  assert.match(panelSource, /const verificationEmail = sendResult\.email;/);
  assert.match(
    panelSource,
    /await verifyCode\(verificationEmail, code, /
  );
});

test("verification guards session writes and invalidates requests on unmount", () => {
  const verifyRequest = panelSource.indexOf(
    "const session = await verifyCode(verificationEmail, code,"
  );
  const commitGuard = panelSource.indexOf(
    "if (!canCommitEmailAction(generation, initialSessionToken))",
    verifyRequest
  );
  const sessionWrite = panelSource.indexOf("writeSession(session);", verifyRequest);

  assert.notEqual(verifyRequest, -1);
  assert.ok(commitGuard > verifyRequest);
  assert.ok(sessionWrite > commitGuard);
  assert.match(panelSource, /mountedRef\.current = false;/);
  assert.match(
    panelSource,
    /invalidateRequestGeneration\(emailActionGenerationRef\)/
  );
});

test("the sign-in panel does not expose implementation notes to end users", () => {
  assert.doesNotMatch(
    panelSource,
    /OAuth sign-in uses the backend redirect flow/
  );
});

test("OAuth starts with a top-level backend redirect", () => {
	assert.match(panelSource, /await getOAuthRedirectURL\(provider, returnTo\)/);
	assert.doesNotMatch(panelSource, /getOAuthAuthorizeURL/);
});

test("sign-in actions wait for backend capabilities and capability failures are retryable", () => {
  assert.match(panelSource, /await settleResource\(getCapabilities\(\), "Sign-in methods"\)/);
  assert.match(panelSource, /oauthProviders\.map/);
  assert.match(panelSource, /emailEnabled && \(!codeIssued/);
  assert.match(panelSource, /<ResourceFailure failure=\{capabilitiesState\.failure\} onRetry=/);
});
