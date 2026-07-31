"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import { Spinner } from "@workspace/ui/components/spinner";
import { BookOpen, ChevronRight, Copy } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
  getAnnouncements,
  getCurrentUser,
  getDocumentDetail,
  getDocuments,
} from "@/lib/client-api";
import { copyText } from "@/lib/copy-text";
import { formatDate } from "@/lib/format";
import type { UserDocument } from "@/lib/models";
import { QueryFailure } from "./settings-common";

export function ResourcesSection() {
  const user = useQuery({
    queryFn: getCurrentUser,
    queryKey: ["alpha-current-user"],
  });
  const documents = useQuery({
    queryFn: getDocuments,
    queryKey: ["alpha-documents"],
  });
  const announcements = useQuery({
    queryFn: getAnnouncements,
    queryKey: ["alpha-announcements"],
  });
  const [selected, setSelected] = useState<UserDocument>();
  const detail = useQuery({
    enabled: Boolean(selected?.id),
    queryFn: () => getDocumentDetail(selected?.id || 0),
    queryKey: ["alpha-document", selected?.id],
  });
  const copyInvite = async () => {
    if (!user.data?.referCode) return;
    try {
      await copyText(user.data.referCode);
      toast.success("邀请码已复制");
    } catch {
      toast.error("邀请码复制失败");
    }
  };

  return (
    <div className="grid gap-4">
      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardTitle className="text-base">产品文档</CardTitle>
          <CardDescription>订阅接入与使用说明。</CardDescription>
        </CardHeader>
        <CardContent className="divide-y px-0 py-0">
          {(documents.data || []).map((document) => (
            <Button
              className="h-auto w-full justify-start rounded-none px-5 py-4 text-left"
              key={document.id}
              onClick={() => setSelected(document)}
              type="button"
              variant="ghost"
            >
              <BookOpen className="shrink-0 text-muted-foreground" />
              <span className="min-w-0 flex-1">
                <strong className="block truncate font-medium text-sm">
                  {document.title}
                </strong>
                <small className="block truncate font-normal text-muted-foreground text-xs">
                  {document.description}
                </small>
              </span>
              <ChevronRight className="ml-auto shrink-0" />
            </Button>
          ))}
          {!(documents.isLoading || documents.data?.length) ? (
            <p className="py-8 text-center text-muted-foreground text-sm">
              暂无公开文档。
            </p>
          ) : null}
        </CardContent>
      </Card>

      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardTitle className="text-base">最新公告</CardTitle>
          <CardDescription>账户和服务的重要变化。</CardDescription>
        </CardHeader>
        <CardContent className="divide-y px-5 py-0">
          {(announcements.data || []).map((announcement) => (
            <article className="grid gap-1 py-4" key={announcement.id}>
              <time className="text-muted-foreground text-xs">
                {formatDate(announcement.createdAt)}
              </time>
              <strong className="font-medium text-sm">
                {announcement.title}
              </strong>
              <p className="text-muted-foreground text-sm leading-relaxed">
                {announcement.content}
              </p>
            </article>
          ))}
          {!(announcements.isLoading || announcements.data?.length) ? (
            <p className="py-8 text-center text-muted-foreground text-sm">
              暂无公告。
            </p>
          ) : null}
        </CardContent>
      </Card>

      <Card className="rounded-lg py-0 shadow-xs" data-reveal>
        <CardHeader className="border-b px-5 py-5">
          <CardTitle className="text-base">邀请好友</CardTitle>
          <CardDescription>仅分享邀请码，不要分享订阅链接。</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 px-5 py-5 sm:flex-row sm:items-center">
          <code className="min-w-0 flex-1 truncate rounded-lg border bg-muted/30 px-4 py-2.5 text-sm">
            {user.data?.referCode || "暂无邀请码"}
          </code>
          <Button
            disabled={!user.data?.referCode}
            onClick={copyInvite}
            variant="outline"
          >
            <Copy />
            复制
          </Button>
        </CardContent>
      </Card>
      <Dialog
        onOpenChange={(open) => {
          if (!open) setSelected(undefined);
        }}
        open={Boolean(selected)}
      >
        <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{detail.data?.title || selected?.title}</DialogTitle>
            <DialogDescription>
              {detail.data?.description || selected?.description}
            </DialogDescription>
          </DialogHeader>
          {detail.isLoading ? (
            <div className="flex min-h-40 items-center justify-center gap-2 text-muted-foreground text-sm">
              <Spinner />
              正在加载文档
            </div>
          ) : detail.isError ? (
            <QueryFailure
              message="文档内容暂不可用"
              retry={() => detail.refetch()}
            />
          ) : (
            <div className="whitespace-pre-wrap text-sm leading-7">
              {detail.data?.content || "此文档暂无正文。"}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
