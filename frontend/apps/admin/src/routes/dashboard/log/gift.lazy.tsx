import { createLazyFileRoute } from "@tanstack/react-router";
import GiftLogPage from "@/sections/log/gift";

export const Route = createLazyFileRoute("/dashboard/log/gift")({
  component: GiftLogPage,
});
