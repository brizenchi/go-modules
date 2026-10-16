import assert from "node:assert/strict";
import { test } from "node:test";
import { logoFileError, logoImageURL, maxLogoBytes, safeLogoURL } from "../lib/site-branding";
import { apiUrl } from "../lib/env";

test("site logo URLs support hosted assets and only the explicit public upload route", () => {
  for (const url of ["https://cdn.example.test/brand.svg", "/brand/logo.png"]) assert.equal(safeLogoURL(url), url);
  const uploaded = "/api/v1/site/logo/00000000-0000-4000-8000-000000000001";
  assert.equal(logoImageURL(uploaded), apiUrl("/site/logo/00000000-0000-4000-8000-000000000001"));
  for (const url of ["javascript:alert(1)", "data:image/svg+xml,x", "http://example.test/logo", "https://user:pass@example.test/logo", "//example.test/logo", "/\\example.test/logo", "/api/v1/uploads/images/private", "/api/v1/site/logo/not-a-uuid", "https://example.test/\nlogo", "x".repeat(1025), undefined]) assert.equal(safeLogoURL(url), "");
});

test("logo uploads validate raster formats and the existing storage size limit", () => {
  for (const type of ["image/png", "image/jpeg", "image/webp", "image/gif"]) assert.equal(logoFileError({ type, size: maxLogoBytes }), null);
  assert.equal(logoFileError({ type: "image/svg+xml", size: 50 }), "type");
  assert.equal(logoFileError({ type: "image/png", size: 0 }), "size");
  assert.equal(logoFileError({ type: "image/png", size: maxLogoBytes + 1 }), "size");
});
