"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@workspace/ui/components/empty";
import { Field, FieldLabel } from "@workspace/ui/components/field";
import { Input } from "@workspace/ui/components/input";
import { Spinner } from "@workspace/ui/components/spinner";
import { Textarea } from "@workspace/ui/components/textarea";
import { TicketImagePicker } from "@workspace/ui/composed/ticket-image-picker";
import { ChevronRight, Plus, Ticket } from "lucide-react";
import { type FormEvent, useState } from "react";
import { toast } from "sonner";
import { createTicket, getTickets } from "@/lib/client-api";
import { formatDate } from "@/lib/format";
import { errorMessage, QueryFailure } from "./settings-common";
import { TicketThreadSheet } from "./ticket-thread-sheet";

const ticketStatus: Record<number, string> = {
  1: "待客服处理",
  2: "等待回复",
  3: "已解决",
  4: "已关闭",
};

export function SupportSection() {
  const queryClient = useQueryClient();
  const tickets = useQuery({
    queryFn: getTickets,
    queryKey: ["alpha-tickets"],
  });
  const [creating, setCreating] = useState(false);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [images, setImages] = useState<string[]>([]);
  const [selectedTicketId, setSelectedTicketId] = useState<number>();
  const [submitting, setSubmitting] = useState(false);
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    try {
      await createTicket({
        description: description.trim(),
        images,
        title: title.trim(),
      });
      await queryClient.invalidateQueries({ queryKey: ["alpha-tickets"] });
      setCreating(false);
      setTitle("");
      setDescription("");
      setImages([]);
      toast.success("工单已提交");
    } catch (error) {
      toast.error(errorMessage(error, "工单提交失败"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="grid gap-4">
      <div className="flex items-center justify-between gap-4" data-reveal>
        <p className="text-muted-foreground text-sm">
          共有 {tickets.data?.length || 0} 个工单
        </p>
        <Button onClick={() => setCreating((current) => !current)}>
          <Plus />
          {creating ? "收起表单" : "新建工单"}
        </Button>
      </div>
      {creating ? (
        <form data-reveal onSubmit={submit}>
          <Card className="rounded-lg py-0 shadow-xs">
            <CardHeader className="border-b px-5 py-5">
              <CardTitle className="text-base">描述问题</CardTitle>
              <CardDescription>
                请勿粘贴完整订阅链接、密码或支付凭据。
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 px-5 py-5">
              <Field>
                <FieldLabel htmlFor="ticket-title">标题</FieldLabel>
                <Input
                  id="ticket-title"
                  maxLength={100}
                  onChange={(event) => setTitle(event.target.value)}
                  required
                  value={title}
                />
              </Field>
              <Field>
                <FieldLabel>图片</FieldLabel>
                <TicketImagePicker
                  addLabel="添加图片"
                  disabled={submitting}
                  images={images}
                  onChange={setImages}
                  onError={(message) => toast.error(message)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="ticket-description">问题描述</FieldLabel>
                <Textarea
                  className="min-h-32 resize-y"
                  id="ticket-description"
                  maxLength={5000}
                  onChange={(event) => setDescription(event.target.value)}
                  required
                  value={description}
                />
              </Field>
            </CardContent>
            <CardFooter className="justify-end border-t px-5 py-4">
              <Button
                disabled={submitting || !title.trim() || !description.trim()}
                type="submit"
              >
                {submitting ? <Spinner /> : null}
                提交工单
              </Button>
            </CardFooter>
          </Card>
        </form>
      ) : null}
      {tickets.isError ? (
        <QueryFailure
          message="工单列表暂不可用"
          retry={() => tickets.refetch()}
        />
      ) : (
        <Card className="rounded-lg py-0 shadow-xs" data-reveal>
          <CardContent className="divide-y px-0 py-0">
            {(tickets.data || []).map((ticket) => (
              <button
                className="flex w-full items-center gap-4 px-5 py-4 text-left transition-colors hover:bg-muted/40"
                key={ticket.id}
                onClick={() => setSelectedTicketId(ticket.id)}
                type="button"
              >
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                  <Ticket className="size-4" />
                </div>
                <span className="min-w-0 flex-1">
                  <strong className="block truncate font-medium text-sm">
                    {ticket.title}
                  </strong>
                  <small className="block truncate text-muted-foreground">
                    {ticket.description ||
                      `更新于 ${formatDate(ticket.updatedAt)}`}
                  </small>
                </span>
                <Badge variant="outline">
                  {ticketStatus[ticket.status] || "处理中"}
                </Badge>
                <ChevronRight className="size-4 text-muted-foreground" />
              </button>
            ))}
            {!(tickets.isLoading || tickets.data?.length) ? (
              <Empty className="min-h-56 border-0">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Ticket />
                  </EmptyMedia>
                  <EmptyTitle>暂无工单</EmptyTitle>
                  <EmptyDescription>
                    遇到问题时可在这里联系支持。
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : null}
          </CardContent>
        </Card>
      )}
      <TicketThreadSheet
        onOpenChange={(open) => {
          if (!open) setSelectedTicketId(undefined);
        }}
        ticketId={selectedTicketId}
      />
    </div>
  );
}
