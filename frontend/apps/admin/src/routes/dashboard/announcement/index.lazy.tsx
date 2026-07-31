import { createLazyFileRoute } from "@tanstack/react-router";
import Announcement from "@/sections/announcement";

export const Route = createLazyFileRoute("/dashboard/announcement/")({
  component: Announcement,
});
