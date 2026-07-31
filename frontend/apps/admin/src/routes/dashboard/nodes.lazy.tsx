import { createLazyFileRoute } from "@tanstack/react-router";
import Nodes from "@/sections/nodes";

export const Route = createLazyFileRoute("/dashboard/nodes")({
  component: Nodes,
});
