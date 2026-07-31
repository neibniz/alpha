import { createLazyFileRoute } from "@tanstack/react-router";
import Document from "@/sections/document";

export const Route = createLazyFileRoute("/dashboard/document/")({
  component: Document,
});
