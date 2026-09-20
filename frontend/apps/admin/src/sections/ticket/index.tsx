import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Drawer,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
} from "@workspace/ui/components/drawer";
import { Textarea } from "@workspace/ui/components/textarea";
import { Alert, AlertDescription } from "@workspace/ui/components/alert";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { useFormCloseGuard } from "@/components/use-form-close-guard";
import { X } from "lucide-react";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import { ConfirmButton } from "@/components/confirm-button";
import { Icon } from "@workspace/ui/composed/icon";
import { TicketImageGallery } from "@workspace/ui/composed/ticket-image-gallery";
import { TicketImagePicker } from "@workspace/ui/composed/ticket-image-picker";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import { cn } from "@workspace/ui/lib/utils";
import {
  createTicketFollow,
  getTicket,
  getTicketList,
  updateTicketStatus,
} from "@workspace/ui/services/admin/ticket";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { formatDate } from "@/utils/common";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { UserDetail } from "../user/user-detail";

export default function Page() {
  const { t } = useTranslation("ticket");

  // i18n status declarations for extraction
  // t("status.0", "Status")
  // t("status.1", "Pending Follow-up")
  // t("status.2", "Pending Reply")
  // t("status.3", "Resolved")
  // t("status.4", "Closed")

  const [ticketId, setTicketId] = useState<number | null>(null);

  const [message, setMessage] = useState("");
  const [images, setImages] = useState<string[]>([]);
  const [sending, setSending] = useState(false);

  const {
    data: ticket,
    refetch: refetchTicket,
    isPending,
    isError,
  } = useQuery({
    queryKey: ["getTicket", ticketId],
    queryFn: async () => {
      const { data } = await getTicket({
        id: ticketId ?? 0,
      });
      return data.data as API.Ticket;
    },
    enabled: !!ticketId,
    refetchInterval: 5000,
  });
  const closeGuard = useFormCloseGuard({
    open: ticketId !== null,
    dirty: Boolean(message || images.length),
    busy: sending,
    onOpenChange: (open) => {
      if (!open) {
        setTicketId(null);
        setMessage("");
        setImages([]);
      }
    },
  });

  const scrollRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const timer = setTimeout(() => {
      if (scrollRef.current) {
        scrollRef.current.children[1]?.scrollTo({
          top: scrollRef.current.children[1].scrollHeight,
          behavior: "smooth",
        });
      }
    }, 66);
    return () => clearTimeout(timer);
  }, [ticket?.follow?.length]);

  const ref = useRef<ProTableActions>(null);
  return (
    <>
      <AdminPage>
        <AdminPageHeader
          title={t("ticketList", "Ticket List")}
          description={t(
            "pageDescription",
            "Customer conversations and follow-up requests."
          )}
        />
        <ProTable<API.TicketListItem, { status: number }>
          action={ref}
          actions={{
            render(row) {
              if (row.status !== 4) {
                return [
                  <Button
                    key="reply"
                    onClick={() => setTicketId(row.id)}
                    size="sm"
                    variant="outline"
                  >
                    {t("reply", "Reply")}
                  </Button>,
                  <ConfirmButton
                    cancelText={t("cancel", "Cancel")}
                    confirmText={t("confirm", "Confirm")}
                    description={t(
                      "closeWarning",
                      "Once closed, the ticket cannot be operated on. Please proceed with caution."
                    )}
                    key="colse"
                    onConfirm={async () => {
                      await updateTicketStatus({
                        id: row.id,
                        status: 4,
                      });
                      toast.success(t("closeSuccess", "Closed successfully"));
                      ref.current?.refresh();
                    }}
                    title={t("confirmClose", "Are you sure you want to close?")}
                    trigger={
                      <Button size="sm" variant="ghost">
                        {t("close", "Close")}
                      </Button>
                    }
                  />,
                ];
              }
              return [
                <Button
                  key="check"
                  onClick={() => setTicketId(row.id)}
                  size="sm"
                >
                  {t("check", "Check")}
                </Button>,
              ];
            },
          }}
          columns={[
            {
              accessorKey: "title",
              header: t("title", "Title"),
            },
            {
              accessorKey: "user_id",
              header: t("user", "User"),
              cell: ({ row }) => <UserDetail id={row.original.user_id} />,
            },
            {
              accessorKey: "status",
              header: t("status.0", "Status"),
              cell: ({ row }) => (
                <span
                  className={cn(
                    "flex items-center gap-2 before:block before:size-1.5 before:rounded-full",
                    {
                      "before:bg-rose-500 before:ring-rose-500":
                        row.original.status === 1,
                      "before:bg-yellow-500 before:ring-yellow-500":
                        row.original.status === 2,
                      "before:bg-green-500 before:ring-green-500":
                        row.original.status === 3,
                      "before:bg-zinc-500 before:ring-zinc-500":
                        row.original.status === 4,
                    }
                  )}
                >
                  {t(`status.${row.original.status}`)}
                </span>
              ),
            },
            {
              accessorKey: "updated_at",
              header: t("updatedAt", "Updated At"),
              cell: ({ row }) => formatDate(row.getValue("updated_at")),
            },
          ]}
          params={[
            {
              key: "status",
              placeholder: t("status.0", "Status"),
              options: [
                { label: t("status.1", "Pending Follow-up"), value: "1" },
                { label: t("status.2", "Pending Reply"), value: "2" },
                { label: t("status.3", "Resolved"), value: "3" },
                {
                  label: t("close", "Close"),
                  value: "4",
                },
              ],
            },
          ]}
          request={async (pagination, filters) => {
            const { data } = await getTicketList({
              ...pagination,
              ...filters,
            });

            const result = normalizePageResult<API.TicketListItem>(data.data);
            const list = result.list.slice();

            // Client-side ordering to improve triage efficiency:
            // - Put "Pending Follow-up" (status=1) before "Pending Reply" (status=2)
            // - Within each group, sort by updated_at desc
            const statusPriority = (status: number) => {
              if (status === 1) return 0;
              if (status === 2) return 1;
              return 2;
            };
            list.sort((a, b) => {
              const pa = statusPriority(a.status);
              const pb = statusPriority(b.status);
              if (pa !== pb) return pa - pb;
              return b.updated_at - a.updated_at;
            });

            return {
              list,
              total: result.total,
            };
          }}
        />
      </AdminPage>
      <Drawer onOpenChange={closeGuard.requestOpenChange} open={!!ticketId}>
        {closeGuard.dialog}
        <DrawerContent className="mx-auto flex h-[92dvh] max-w-4xl flex-col *:select-text">
          <DrawerHeader className="relative shrink-0 border-b pr-16 text-left">
            <DrawerTitle className="break-words">
              {ticket?.title || t("ticketList", "Ticket List")}
            </DrawerTitle>
            <DrawerDescription>
              #{ticketId} ·{" "}
              {ticket
                ? t(`status.${ticket.status}`)
                : t("loading", "Loading conversation...")}
            </DrawerDescription>
            <Button
              className="absolute right-4 top-4"
              size="icon-sm"
              variant="ghost"
              aria-label={t("close", "Close")}
              disabled={sending}
              onClick={() => closeGuard.requestOpenChange(false)}
            >
              <X />
            </Button>
          </DrawerHeader>
          <ScrollArea
            className="min-h-0 flex-1 overflow-hidden"
            ref={scrollRef}
          >
            <div className="flex h-full flex-col gap-4 p-4">
              {isPending && (
                <div
                  role="status"
                  aria-label={t("loading", "Loading conversation...")}
                  className="flex flex-col gap-4"
                >
                  <Skeleton className="h-20 w-3/4" />
                  <Skeleton className="ml-auto h-20 w-3/4" />
                </div>
              )}
              {isError && (
                <Alert variant="destructive">
                  <AlertDescription>
                    {t("loadFailed", "Unable to load the conversation.")}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => refetchTicket()}
                    >
                      {t("retry", "Retry")}
                    </Button>
                  </AlertDescription>
                </Alert>
              )}
              {/* 显示工单描述作为第一条用户消息 */}
              {ticket?.description && (
                <div className="flex items-center gap-4">
                  <div className="flex min-w-0 max-w-[90%] flex-col gap-1">
                    <p className="text-muted-foreground text-sm">
                      {formatDate(ticket.created_at)}
                    </p>
                    <p className="w-fit whitespace-pre-wrap break-words rounded-lg bg-accent p-3 text-sm leading-6">
                      {ticket.description}
                    </p>
                    <TicketImageGallery images={ticket.images || []} />
                  </div>
                </div>
              )}

              {/* 显示后续跟进消息 */}
              {ticket?.follow?.map((item) => (
                <div
                  className={cn("flex items-center gap-4", {
                    "flex-row-reverse": item.from === "System",
                  })}
                  key={item.id}
                >
                  <div
                    className={cn("flex min-w-0 max-w-[90%] flex-col gap-1", {
                      "items-end": item.from === "System",
                    })}
                  >
                    <p className="text-muted-foreground text-sm">
                      {formatDate(item.created_at)}
                    </p>
                    {item.content ? (
                      <p
                        className={cn(
                          "w-fit whitespace-pre-wrap break-words rounded-lg bg-accent p-3 text-sm leading-6",
                          {
                            "bg-primary text-primary-foreground":
                              item.from === "System",
                          }
                        )}
                      >
                        {item.content}
                      </p>
                    ) : null}
                    <TicketImageGallery images={item.images || []} />
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
          {ticket && !isError && ticket.status !== 4 && (
            <DrawerFooter className="shrink-0 border-t">
              <form
                className="flex w-full flex-col gap-3"
                onSubmit={async (event) => {
                  event.preventDefault();
                  const content = message.trim();
                  if (
                    (content || images.length) &&
                    ticketId != null &&
                    !sending
                  ) {
                    setSending(true);
                    try {
                      await createTicketFollow({
                        ticket_id: ticketId,
                        type: 1,
                        content,
                        images,
                      });
                      await refetchTicket();
                      setMessage("");
                      setImages([]);
                    } catch {
                      toast.error(t("replyFailed", "Reply failed"));
                    } finally {
                      setSending(false);
                    }
                  }
                }}
              >
                <TicketImagePicker
                  addLabel={t("addImages", "Add images")}
                  disabled={sending}
                  images={images}
                  onChange={setImages}
                  onError={(error) => toast.error(error)}
                />
                <div className="flex items-center gap-2">
                  <Textarea
                    className="max-h-40 min-h-20 resize-y"
                    aria-label={t("reply", "Reply")}
                    disabled={sending}
                    onChange={(event) => setMessage(event.target.value)}
                    placeholder={t(
                      "inputPlaceholder",
                      "Enter a reply for this ticket"
                    )}
                    value={message}
                  />
                  <Button
                    aria-label={t("send", "Send")}
                    disabled={!(message.trim() || images.length) || sending}
                    type="submit"
                  >
                    <Icon
                      className={sending ? "animate-spin" : undefined}
                      icon={sending ? "mdi:loading" : "uil:navigator"}
                    />
                  </Button>
                </div>
              </form>
            </DrawerFooter>
          )}
        </DrawerContent>
      </Drawer>
    </>
  );
}
