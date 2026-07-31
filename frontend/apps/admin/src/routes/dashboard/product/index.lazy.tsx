import { createLazyFileRoute } from "@tanstack/react-router";
import Product from "@/sections/product";

export const Route = createLazyFileRoute("/dashboard/product/")({
  component: Product,
});
