import assert from "node:assert/strict";
import test from "node:test";
import { analyticsMeasurementId, crispChatURL, analyticsPageURL, analyticsConsent } from "../lib/site-extensions";

test("extensions accept only known provider identifiers and fixed embed hosts", () => {
 assert.equal(analyticsMeasurementId("G-EXAMPLE123"),"G-EXAMPLE123");
 for(const id of ["G-<script>","https://evil.test","G-X\" onload=\"x"]) assert.equal(analyticsMeasurementId(id),"");
 assert.equal(crispChatURL("00000000-0000-0000-0000-000000000000"),"https://go.crisp.chat/chat/embed/?website_id=00000000-0000-0000-0000-000000000000");
 assert.equal(crispChatURL("https://evil.test"),"");
});
test("analytics exclude private destinations and remove every query and fragment", () => {
 assert.equal(analyticsPageURL("https://example.test","/zh/pricing?email=private@example.test#plans"),"https://example.test/zh/pricing");
 for(const path of ["/account","/billing?checkout=success","/admin","/oauth/callback?code=private","/login?code=private","/notes","https://evil.test/pricing"]) assert.equal(analyticsPageURL("https://example.test",path),null,path);
 assert.equal(analyticsConsent("allowed"),"allowed"); assert.equal(analyticsConsent("declined"),"declined"); assert.equal(analyticsConsent("true"),null);
});
