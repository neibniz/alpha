import { createLazyFileRoute } from "@tanstack/react-router";
import XrayTemplates from "@/sections/xray-templates";

export const Route = createLazyFileRoute("/dashboard/xray-templates")({
  component: XrayTemplates,
});
