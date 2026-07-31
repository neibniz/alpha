import { createLazyFileRoute } from "@tanstack/react-router";
import AuthControl from "@/sections/auth-control";

export const Route = createLazyFileRoute("/dashboard/auth-control/")({
  component: AuthControl,
});
