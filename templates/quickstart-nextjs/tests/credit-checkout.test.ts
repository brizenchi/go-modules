import assert from "node:assert/strict";
import { test } from "node:test";
import { readCreditCheckoutSession, waitForCreditCheckout } from "../lib/credit-checkout";
import type { CreditCheckoutResult } from "../lib/api";

test("credit return accepts a single successful checkout reference only", () => {
 assert.equal(readCreditCheckoutSession("?checkout=success&session_id=cs_test_example"), "cs_test_example");
 for (const search of ["", "?checkout=cancelled&session_id=cs_test_example", "?checkout=success", "?checkout=success&session_id=evil", "?checkout=success&session_id=cs_one&session_id=cs_two", "?checkout=success&checkout=cancelled&session_id=cs_one"]) assert.equal(readCreditCheckoutSession(search), null);
});

test("payment processing and an unchanged balance cannot produce credit purchase success", async () => {
 const calls:number[]=[];
 const result=await waitForCreditCheckout({load:async()=>({state:"processing",credits_granted:0}),current:()=>true,wait:async delay=>{calls.push(delay);}});
 assert.equal(result?.state,"processing");
 assert.ok(calls.length>1 && calls.length<=10,"confirmation must stop after bounded retries");
});

test("return confirmation can fulfill immediately before any webhook is delivered", async () => {
 let reads=0, confirms=0;
 const result=await waitForCreditCheckout({confirm:async()=>{confirms++;return {state:"fulfilled",credits_granted:100};},load:async()=>{reads++;return {state:"pending",credits_granted:0};},current:()=>true,wait:async()=>{}});
 assert.equal(result?.credits_granted,100);assert.equal(confirms,1);assert.equal(reads,0);
});

test("a lost confirmation response is retried safely and does not stop the purchase check", async () => {
 let confirms=0;
 const result=await waitForCreditCheckout({confirm:async()=>{if(++confirms===1)throw new Error("connection lost");return {state:"fulfilled",credits_granted:100};},load:async()=>({state:"pending",credits_granted:0}),current:()=>true,wait:async()=>{}});
 assert.equal(result?.state,"fulfilled");assert.equal(confirms,2);
});

test("a delayed ledger grant stops confirmation as soon as that purchase is fulfilled", async () => {
 let calls=0;
 const result=await waitForCreditCheckout({load:async():Promise<CreditCheckoutResult>=>++calls<3?{state:"processing",credits_granted:0}:{state:"fulfilled",credits_granted:500},current:()=>true,wait:async()=>{}});
 assert.equal(result?.credits_granted,500);assert.equal(calls,3);
});

test("failed, expired and stale sessions stop without claiming fulfillment", async () => {
 for (const state of ["failed","expired"] as const) {
  let calls=0;
  assert.equal((await waitForCreditCheckout({load:async()=>{calls++;return {state,credits_granted:0};},current:()=>true,wait:async()=>{}}))?.state,state);
  assert.equal(calls,1);
 }
 let current=true;
 assert.equal(await waitForCreditCheckout({load:async()=>{current=false;return {state:"fulfilled",credits_granted:500};},current:()=>current,wait:async()=>{}}),null);
 await assert.rejects(waitForCreditCheckout({load:async()=>({state:"fulfilled",credits_granted:0}),current:()=>true,wait:async()=>{}}));
});
