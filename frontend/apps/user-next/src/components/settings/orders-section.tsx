"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@workspace/ui/components/alert-dialog";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Card } from "@workspace/ui/components/card";
import { Dialog, DialogContent } from "@workspace/ui/components/dialog";
import { Field, FieldLabel } from "@workspace/ui/components/field";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Spinner } from "@workspace/ui/components/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@workspace/ui/components/table";
import { ChevronLeft, ChevronRight, Search } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
  type OrderCheckoutKind,
  OrderCheckoutStatus,
} from "@/components/orders/order-checkout-status";
import { checkoutOrder, closeOrder, getOrders } from "@/lib/client-api";
import { formatCurrency, formatDate } from "@/lib/format";
import type { CheckoutResult, UserOrder } from "@/lib/models";
import { errorMessage, QueryFailure } from "./settings-common";

const orderStatus: Record<number, string> = {
  1: "待支付",
  2: "生效中",
  3: "已取消",
  4: "处理失败",
  5: "已完成",
};

const orderTypes: Record<number, string> = {
  1: "购买订阅",
  2: "续费订阅",
  3: "重置流量",
  4: "账户充值",
  5: "会员服务",
};

function checkoutKind(type: number): OrderCheckoutKind {
  if (type === 2) return "renewal";
  if (type === 3) return "reset";
  if (type === 4) return "recharge";
  if (type === 5) return "membership";
  return "purchase";
}

export function OrdersSection() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [status, setStatus] = useState(0);
  const [checkout, setCheckout] = useState<CheckoutResult>();
  const [activeOrder, setActiveOrder] = useState<UserOrder>();
  const [cancelOrder, setCancelOrder] = useState<UserOrder>();
  const [busyOrder, setBusyOrder] = useState("");
  const orders = useQuery({
    queryFn: () =>
      getOrders({
        page,
        search: appliedSearch,
        size: 20,
        status: status || undefined,
      }),
    queryKey: ["alpha-orders", page, status, appliedSearch],
  });
  const pages = Math.max(1, Math.ceil((orders.data?.total || 0) / 20));

  const continueOrder = async (order: UserOrder) => {
    setBusyOrder(order.orderNo);
    try {
      const result = await checkoutOrder(order.orderNo);
      setActiveOrder(order);
      setCheckout(result);
    } catch (error) {
      toast.error(errorMessage(error, "支付初始化失败"));
    } finally {
      setBusyOrder("");
    }
  };

  return (
    <div className="grid gap-4">
      <form
        className="grid gap-2 sm:grid-cols-[minmax(220px,1fr)_180px_auto]"
        onSubmit={(event) => {
          event.preventDefault();
          setPage(1);
          setAppliedSearch(search.trim());
        }}
      >
        <Field>
          <FieldLabel className="sr-only" htmlFor="order-search">
            搜索订单
          </FieldLabel>
          <InputGroup>
            <InputGroupInput
              id="order-search"
              onChange={(event) => setSearch(event.target.value)}
              placeholder="订单号或商品名称"
              value={search}
            />
            <InputGroupAddon>
              <Search />
            </InputGroupAddon>
          </InputGroup>
        </Field>
        <Select
          onValueChange={(value) => {
            setPage(1);
            setStatus(Number(value));
          }}
          value={String(status)}
        >
          <SelectTrigger aria-label="订单状态" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="0">全部状态</SelectItem>
              {Object.entries(orderStatus).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Button type="submit" variant="outline">
          查询
        </Button>
      </form>
      {orders.isError ? (
        <QueryFailure
          message="订单列表暂不可用"
          retry={() => orders.refetch()}
        />
      ) : (
        <Card className="overflow-hidden rounded-lg py-0 shadow-xs" data-reveal>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pl-4">订单号</TableHead>
                <TableHead>内容</TableHead>
                <TableHead>金额</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>时间</TableHead>
                <TableHead className="pr-4 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(orders.data?.list || []).map((order) => (
                <TableRow key={order.orderNo}>
                  <TableCell
                    className="max-w-48 pl-4 font-mono text-xs"
                    title={order.orderNo}
                  >
                    <span className="block truncate">{order.orderNo}</span>
                  </TableCell>
                  <TableCell>
                    <strong className="block font-medium">
                      {order.productName || orderTypes[order.type]}
                    </strong>
                    <small className="text-muted-foreground">
                      {orderTypes[order.type]} · {order.quantity || 1} 份
                    </small>
                  </TableCell>
                  <TableCell className="tabular-nums">
                    {formatCurrency(order.amount / 100)}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        order.status === 4
                          ? "destructive"
                          : order.status === 5
                            ? "secondary"
                            : "outline"
                      }
                    >
                      {orderStatus[order.status] || "未知"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    <time>{formatDate(order.createdAt)}</time>
                  </TableCell>
                  <TableCell className="pr-4 text-right">
                    {order.status === 1 ? (
                      <div className="flex justify-end gap-1">
                        <Button
                          disabled={busyOrder === order.orderNo}
                          onClick={() => continueOrder(order)}
                          size="sm"
                          variant="outline"
                        >
                          {busyOrder === order.orderNo ? <Spinner /> : null}
                          继续支付
                        </Button>
                        <Button
                          onClick={() => setCancelOrder(order)}
                          size="sm"
                          variant="ghost"
                        >
                          取消
                        </Button>
                      </div>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
              {!(orders.isLoading || orders.data?.list.length) ? (
                <TableRow>
                  <TableCell
                    className="h-28 text-center text-muted-foreground"
                    colSpan={6}
                  >
                    没有符合条件的订单。
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </Card>
      )}
      <div className="flex items-center justify-between gap-3">
        <Button
          disabled={page <= 1}
          onClick={() => setPage((value) => value - 1)}
          size="sm"
          variant="outline"
        >
          <ChevronLeft />
          上一页
        </Button>
        <span className="text-muted-foreground text-sm tabular-nums">
          {page} / {pages}
        </span>
        <Button
          disabled={page >= pages}
          onClick={() => setPage((value) => value + 1)}
          size="sm"
          variant="outline"
        >
          下一页
          <ChevronRight />
        </Button>
      </div>
      <Dialog
        onOpenChange={(open) => {
          if (!open) {
            setCheckout(undefined);
            setActiveOrder(undefined);
          }
        }}
        open={Boolean(checkout && activeOrder)}
      >
        <DialogContent className="sm:max-w-lg">
          {checkout && activeOrder ? (
            <OrderCheckoutStatus
              checkout={checkout}
              kind={checkoutKind(activeOrder.type)}
              onClose={() => {
                setCheckout(undefined);
                setActiveOrder(undefined);
              }}
              onComplete={async () => {
                await Promise.all([
                  queryClient.invalidateQueries({ queryKey: ["alpha-orders"] }),
                  queryClient.invalidateQueries({
                    queryKey: ["alpha-subscriptions"],
                  }),
                  queryClient.invalidateQueries({
                    queryKey: ["alpha-current-user"],
                  }),
                  queryClient.invalidateQueries({
                    queryKey: ["alpha-membership-card"],
                  }),
                ]);
              }}
              orderNo={activeOrder.orderNo}
            />
          ) : null}
        </DialogContent>
      </Dialog>
      <AlertDialog
        onOpenChange={(open) => {
          if (!open) setCancelOrder(undefined);
        }}
        open={Boolean(cancelOrder)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>取消待支付订单？</AlertDialogTitle>
            <AlertDialogDescription>
              订单 {cancelOrder?.orderNo}{" "}
              将无法继续支付。已完成的付款不会通过此操作退款。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>保留订单</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                if (!cancelOrder) return;
                try {
                  await closeOrder(cancelOrder.orderNo);
                  await queryClient.invalidateQueries({
                    queryKey: ["alpha-orders"],
                  });
                  toast.success("订单已取消");
                } catch (error) {
                  toast.error(errorMessage(error, "订单取消失败"));
                } finally {
                  setCancelOrder(undefined);
                }
              }}
            >
              确认取消
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
