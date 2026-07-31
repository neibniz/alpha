import { createLazyFileRoute } from "@tanstack/react-router";
import System from "@/sections/system";

export const Route = createLazyFileRoute("/dashboard/system/")({
  component: System,
});
