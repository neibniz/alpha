import { createLazyFileRoute } from "@tanstack/react-router";
import Membership from "@/sections/membership";

export const Route = createLazyFileRoute("/dashboard/membership/")({
  component: Membership,
});
