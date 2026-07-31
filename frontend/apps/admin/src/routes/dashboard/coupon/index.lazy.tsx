import { createLazyFileRoute } from "@tanstack/react-router";
import Coupon from "@/sections/coupon";

export const Route = createLazyFileRoute("/dashboard/coupon/")({
  component: Coupon,
});
