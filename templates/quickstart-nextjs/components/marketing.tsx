import Link from "next/link";

type FeatureCardProps = {
  label: string;
  title: string;
  description: string;
};

export function FeatureCard({ label, title, description }: FeatureCardProps) {
  return (
    <article className="feature-card">
      <span className="panel-kicker">{label}</span>
      <h3>{title}</h3>
      <p>{description}</p>
    </article>
  );
}

export function MetricCard({
  label,
  value,
  detail
}: {
  label: string;
  value: string;
  detail: string;
}) {
  return (
    <div className="metric-card">
      <span className="panel-kicker">{label}</span>
      <strong>{value}</strong>
      <p>{detail}</p>
    </div>
  );
}

export function DocArticle({
  id,
  title,
  description,
  children
}: {
  id: string;
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="doc-article" id={id}>
      <div className="doc-article-head">
        <h2>{title}</h2>
        {description ? <p>{description}</p> : null}
      </div>
      <div className="doc-article-body">{children}</div>
    </section>
  );
}

export function PricingCard({
  tier,
  price,
  subtitle,
  features,
  href,
  cta,
  featured = false
}: {
  tier: string;
  price: string;
  subtitle: string;
  features: string[];
  href: string;
  cta: string;
  featured?: boolean;
}) {
  return (
    <article className={`pricing-card${featured ? " featured" : ""}`}>
      <div className="pricing-head">
        <span className="panel-kicker">{tier}</span>
        <strong>{price}</strong>
        <p>{subtitle}</p>
      </div>

      <div className="pricing-features">
        {features.map((feature) => (
          <div className="pricing-feature" key={feature}>
            <span className="pricing-dot" />
            <span>{feature}</span>
          </div>
        ))}
      </div>

      <Link className={`button${featured ? " primary" : ""} wide`} href={href}>
        {cta}
      </Link>
    </article>
  );
}
