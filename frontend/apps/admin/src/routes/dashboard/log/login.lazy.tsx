import { createLazyFileRoute } from "@tanstack/react-router";
import LoginLogPage from "@/sections/log/login";

export const Route = createLazyFileRoute("/dashboard/log/login")({
  component: LoginLogPage,
});
