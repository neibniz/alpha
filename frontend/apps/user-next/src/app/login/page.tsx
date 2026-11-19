import type { Metadata } from "next";
import { AuthScreen } from "@/components/auth/auth-screen";

export const metadata: Metadata = { title: "登录" };
export const dynamic = "force-dynamic";

export default function LoginPage() {
  return <AuthScreen />;
}
