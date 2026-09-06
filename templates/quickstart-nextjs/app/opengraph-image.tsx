import { ImageResponse } from "next/og";
import { appEnv } from "@/lib/env";

export const alt = "A free SaaS starter with authentication, billing and referrals";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpenGraphImage() {
  return new ImageResponse(
    <div style={{ width: "100%", height: "100%", display: "flex", flexDirection: "column", justifyContent: "space-between", background: "#f5f6f2", color: "#151815", padding: "64px 72px" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 18, fontSize: 26 }}>
        <div style={{ width: 24, height: 24, borderRadius: 6, background: "#38d477" }} />
        <span>{appEnv.appName}</span>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 26 }}>
        <div style={{ display: "flex", fontSize: 76, fontWeight: 700, lineHeight: 1.06, letterSpacing: "-4px", maxWidth: 900 }}>Get your SaaS to launch, faster.</div>
        <div style={{ display: "flex", fontSize: 26, color: "#4d554e" }}>Authentication · Subscriptions · Credits · Referrals</div>
      </div>
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 20, borderTop: "1px solid #c8cec5", paddingTop: 22 }}>
        <span>Free SaaS starter</span><span>Next.js + Go</span>
      </div>
    </div>, size
  );
}
