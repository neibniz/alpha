"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import { Spinner } from "@workspace/ui/components/spinner";
import { Textarea } from "@workspace/ui/components/textarea";
import { TicketImageGallery } from "@workspace/ui/composed/ticket-image-gallery";
import { TicketImagePicker } from "@workspace/ui/composed/ticket-image-picker";
import { cn } from "@workspace/ui/lib/utils";
import { Send } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { getTicketDetail, replyToTicket } from "@/lib/client-api";
import { formatDate } from "@/lib/format";
import { errorMessage } from "./settings-common";

interface TicketThreadSheetProps {
  ticketId?: number;
  onOpenChange: (open: boolean) => void;
}

const statusLabel: Record<number, string> = {
  1: "待客服处理",
  2: "等待你的回复",
  3: "已解决",
  4: "已关闭",
};

export function TicketThreadSheet({
  ticketId,
  onOpenChange,
}: TicketThreadSheetProps) {
  const queryClient = useQueryClient();
  const thread = useQuery({
    enabled: ticketId !== undefined,
    queryFn: () => getTicketDetail(ticketId || 0),
    queryKey: ["alpha-ticket", ticketId],
    refetchInterval: ticketId ? 10_000 : false,
  });
  const [content, setContent] = useState("");
  const [images, setImages] = useState<string[]>([]);
  const [sending, setSending] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const ticket = thread.data;

  useEffect(() => {
    if (!ticket) return;
    requestAnimationFrame(() => {
      scrollRef.current?.scrollTo({
        behavior: "smooth",
        top: scrollRef.current.scrollHeight,
      });
    });
  }, [ticket]);

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!(ticketId && (content.trim() || images.length)) || sending) return;
    setSending(true);
    try {
      await replyToTicket({
        content: content.trim(),
        images,
        ticketId,
      });
      setContent("");
      setImages([]);
      await Promise.all([
        thread.refetch(),
        queryClient.invalidateQueries({ queryKey: ["alpha-tickets"] }),
      ]);
      toast.success("回复已发送");
    } catch (error) {
      toast.error(errorMessage(error, "回复发送失败"));
    } finally {
      setSending(false);
    }
  };

  return (
    <Sheet
      onOpenChange={(open) => {
        if (!open) {
          setContent("");
          setImages([]);
        }
        onOpenChange(open);
      }}
      open={ticketId !== undefined}
    >
      <SheetContent className="flex w-full flex-col gap-0 p-0 sm:max-w-2xl">
        <SheetHeader className="border-b px-5 py-4 text-left">
          <div className="flex items-start justify-between gap-4 pr-8">
            <div className="min-w-0">
              <SheetTitle className="truncate">
                {ticket?.title || "工单详情"}
              </SheetTitle>
              <SheetDescription>
                {ticket
                  ? `创建于 ${formatDate(ticket.createdAt)}`
                  : "正在加载工单"}
              </SheetDescription>
            </div>
            {ticket ? (
              <Badge className="shrink-0" variant="outline">
                {statusLabel[ticket.status] || "处理中"}
              </Badge>
            ) : null}
          </div>
        </SheetHeader>

        <div
          className="min-h-0 flex-1 space-y-5 overflow-y-auto bg-muted/15 px-5 py-5"
          ref={scrollRef}
        >
          {thread.isLoading ? (
            <div className="flex min-h-56 items-center justify-center">
              <Spinner />
            </div>
          ) : null}
          {thread.isError ? (
            <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-destructive text-sm">
              工单详情加载失败
            </div>
          ) : null}
          {ticket ? (
            <>
              <TicketMessage
                content={ticket.description}
                createdAt={ticket.createdAt}
                from="User"
                images={ticket.images}
              />
              {ticket.messages.map((message) => (
                <TicketMessage
                  content={message.content}
                  createdAt={message.createdAt}
                  from={message.from}
                  images={message.images}
                  key={message.id}
                />
              ))}
            </>
          ) : null}
        </div>

        {ticket && ticket.status !== 4 ? (
          <form
            className="space-y-3 border-t bg-background p-4"
            onSubmit={submit}
          >
            <Textarea
              className="min-h-20 resize-y"
              disabled={sending}
              maxLength={10_000}
              onChange={(event) => setContent(event.target.value)}
              placeholder="输入回复内容"
              value={content}
            />
            <TicketImagePicker
              addLabel="添加图片"
              disabled={sending}
              images={images}
              onChange={setImages}
              onError={(message) => toast.error(message)}
            />
            <div className="flex justify-end">
              <Button
                disabled={sending || !(content.trim() || images.length)}
                type="submit"
              >
                {sending ? <Spinner /> : <Send className="size-4" />}
                发送回复
              </Button>
            </div>
          </form>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

function TicketMessage({
  content,
  createdAt,
  from,
  images,
}: {
  content: string;
  createdAt: number;
  from: "System" | "User";
  images: readonly string[];
}) {
  const own = from === "User";
  return (
    <article className={cn("flex", own && "justify-end")}>
      <div
        className={cn(
          "max-w-[88%] space-y-2",
          own ? "items-end" : "items-start"
        )}
      >
        <div
          className={cn(
            "flex items-center gap-2 text-muted-foreground text-xs",
            own && "justify-end"
          )}
        >
          <span>{own ? "你" : "客服"}</span>
          <time>{formatDate(createdAt)}</time>
        </div>
        {content ? (
          <p
            className={cn(
              "whitespace-pre-wrap rounded-lg border bg-card px-3 py-2 text-sm leading-6 shadow-xs",
              own && "border-primary bg-primary text-primary-foreground"
            )}
          >
            {content}
          </p>
        ) : null}
        <TicketImageGallery className={cn(own && "ml-auto")} images={images} />
      </div>
    </article>
  );
}
