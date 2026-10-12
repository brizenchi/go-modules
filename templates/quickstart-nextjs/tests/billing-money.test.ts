import test from "node:test";
import assert from "node:assert/strict";
import { formatBillingAmount } from "../lib/billing-money";

test("billing amounts preserve zero and use currency precision",()=>{
 assert.equal(formatBillingAmount(0,"usd"),"$0.00");
 assert.equal(formatBillingAmount(1299,"usd"),"$12.99");
 assert.equal(formatBillingAmount(1299,"jpy"),"¥1,299");
 for (const currency of ["ISK", "UGX"]) {
  assert.equal(formatBillingAmount(500,currency),new Intl.NumberFormat("en-US",{style:"currency",currency}).format(5));
 }
 assert.equal(formatBillingAmount(null,"usd"),"—");
 assert.equal(formatBillingAmount(1,"invalid"),"—");
});
