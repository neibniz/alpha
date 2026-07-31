import { createLazyFileRoute } from "@tanstack/react-router";
import User from "@/sections/user";

export const Route = createLazyFileRoute("/dashboard/user/")({
  component: User,
});
