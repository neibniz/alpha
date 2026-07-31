import { createLazyFileRoute } from "@tanstack/react-router";
import Forwarders from "@/sections/forwarders";

export const Route = createLazyFileRoute("/dashboard/forwarders")({
  component: Forwarders,
});
