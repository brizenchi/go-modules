import { CustomerArea } from "@/components/customer-area";

export default function CustomerLayout({ children }: { children: React.ReactNode }) {
  return <CustomerArea>{children}</CustomerArea>;
}
