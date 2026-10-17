# Pricing Transitions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Disable the current tier in pricing, label other recurring tiers Change plan, prorate upgrades, defer downgrades, and allow full-price Lifetime purchases during a subscription.

**Architecture:** Keep recurring changes on the existing preview/confirmation endpoints. Lifetime remains a separate one-time Checkout. A verified paid Lifetime activation retires the prior recurring subscription without refunds or proration before committing the permanent entitlement; failed Checkout does not touch the subscription. Reuse existing webhook retry and reservation protections.

**Tech Stack:** Next.js, TypeScript, Go, Stripe SDK, node:test and Go fake-provider tests.

---

1. Add failing frontend tests for card labels, current-tier selection, and Lifetime Checkout alongside Pro.
2. Add failing backend tests for Lifetime eligibility, safe replacement of settled recurring Checkout reservations, paid activation cleanup, retries and Stripe cancellation parameters.
3. Implement frontend labels/eligibility, additive Lifetime retirement capability, and standard module wiring; preserve all ownership, unpaid-checkout and duplicate-lifetime protection.
4. Verify existing recurring mode tests for immediate prorated upgrades and period-end downgrades. Run required Go formatting, vet/race/purity checks and isolated frontend verify.
5. Inspect the pricing/confirmation UI using an in-memory API only. Leave code local; no deployment, real payment, migration or production write.
