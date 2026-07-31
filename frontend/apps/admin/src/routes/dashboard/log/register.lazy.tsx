import { createLazyFileRoute } from "@tanstack/react-router";
import RegisterLogPage from "@/sections/log/register";

export const Route = createLazyFileRoute("/dashboard/log/register")({
  component: RegisterLogPage,
});
