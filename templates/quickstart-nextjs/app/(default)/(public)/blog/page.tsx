import { BlogIndex } from "@/components/content/blog";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Product guides", "Practical guides to configure your SaaS, understand invitation rewards, and support your users.", "/blog");

export default function BlogPage() { return <BlogIndex />; }
