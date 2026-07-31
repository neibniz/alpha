"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  DialogDescription,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import {
  Check,
  CircleAlert,
  Clock3,
  ExternalLink,
  QrCode,
  ReceiptText,
} from "lucide-react";
import { Spinner } from "@workspace/ui/components/spinner";
import dynamic from "next/dynamic";
import { QRCodeCanvas } from "qrcode.react";
import { useEffect, useRef } from "react";
import { toast } from "sonner";
import { getOrderDetail } from "@/lib/client-api";
import type { CheckoutResult } from "@/lib/models";

const StripeCheckoutPanel = dynamic(
  () => import("@/components/dashboard/stripe-checkout"),
  {
    loading: () => (
      <div className="flex min-h-48 items-center justify-center gap-2 text-muted-foreground text-sm">
        <Spinner />
        正在加载安全支付组件
      </div>
    ),
    ssr: false,
  }
);

export type OrderCheckoutKind = "purchase" | "recharge" | "renewal" | "reset";

const completedTitles: Record<OrderCheckoutKind, string> = {
  purchase: "订阅已开通",
  recharge: "充值已到账",
  renewal: "续费已完成",
  reset: "流量已重置",
};

export function OrderCheckoutStatus({
  checkout,
  kind,
  onClose,
  onComplete,
  orderNo,
}: Readonly<{
  checkout: CheckoutResult;
  kind: OrderCheckoutKind;
  onClose: () => void;
  onComplete?: () => Promise<void> | void;
  orderNo: string;
}>) {
  const completedOrder = useRef("");
  const order = useQuery({
    enabled: Boolean(orderNo),
    queryFn: () => getOrderDetail(orderNo),
    queryKey: ["alpha-order", orderNo],
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status === undefined || status === 1 || status === 2
        ? 1200
        : false;
    },
    refetchIntervalInBackground: true,
    retry: 3,
  });
  const status = order.data?.status;

  useEffect(() => {
    if (status !== 5 || completedOrder.current === orderNo) return;
    completedOrder.current = orderNo;
    Promise.resolve(onComplete?.()).catch(() => {
      toast.error("订单已完成，但页面数据刷新失败");
    });
  }, [onComplete, orderNo, status]);

  if (status === 5) {
    return (
      <div
        aria-live="polite"
        className="flex flex-col items-center gap-4 py-6 text-center"
      >
        <span className="flex size-12 items-center justify-center rounded-full bg-teal-50 text-teal-700">
          <Check aria-hidden="true" />
        </span>
        <DialogTitle>{completedTitles[kind]}</DialogTitle>
        <DialogDescription>
          订单 {orderNo} 已完成，账户数据已按服务端结果更新。
        </DialogDescription>
        <Button onClick={onClose}>完成</Button>
      </div>
    );
  }

  if (status === 3 || status === 4) {
    return (
      <div
        aria-live="assertive"
        className="flex flex-col items-center gap-4 py-6 text-center"
      >
        <span className="flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
          <CircleAlert aria-hidden="true" />
        </span>
        <DialogTitle>
          {status === 3 ? "订单已取消" : "订单处理失败"}
        </DialogTitle>
        <DialogDescription>
          订单 {orderNo} 未完成，请返回后重新发起或联系支持。
        </DialogDescription>
        <Button onClick={onClose} variant="outline">
          返回
        </Button>
      </div>
    );
  }

  if (
    status === 2 ||
    ["balance", "free", "processing"].includes(checkout.type)
  ) {
    return (
      <div
        aria-live="polite"
        className="flex flex-col items-center gap-4 py-6 text-center"
      >
        <span className="flex size-12 items-center justify-center rounded-full bg-muted">
          <Spinner aria-hidden="true" />
        </span>
        <DialogTitle>支付已确认，正在生效</DialogTitle>
        <DialogDescription>
          订单 {orderNo} 正在异步处理，本页面会自动更新最终结果。
        </DialogDescription>
        {order.isError ? (
          <Button onClick={() => order.refetch()} variant="outline">
            重新查询
          </Button>
        ) : (
          <Button onClick={onClose} variant="outline">
            后台处理
          </Button>
        )}
      </div>
    );
  }

  if (checkout.type === "qr" && checkout.checkoutUrl) {
    return (
      <div
        aria-live="polite"
        className="flex flex-col items-center gap-4 py-4 text-center"
      >
        <span className="flex size-12 items-center justify-center rounded-full bg-muted">
          <QrCode aria-hidden="true" />
        </span>
        <DialogTitle>扫码完成支付</DialogTitle>
        <DialogDescription>
          订单 {orderNo} · 支付结果将在此处自动确认
        </DialogDescription>
        <div className="rounded-lg border bg-white p-3 shadow-xs">
          <QRCodeCanvas size={196} value={checkout.checkoutUrl} />
        </div>
        <Button onClick={onClose} variant="outline">
          稍后在订单中查看
        </Button>
      </div>
    );
  }

  if (checkout.type === "url" && checkout.checkoutUrl) {
    return (
      <div
        aria-live="polite"
        className="flex flex-col items-center gap-4 py-6 text-center"
      >
        <span className="flex size-12 items-center justify-center rounded-full bg-muted">
          <ExternalLink aria-hidden="true" />
        </span>
        <DialogTitle>继续在线支付</DialogTitle>
        <DialogDescription>
          订单 {orderNo} 已创建，支付后本页面会自动确认结果。
        </DialogDescription>
        <Button asChild>
          <a href={checkout.checkoutUrl} rel="noreferrer" target="_blank">
            打开支付页面
            <ExternalLink aria-hidden="true" />
          </a>
        </Button>
        <Button onClick={onClose} variant="outline">
          稍后处理
        </Button>
      </div>
    );
  }

  if (checkout.type === "stripe" && checkout.stripe) {
    return (
      <div className="space-y-5">
        <div className="flex items-center gap-3 border-b pb-4">
          <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
            <ReceiptText aria-hidden="true" />
          </span>
          <span className="min-w-0">
            <strong className="block font-medium text-sm">安全支付</strong>
            <small className="block truncate text-muted-foreground text-xs">
              订单 {orderNo}
            </small>
          </span>
        </div>
        <StripeCheckoutPanel
          checkout={checkout.stripe}
          onSuccess={() => {
            toast.success("支付已提交，正在确认订单状态");
            order.refetch().catch(() => {
              toast.error("订单状态查询失败，请稍后重试");
            });
          }}
        />
      </div>
    );
  }

  return (
    <div
      aria-live="assertive"
      className="flex flex-col items-center gap-4 py-6 text-center"
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-muted">
        <Clock3 aria-hidden="true" />
      </span>
      <DialogTitle>支付状态暂不可用</DialogTitle>
      <DialogDescription>
        订单 {orderNo} 已保留，可重新查询或稍后从订单中心继续处理。
      </DialogDescription>
      <Button onClick={() => order.refetch()} variant="outline">
        重新查询
      </Button>
      <Button onClick={onClose} variant="ghost">
        关闭
      </Button>
    </div>
  );
}
