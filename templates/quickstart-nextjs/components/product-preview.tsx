const navigation = [
  { label: "Overview", active: true },
  { label: "Customers" },
  { label: "Billing" },
  { label: "Referrals" }
];

const events = [
  { initials: "AL", name: "Avery Lin", detail: "Upgraded to Pro", time: "2m" },
  { initials: "KM", name: "Kai Morgan", detail: "Joined via referral", time: "18m" },
  { initials: "RS", name: "Riley Shah", detail: "Purchased 2,000 credits", time: "1h" }
];

export function ProductPreview() {
  return (
    <section className="product-stage" aria-label="SaaS dashboard preview">
      <div className="product-window">
        <div className="window-bar">
          <div className="window-dots" aria-hidden="true">
            <span />
            <span />
            <span />
          </div>
          <span className="window-address">app.yourproduct.com/overview</span>
          <span className="window-secure">Live</span>
        </div>

        <div className="preview-app">
          <aside className="preview-sidebar" aria-label="Preview navigation">
            <div className="preview-workspace">
              <span className="workspace-mark">A</span>
              <span>
                <strong>Acme Labs</strong>
                <small>Production</small>
              </span>
            </div>

            <nav className="preview-nav">
              {navigation.map((item, index) => (
                <span className={item.active ? "active" : ""} key={item.label}>
                  <i aria-hidden="true">0{index + 1}</i>
                  {item.label}
                </span>
              ))}
            </nav>

            <div className="preview-sidebar-footer">
              <span className="preview-avatar">BC</span>
              <span>
                <strong>Builder</strong>
                <small>Workspace admin</small>
              </span>
            </div>
          </aside>

          <div className="preview-canvas">
            <header className="preview-header">
              <div>
                <span className="preview-overline">Workspace overview</span>
                <h2>Good morning, Builder.</h2>
              </div>
              <span className="preview-action">New checkout <span aria-hidden="true">↗</span></span>
            </header>

            <div className="preview-metrics">
              <article>
                <span>Monthly revenue</span>
                <strong>$18,240</strong>
                <small className="positive">↗ 18.4%</small>
              </article>
              <article>
                <span>Active customers</span>
                <strong>1,429</strong>
                <small>Across 3 plans</small>
              </article>
              <article>
                <span>Referral activation</span>
                <strong>32.8%</strong>
                <small className="positive">↗ 4.2%</small>
              </article>
            </div>

            <div className="preview-grid">
              <article className="revenue-card">
                <div className="preview-card-head">
                  <div>
                    <span>Net revenue</span>
                    <strong>$42.8k</strong>
                  </div>
                  <span className="preview-period">Last 30 days</span>
                </div>
                <div className="chart-shell" aria-hidden="true">
                  <div className="chart-grid" />
                  <svg viewBox="0 0 620 180" preserveAspectRatio="none">
                    <path className="chart-fill" d="M0 158 C38 148 62 151 92 132 S148 146 184 114 S242 125 275 97 S338 105 370 72 S430 90 466 48 S534 66 620 20 L620 180 L0 180 Z" />
                    <path className="chart-line" d="M0 158 C38 148 62 151 92 132 S148 146 184 114 S242 125 275 97 S338 105 370 72 S430 90 466 48 S534 66 620 20" />
                  </svg>
                  <span className="chart-point" />
                </div>
                <div className="chart-axis" aria-hidden="true">
                  <span>Aug 06</span><span>Aug 13</span><span>Aug 20</span><span>Today</span>
                </div>
              </article>

              <article className="activity-card">
                <div className="preview-card-head">
                  <div>
                    <span>Live activity</span>
                    <strong>Customer events</strong>
                  </div>
                  <span className="live-dot">Live</span>
                </div>
                <div className="activity-list">
                  {events.map((event) => (
                    <div className="activity-row" key={event.name}>
                      <span className="event-avatar">{event.initials}</span>
                      <span>
                        <strong>{event.name}</strong>
                        <small>{event.detail}</small>
                      </span>
                      <time>{event.time}</time>
                    </div>
                  ))}
                </div>
              </article>
            </div>
          </div>
        </div>
      </div>

      <div className="preview-callout callout-auth">
        <span className="callout-icon">✓</span>
        <span><strong>Authentication</strong><small>Email &amp; OAuth ready</small></span>
      </div>
      <div className="preview-callout callout-webhook">
        <span className="callout-pulse" />
        <span><strong>Webhook verified</strong><small>Stripe event processed</small></span>
      </div>
    </section>
  );
}
