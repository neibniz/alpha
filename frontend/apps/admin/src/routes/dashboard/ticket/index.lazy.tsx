import { createLazyFileRoute } from "@tanstack/react-router";
import Ticket from "@/sections/ticket";

export const Route = createLazyFileRoute("/dashboard/ticket/")({
  component: Ticket,
});
