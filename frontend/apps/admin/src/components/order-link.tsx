import { Link } from "@tanstack/react-router";
import { Button } from "@workspace/ui/components/button";

interface OrderLinkProps {
  orderId?: string | number;
  className?: string;
}

export function OrderLink({ orderId, className }: OrderLinkProps) {
  if (!orderId) return <span>--</span>;

  return (
    <Button asChild className={`p-0 ${className || ""}`} variant="link">
      <Link search={{ search: orderId }} to="/dashboard/order">
        {orderId}
      </Link>
    </Button>
  );
}
