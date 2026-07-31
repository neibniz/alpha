import { createLazyFileRoute } from "@tanstack/react-router";
import EmailLogPage from "@/sections/log/email";

export const Route = createLazyFileRoute("/dashboard/log/email")({
  component: EmailLogPage,
});
