"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { getMembership, listPurchases, type MembershipView, type PurchaseItem } from "@/lib/api";
import { formatBillingAmount } from "@/lib/billing-money";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { settleResource, type ResourceState, idleResource } from "@/lib/request-state";
import { ResourceFailure } from "@/components/resource-feedback";
import { Panel } from "@/components/ui";

export function BillingHistory({ token }: { token: string }) {
  const { t, locale } = useI18n();
  const [page, setPage] = useState(1);
  const [refresh, setRefresh] = useState(0);
  const [state, setState] = useState<ResourceState<{ items: PurchaseItem[]; total: number }>>(idleResource());
  const [membership, setMembership] = useState<ResourceState<MembershipView>>(idleResource());

  useEffect(() => {
    let active = true;
    void settleResource(listPurchases(token, page), "Purchase history").then(value => {
      if (active) setState(value);
    });
    void settleResource(getMembership(token), "Membership").then(value => {
      if (active) setMembership(value);
    });
    return () => { active = false; };
  }, [token, page, refresh]);

  function paymentStatus(value: string) {
    switch (value) {
      case "paid": return t({ en: "Paid", zh: "已付款" });
      case "failed": return t({ en: "Failed", zh: "付款失败" });
      case "unpaid": return t({ en: "Unpaid", zh: "待付款" });
      case "no_payment_required": return t({ en: "No payment required", zh: "无需付款" });
      default: return t({ en: "Pending", zh: "待确认" });
    }
  }

  return (
    <Panel className="span-12" title={t({ en: "Purchases and access", zh: "购买记录与会员权益" })}>
      <div className="button-row">
        <button className="button" onClick={() => setRefresh(value => value + 1)}>
          {t({ en: "Refresh", zh: "刷新" })}
        </button>
        <Link className="button" href="/credits">{t({ en: "Credits balance and history", zh: "积分余额与明细" })}</Link>
      </div>
      {membership.data && (
        <p>
          {t({ en: "Effective membership", zh: "实际生效会员" })}: {membership.data.plan} · {membership.data.active
            ? t({ en: "Active", zh: "有效" })
            : t({ en: "Inactive", zh: "未生效或已到期" })}
          {membership.data.valid_until ? ` · ${formatDate(membership.data.valid_until, locale)}` : ""}
        </p>
      )}
      {membership.status === "error" && <ResourceFailure failure={membership.failure} />}
      {state.status === "error" && <ResourceFailure failure={state.failure} />}
      <table className="table">
        <thead>
          <tr>
            <th>{t({ en: "Purchase", zh: "订单" })}</th>
            <th>{t({ en: "Amount", zh: "金额" })}</th>
            <th>{t({ en: "Payment / processing", zh: "付款 / 处理状态" })}</th>
            <th>{t({ en: "Time", zh: "时间" })}</th>
          </tr>
        </thead>
        <tbody>
          {(state.data?.items || []).map(item => (
            <tr key={item.id}>
              <td>
                {item.product_type === "credits" ? t({ en: "Credits", zh: "积分包" })
                  : item.product_type === "lifetime" ? t({ en: "Lifetime", zh: "终身会员" })
                  : item.product_type === "subscription" ? t({ en: "Subscription", zh: "订阅" }) : t({ en: "Payment", zh: "付款" })}
                <br /><small>{item.id}</small>
              </td>
              <td>{formatBillingAmount(item.amount, item.currency, locale)}</td>
              <td>{paymentStatus(item.status)} / {item.processed
                ? t({ en: "Processed", zh: "已处理" })
                : t({ en: "Processing", zh: "处理中" })}</td>
              <td>{formatDate(item.created_at, locale)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {state.data?.total === 0 && <p>{t({ en: "No purchases yet.", zh: "暂无购买记录。" })}</p>}
      <div className="button-row">
        <button className="button" disabled={page === 1} onClick={() => setPage(value => value - 1)}>
          {t({ en: "Previous", zh: "上一页" })}
        </button>
        <span>{page}</span>
        <button className="button" disabled={!state.data || page * 10 >= state.data.total} onClick={() => setPage(value => value + 1)}>
          {t({ en: "Next", zh: "下一页" })}
        </button>
      </div>
    </Panel>
  );
}
