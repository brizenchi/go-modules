import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { ResourceFailure, SignInRequired } from "../components/resource-feedback";

test("ResourceFailure renders an explicit module-disabled state without a retry action", () => {
  const markup = renderToStaticMarkup(
    <ResourceFailure failure={{
      kind: "disabled",
      title: "Referral center is not enabled",
      message: "Enable the referral module.",
      retryable: false
    }} />
  );

  assert.match(markup, /Referral center is not enabled/);
  assert.match(markup, /Enable the referral module/);
  assert.doesNotMatch(markup, /Try again/);
});

test("SignInRequired includes a usable login destination", () => {
  const markup = renderToStaticMarkup(<SignInRequired message="A valid session is required." />);
  assert.match(markup, /href="\/login"/);
  assert.match(markup, /Sign in to continue/);
});

test("ResourceFailure exposes a retry action for a retryable request", () => {
  const markup = renderToStaticMarkup(
    <ResourceFailure
      failure={{
        kind: "network",
        title: "Subscription change preview unavailable",
        message: "Try the request again.",
        retryable: true
      }}
      onRetry={() => {}}
    />
  );

  assert.match(markup, /Try again/);
});
