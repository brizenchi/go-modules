type IconName = "account" | "billing" | "credits" | "notes" | "files" | "referrals" | "arrow" | "chevron" | "close" | "logout" | "menu" | "ai" | "overview" | "settings" | "integrations" | "audit" | "refresh";
const paths: Record<IconName, React.ReactNode> = {
  overview: <><rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" /></>,
  settings: <><path d="M4 7h16M4 17h16" /><circle cx="9" cy="7" r="3" fill="var(--surface)" /><circle cx="15" cy="17" r="3" fill="var(--surface)" /></>,
  integrations: <><path d="m9 15-2 2a4 4 0 0 1-5.7-5.7l4-4A4 4 0 0 1 11 7M15 9l2-2a4 4 0 0 1 5.7 5.7l-4 4A4 4 0 0 1 13 17M8 16l8-8" /></>,
  audit: <><path d="M5 3h14v18H5zM9 7h6M9 11h6m-6 5 2 2 4-4" /></>,
  refresh: <><path d="M20 11a8 8 0 1 0-2 6M20 4v7h-7" /></>,
  ai: <><path d="m12 3 2.5 6.5L21 12l-6.5 2.5L12 21l-2.5-6.5L3 12l6.5-2.5z" /></>,
  menu: <path d="M4 6h16M4 12h16M4 18h16" />,
  account: <><circle cx="12" cy="8" r="3.5" /><path d="M5 20c0-4 2.5-6 7-6s7 2 7 6" /></>,
  billing: <><rect x="3" y="5" width="18" height="14" rx="3" /><path d="M3 10h18M7 15h4" /></>,
  credits: <><ellipse cx="12" cy="6" rx="8" ry="3" /><path d="M4 6v6c0 4 16 4 16 0V6M4 12v6c0 4 16 4 16 0v-6" /></>,
  notes: <><path d="M5 3h10l4 4v14H5zM14 3v5h5M8 12h8M8 16h6" /></>,
  files: <><rect x="3" y="4" width="18" height="16" rx="3" /><circle cx="8" cy="9" r="1.5" /><path d="m4 18 5-5 4 4 3-3 5 5" /></>,
  referrals: <><circle cx="8" cy="8" r="3" /><circle cx="17" cy="7" r="2.5" /><path d="M3 19c0-3.5 2-5.3 5-5.3s5 1.8 5 5.3M15 14c3-.5 5 1.3 5.5 4" /></>,
  arrow: <path d="M5 12h14m-5-5 5 5-5 5" />,
  chevron: <path d="m7 10 5 5 5-5" />,
  close: <path d="m6 6 12 12M18 6 6 18" />,
  logout: <><path d="M9 5H5v14h4M10 12h10m-4-4 4 4-4 4" /></>
};
export function InterfaceIcon({ name, className }: { name: IconName; className?: string }) {
  return <svg className={className} viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>;
}
