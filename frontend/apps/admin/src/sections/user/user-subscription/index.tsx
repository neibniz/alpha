import { Link } from "@tanstack/react-router";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@workspace/ui/components/empty";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemTitle,
} from "@workspace/ui/components/item";
import { Progress } from "@workspace/ui/components/progress";
import { Skeleton } from "@workspace/ui/components/skeleton";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { ConfirmButton } from "@workspace/ui/composed/confirm-button";
import { SubscriptionTrafficDialog } from "@workspace/ui/composed/subscription-traffic-dialog";
import { cn } from "@workspace/ui/lib/utils";
import {
  createUserSubscribe,
  deleteUserSubscribe,
  getUserSubscribe,
  getUserSubscribeAccess,
  getUserSubscribeById,
  getUserSubscribeTrafficSeries,
  resetUserSubscribeToken,
  resetUserSubscribeTraffic,
  toggleUserSubscribeStatus,
  updateUserSubscribe,
} from "@workspace/ui/services/admin/user";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import {
  Activity,
  CalendarClock,
  Check,
  ChevronRight,
  CircleGauge,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  FileClock,
  Gauge,
  KeyRound,
  Link2,
  LoaderCircle,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Router,
  Settings2,
  ShieldOff,
  Smartphone,
  Trash2,
} from "lucide-react";
import {
  type ReactNode,
  type RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Display } from "@/components/display";
import { OrderLink } from "@/components/order-link";
import { useGlobalStore } from "@/stores/global";
import { formatDate } from "@/utils/common";
import { IssueSubscriptionDialog } from "./issue-subscription-dialog";
import { SubscriptionDetail } from "./subscription-detail";
import {
  SubscriptionForm,
  type SubscriptionFormValues,
} from "./subscription-form";

type UserSubscribeRow = API.UserSubscribeListItem;
type StatusMeta = { key: string; fallback: string; className: string };

const DEFAULT_STATUS_META: StatusMeta = {
  key: "statusPending",
  fallback: "Pending",
  className: "border-amber-500/30 bg-amber-500/10 text-amber-600",
};

const STATUS_META: Record<number, StatusMeta> = {
  0: DEFAULT_STATUS_META,
  1: {
    key: "statusActive",
    fallback: "Active",
    className: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600",
  },
  2: {
    key: "statusFinished",
    fallback: "Traffic exhausted",
    className: "border-orange-500/30 bg-orange-500/10 text-orange-600",
  },
  3: {
    key: "statusExpired",
    fallback: "Expired",
    className: "border-destructive/30 bg-destructive/10 text-destructive",
  },
  4: {
    key: "statusDeducted",
    fallback: "Deducted",
    className: "border-muted-foreground/30 bg-muted text-muted-foreground",
  },
  5: {
    key: "statusStopped",
    fallback: "Stopped",
    className: "border-destructive/30 bg-destructive/10 text-destructive",
  },
};

export default function UserSubscription({ userId }: { userId: number }) {
  const { t } = useTranslation("user");
  const [rows, setRows] = useState<UserSubscribeRow[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedId, setSelectedId] = useState<number>();
  const [detail, setDetail] = useState<API.UserSubscribeDetail>();
  const [access, setAccess] = useState<API.UserSubscribeAccess>();
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [accessLoading, setAccessLoading] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const [trafficOpen, setTrafficOpen] = useState(false);
  const [revision, setRevision] = useState(0);
  const deviceTriggerRef = useRef<HTMLButtonElement>(null);
  const resetTokenRef = useRef<HTMLButtonElement>(null);
  const resetTrafficRef = useRef<HTMLButtonElement>(null);
  const toggleStatusRef = useRef<HTMLButtonElement>(null);
  const deleteRef = useRef<HTMLButtonElement>(null);
  const { getUserSubscribe: buildSubscriptionUrls } = useGlobalStore();

  const refresh = useCallback(() => setRevision((value) => value + 1), []);

  useEffect(() => {
    let active = true;
    setLoading(true);
    getUserSubscribe({ user_id: userId, page: 1, size: 100 })
      .then(({ data }) => {
        if (!active) return;
        const result = normalizePageResult<UserSubscribeRow>(data.data);
        setRows(result.list);
        setTotal(result.total);
        setSelectedId((current) => {
          if (current && result.list.some((row) => row.id === current)) {
            return current;
          }
          return result.list[0]?.id;
        });
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [revision, userId]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(undefined);
      return;
    }
    let active = true;
    setDetailLoading(true);
    setDetail(undefined);
    setAccess(undefined);
    setRevealed(false);
    getUserSubscribeById({ id: selectedId })
      .then(({ data }) => {
        if (active) setDetail(data.data);
      })
      .finally(() => {
        if (active) setDetailLoading(false);
      });
    return () => {
      active = false;
    };
  }, [revision, selectedId]);

  const selectedRow = useMemo(
    () => rows.find((row) => row.id === selectedId),
    [rows, selectedId]
  );
  const summary = useMemo(() => {
    let active = 0;
    let attention = 0;
    let used = 0;
    for (const row of rows) {
      const status = row.effective_status ?? row.status;
      if (status === 1) active += 1;
      if (status === 2 || status === 3 || status === 5) attention += 1;
      used += Math.max(0, row.used_traffic ?? row.download + row.upload);
    }
    return { active, attention, used };
  }, [rows]);

  const loadAccess = useCallback(async () => {
    if (access) return access;
    if (!selectedId || accessLoading) return undefined;
    setAccessLoading(true);
    try {
      const { data } = await getUserSubscribeAccess({
        user_subscribe_id: selectedId,
      });
      setAccess(data.data);
      return data.data;
    } finally {
      setAccessLoading(false);
    }
  }, [access, accessLoading, selectedId]);

  const subscriptionUrls = useMemo(() => {
    if (!(access?.short && access.token)) return [];
    return buildSubscriptionUrls(access.short, access.token);
  }, [access, buildSubscriptionUrls]);

  const updateSubscription = useCallback(
    async (row: UserSubscribeRow, values: SubscriptionFormValues) => {
      await updateUserSubscribe({
        user_subscribe_id: row.id,
        subscribe_id: values.subscribe_id ?? row.subscribe_id,
        traffic: values.traffic ?? 0,
        expired_at: values.expired_at ?? 0,
        upload: values.upload ?? 0,
        download: values.download ?? 0,
      });
      toast.success(t("updateSuccess", "Updated successfully"));
      refresh();
      return true;
    },
    [refresh, t]
  );

  const createDirectSubscription = useCallback(
    async (values: SubscriptionFormValues) => {
      await createUserSubscribe({
        user_id: userId,
        subscribe_id: values.subscribe_id ?? 0,
        traffic: values.traffic ?? 0,
        expired_at: values.expired_at ?? 0,
      });
      toast.success(t("createSuccess", "Created successfully"));
      refresh();
      return true;
    },
    [refresh, t, userId]
  );

  const onIssued = useCallback(() => {
    refresh();
    window.setTimeout(refresh, 1200);
  }, [refresh]);

  if (loading && rows.length === 0) {
    return <WorkspaceSkeleton />;
  }

  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <header className="flex flex-col gap-4 border-b px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h2 className="font-semibold text-base">
              {t("subscriptionWorkspace", "Subscription workspace")}
            </h2>
            <Badge variant="outline">{total}</Badge>
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-muted-foreground text-xs">
            <span className="inline-flex items-center gap-1.5">
              <Check className="size-3.5 text-emerald-500" />
              {t("activeCount", "{{count}} active", { count: summary.active })}
            </span>
            <span className="inline-flex items-center gap-1.5">
              <CircleGauge className="size-3.5 text-amber-500" />
              {t("attentionCount", "{{count}} need attention", {
                count: summary.attention,
              })}
            </span>
            <span className="inline-flex items-center gap-1.5 tabular-nums">
              <Activity className="size-3.5" />
              <Display type="traffic" value={summary.used} />
              <span>{t("used", "used")}</span>
            </span>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                aria-label={t("refresh", "Refresh")}
                onClick={refresh}
                size="icon-sm"
                variant="outline"
              >
                <RefreshCw
                  className={cn("size-4", loading && "animate-spin")}
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("refresh", "Refresh")}</TooltipContent>
          </Tooltip>
          <SubscriptionForm
            buttonSize="sm"
            buttonVariant="outline"
            onSubmit={createDirectSubscription}
            title={t("directGrant", "Direct grant")}
            trigger={
              <span className="inline-flex items-center gap-2">
                <Plus className="size-4" />
                {t("directGrant", "Direct grant")}
              </span>
            }
          />
          <IssueSubscriptionDialog onIssued={onIssued} userId={userId} />
        </div>
      </header>

      {rows.length === 0 ? (
        <Empty className="min-h-[460px] border-0">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Router />
            </EmptyMedia>
            <EmptyTitle>{t("noSubscriptions", "No subscriptions")}</EmptyTitle>
            <EmptyDescription>
              {t(
                "noSubscriptionsDescription",
                "Issue a traceable order or create a direct grant for this user."
              )}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="grid min-h-[610px] md:grid-cols-[280px_minmax(0,1fr)]">
          <aside className="border-b bg-muted/15 md:border-r md:border-b-0">
            <div className="flex items-center justify-between border-b px-3 py-2.5">
              <p className="font-medium text-xs uppercase text-muted-foreground">
                {t("subscriptions", "Subscriptions")}
              </p>
              <span className="text-muted-foreground text-[11px]">
                {rows.length}/{total}
              </span>
            </div>
            <div className="max-h-64 overflow-y-auto p-2 md:max-h-[565px]">
              {rows.map((row) => (
                <SubscriptionNavItem
                  active={row.id === selectedId}
                  key={row.id}
                  onSelect={() => setSelectedId(row.id)}
                  row={row}
                />
              ))}
            </div>
          </aside>

          <main className="min-w-0">
            {detailLoading && !detail ? (
              <DetailSkeleton />
            ) : selectedRow && detail ? (
              <>
                <SubscriptionHeader
                  detail={detail}
                  onCopy={async () => {
                    const current = await loadAccess();
                    if (current?.short && current.token) {
                      await copyText(
                        buildSubscriptionUrls(current.short, current.token)[0]
                      );
                      toast.success(t("copySuccess", "Copied successfully"));
                    }
                  }}
                  onDevices={() => deviceTriggerRef.current?.click()}
                  onOpenTraffic={() => setTrafficOpen(true)}
                  resetTokenRef={resetTokenRef}
                  resetTrafficRef={resetTrafficRef}
                  row={selectedRow}
                  toggleStatusRef={toggleStatusRef}
                  deleteRef={deleteRef}
                  updateSubscription={updateSubscription}
                />

                <Tabs
                  className="min-w-0"
                  defaultValue="overview"
                  onValueChange={(value) => {
                    if (value === "access") {
                      loadAccess().catch(() => undefined);
                    }
                  }}
                >
                  <div className="border-b px-4">
                    <TabsList className="h-11 w-full justify-start gap-5 rounded-none bg-transparent p-0">
                      <TabsTrigger
                        className="h-11 rounded-none border-b-2 border-transparent px-0 data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                        value="overview"
                      >
                        {t("overview", "Overview")}
                      </TabsTrigger>
                      <TabsTrigger
                        className="h-11 rounded-none border-b-2 border-transparent px-0 data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                        value="presets"
                      >
                        {t("presets", "Presets")}
                        <span className="ml-1 text-muted-foreground">
                          {detail.presets?.length || 0}
                        </span>
                      </TabsTrigger>
                      <TabsTrigger
                        className="h-11 rounded-none border-b-2 border-transparent px-0 data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                        value="access"
                      >
                        {t("accessLinks", "Access links")}
                      </TabsTrigger>
                      <TabsTrigger
                        className="h-11 rounded-none border-b-2 border-transparent px-0 data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none"
                        value="activity"
                      >
                        {t("activity", "Activity")}
                      </TabsTrigger>
                    </TabsList>
                  </div>

                  <TabsContent className="m-0 p-4" value="overview">
                    <OverviewPanel detail={detail} />
                  </TabsContent>
                  <TabsContent className="m-0 p-4" value="presets">
                    <PresetPanel detail={detail} />
                  </TabsContent>
                  <TabsContent className="m-0 p-4" value="access">
                    <AccessPanel
                      access={access}
                      loading={accessLoading}
                      onReveal={() => setRevealed((value) => !value)}
                      presets={detail.presets || []}
                      revealed={revealed}
                      urls={subscriptionUrls}
                    />
                  </TabsContent>
                  <TabsContent className="m-0 p-4" value="activity">
                    <ActivityPanel row={selectedRow} userId={userId} />
                  </TabsContent>
                </Tabs>

                <SubscriptionTrafficDialog
                  loadSeries={async (period) => {
                    const { data } = await getUserSubscribeTrafficSeries({
                      period,
                      user_id: userId,
                      user_subscribe_id: selectedRow.id,
                    });
                    return data.data;
                  }}
                  onOpenChange={setTrafficOpen}
                  open={trafficOpen}
                  queryKey={[
                    "admin-subscription-traffic",
                    userId,
                    selectedRow.id,
                  ]}
                  title={`${t("trafficChart", "Traffic chart")} · ${subscriptionName(selectedRow)}`}
                />

                <SubscriptionDetail
                  subscriptionId={selectedRow.id}
                  trigger={<Button className="hidden" ref={deviceTriggerRef} />}
                  userId={userId}
                />

                <HiddenActions
                  deleteRef={deleteRef}
                  refresh={refresh}
                  resetTokenRef={resetTokenRef}
                  resetTrafficRef={resetTrafficRef}
                  row={selectedRow}
                  toggleStatusRef={toggleStatusRef}
                />
              </>
            ) : null}
          </main>
        </div>
      )}
    </div>
  );
}

function SubscriptionNavItem({
  active,
  onSelect,
  row,
}: {
  active: boolean;
  onSelect: () => void;
  row: UserSubscribeRow;
}) {
  const { t } = useTranslation("user");
  const status = row.effective_status ?? row.status;
  const usage = usagePercent(row.used_traffic, row.traffic);
  return (
    <Item
      asChild
      className={cn("mb-1 w-full cursor-pointer", active && "bg-background")}
      size="sm"
      variant={active ? "outline" : "default"}
    >
      <button onClick={onSelect} type="button">
        <ItemContent>
          <ItemTitle className="truncate">{subscriptionName(row)}</ItemTitle>
          <ItemDescription className="tabular-nums">
            #{row.id} · {statusLabel(status, t)}
          </ItemDescription>
          <Progress className="mt-2 h-1" value={usage} />
          <div className="flex justify-between text-[10px] text-muted-foreground tabular-nums">
            <span>{usage}%</span>
            <span>
              {row.expire_time
                ? formatDate(row.expire_time, false)
                : t("permanent", "Permanent")}
            </span>
          </div>
        </ItemContent>
        <ItemActions>
          <ChevronRight
            className={cn(
              "text-muted-foreground transition-transform",
              active && "translate-x-0.5 text-foreground"
            )}
          />
        </ItemActions>
      </button>
    </Item>
  );
}

function SubscriptionHeader({
  deleteRef,
  detail,
  onCopy,
  onDevices,
  onOpenTraffic,
  resetTokenRef,
  resetTrafficRef,
  row,
  toggleStatusRef,
  updateSubscription,
}: {
  detail: API.UserSubscribeDetail;
  row: UserSubscribeRow;
  onCopy: () => Promise<void>;
  onDevices: () => void;
  onOpenTraffic: () => void;
  resetTokenRef: RefObject<HTMLButtonElement | null>;
  resetTrafficRef: RefObject<HTMLButtonElement | null>;
  toggleStatusRef: RefObject<HTMLButtonElement | null>;
  deleteRef: RefObject<HTMLButtonElement | null>;
  updateSubscription: (
    row: UserSubscribeRow,
    values: SubscriptionFormValues
  ) => Promise<boolean>;
}) {
  const { t } = useTranslation("user");
  const status = detail.effective_status ?? detail.status;
  return (
    <div className="flex flex-col gap-4 border-b px-4 py-4 xl:flex-row xl:items-start xl:justify-between">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="truncate font-semibold text-lg">
            {detail.product_name || detail.subscribe?.name || `#${detail.id}`}
          </h3>
          <StatusBadge status={status} />
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-muted-foreground text-xs">
          <span className="tabular-nums">
            {t("subscriptionId", "Subscription ID")} #{detail.id}
          </span>
          {detail.order_no ? (
            <span className="inline-flex items-center gap-1">
              {t("order", "Order")}
              <OrderLink orderId={detail.order_no} />
            </span>
          ) : (
            <span>{t("directGrant", "Direct grant")}</span>
          )}
          <span>
            {t("started", "Started")} {formatDate(detail.start_time, false)}
          </span>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        <IconAction
          icon={Copy}
          label={t("copySubscription", "Copy subscription")}
          onClick={() => {
            onCopy().catch(() => undefined);
          }}
        />
        <IconAction
          icon={Activity}
          label={t("trafficChart", "Traffic chart")}
          onClick={onOpenTraffic}
        />
        <IconAction
          icon={Smartphone}
          label={t("onlineDevices", "Devices")}
          onClick={onDevices}
        />
        <SubscriptionForm
          buttonSize="icon-sm"
          buttonVariant="outline"
          initialData={row}
          onSubmit={(values) => updateSubscription(row, values)}
          title={t("editSubscription", "Edit subscription")}
          trigger={<Pencil className="size-4" />}
        />
        <DropdownMenu>
          <Tooltip>
            <TooltipTrigger asChild>
              <DropdownMenuTrigger asChild>
                <Button
                  aria-label={t("moreActions", "More actions")}
                  size="icon-sm"
                  variant="outline"
                >
                  <MoreHorizontal />
                </Button>
              </DropdownMenuTrigger>
            </TooltipTrigger>
            <TooltipContent>{t("moreActions", "More actions")}</TooltipContent>
          </Tooltip>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                resetTrafficRef.current?.click();
              }}
            >
              <RotateCcw />
              {t("resetTraffic", "Reset traffic")}
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={(event) => {
                event.preventDefault();
                resetTokenRef.current?.click();
              }}
            >
              <KeyRound />
              {t("resetToken", "Reset access token")}
            </DropdownMenuItem>
            {[0, 1, 5].includes(row.status) ? (
              <DropdownMenuItem
                onSelect={(event) => {
                  event.preventDefault();
                  toggleStatusRef.current?.click();
                }}
              >
                <ShieldOff />
                {row.status === 5
                  ? t("resumeSubscribe", "Resume subscription")
                  : t("stopSubscribe", "Stop subscription")}
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive"
              onSelect={(event) => {
                event.preventDefault();
                deleteRef.current?.click();
              }}
            >
              <Trash2 />
              {t("delete", "Delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}

function OverviewPanel({ detail }: { detail: API.UserSubscribeDetail }) {
  const { t } = useTranslation("user");
  const usage = usagePercent(detail.used_traffic, detail.traffic);
  return (
    <div className="space-y-5">
      <section>
        <div className="flex items-end justify-between gap-4">
          <div>
            <p className="text-muted-foreground text-xs">
              {t("trafficConsumption", "Traffic consumption")}
            </p>
            <p className="mt-1 font-semibold text-2xl tabular-nums">
              <Display type="traffic" value={detail.used_traffic} />
            </p>
          </div>
          <div className="text-right text-muted-foreground text-xs tabular-nums">
            <p>{usage}%</p>
            <p className="mt-1">
              {detail.traffic > 0 ? (
                <Display type="traffic" value={detail.traffic} />
              ) : (
                t("unlimited", "Unlimited")
              )}
            </p>
          </div>
        </div>
        <Progress className="mt-3 h-2" value={usage} />
        <div className="mt-3 grid grid-cols-2 divide-x rounded-md border sm:grid-cols-4">
          <Metric
            icon={Activity}
            label={t("download", "Download")}
            value={<Display type="traffic" value={detail.download} />}
          />
          <Metric
            icon={Gauge}
            label={t("upload", "Upload")}
            value={<Display type="traffic" value={detail.upload} />}
          />
          <Metric
            icon={CircleGauge}
            label={t("speedLimit", "Speed limit")}
            value={
              <Display
                type="trafficSpeed"
                value={detail.subscribe?.speed_limit}
              />
            }
          />
          <Metric
            icon={Smartphone}
            label={t("deviceLimit", "Device limit")}
            value={
              <Display
                type="number"
                unlimited
                value={detail.subscribe?.device_limit}
              />
            }
          />
        </div>
      </section>

      <section className="grid gap-px overflow-hidden rounded-md border bg-border sm:grid-cols-2">
        <Definition
          icon={CalendarClock}
          label={t("expireTime", "Expires")}
          value={
            detail.expire_time
              ? formatDate(detail.expire_time)
              : t("permanent", "Permanent")
          }
        />
        <Definition
          icon={RefreshCw}
          label={t("lastUpdated", "Last updated")}
          value={formatDate(detail.updated_at)}
        />
        <Definition
          icon={Settings2}
          label={t("productSnapshot", "Product snapshot")}
          value={`${detail.subscribe?.name || detail.product_name} · #${detail.subscribe_id}`}
        />
        <Definition
          icon={FileClock}
          label={t("note", "Note")}
          value={detail.note || t("noNote", "No note")}
        />
      </section>
    </div>
  );
}

function PresetPanel({ detail }: { detail: API.UserSubscribeDetail }) {
  const { t } = useTranslation("user");
  const presets = detail.presets || [];
  if (presets.length === 0) {
    return (
      <Empty className="min-h-72 border-0">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Settings2 />
          </EmptyMedia>
          <EmptyTitle>{t("noPresets", "No presets")}</EmptyTitle>
          <EmptyDescription>
            {t(
              "noPresetsDescription",
              "This user has not created any protocol or node presets for the subscription."
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }
  return (
    <div className="divide-y rounded-md border">
      {presets.map((preset) => (
        <div className="p-4" key={preset.id}>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <p className="truncate font-medium text-sm">{preset.name}</p>
                <Badge variant={preset.enabled ? "outline" : "secondary"}>
                  {preset.enabled
                    ? t("enabled", "Enabled")
                    : t("disabled", "Disabled")}
                </Badge>
              </div>
              <p className="mt-1 text-muted-foreground text-xs">
                {preset.mode === "profile"
                  ? `${t("protocolProfile", "Protocol profile")}: ${preset.profile_key}`
                  : t("customNodeSelection", "Custom node selection")}
              </p>
            </div>
            <code className="rounded-xs bg-muted px-2 py-1 text-[11px]">
              {preset.preset_key}
            </code>
          </div>
          {preset.items?.length ? (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {preset.items.map((item) => (
                <span
                  className="rounded-sm border bg-muted/30 px-2 py-1 text-xs"
                  key={`${item.node_id}-${item.inbound_alias}`}
                >
                  {item.display_name ||
                    `${item.inbound_alias} · #${item.node_id}`}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}

function AccessPanel({
  access,
  loading,
  onReveal,
  presets,
  revealed,
  urls,
}: {
  access?: API.UserSubscribeAccess;
  loading: boolean;
  onReveal: () => void;
  presets: API.UserSubscribePreset[];
  revealed: boolean;
  urls: string[];
}) {
  const { t } = useTranslation("user");
  if (loading) {
    return (
      <div className="flex min-h-64 items-center justify-center text-muted-foreground">
        <LoaderCircle className="size-5 animate-spin" />
      </div>
    );
  }
  if (!access) {
    return (
      <Empty className="min-h-64 border-0">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Link2 />
          </EmptyMedia>
          <EmptyTitle>
            {t("accessUnavailable", "Access unavailable")}
          </EmptyTitle>
        </EmptyHeader>
      </Empty>
    );
  }
  const presetLinks = presets
    .filter((preset) => preset.enabled)
    .flatMap((preset) =>
      urls.slice(0, 1).map((url) => ({
        label: preset.name,
        url: `${url}${url.includes("?") ? "&" : "?"}preset=${encodeURIComponent(preset.preset_key)}`,
      }))
    );
  const links = [
    ...urls.map((url, index) => ({
      label:
        urls.length > 1
          ? t("subscriptionDomain", "Subscription domain {{index}}", {
              index: index + 1,
            })
          : t("defaultSubscriptionLink", "Default subscription link"),
      url,
    })),
    ...presetLinks,
  ];
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="font-medium text-sm">
            {t("subscriptionCredentials", "Subscription credentials")}
          </p>
          <p className="mt-1 text-muted-foreground text-xs">
            {t(
              "subscriptionCredentialsDescription",
              "Links are hidden by default. Copy actions always use the full value."
            )}
          </p>
        </div>
        <Button onClick={onReveal} size="sm" variant="outline">
          {revealed ? <EyeOff /> : <Eye />}
          {revealed ? t("hide", "Hide") : t("reveal", "Reveal")}
        </Button>
      </div>
      <div className="divide-y rounded-md border">
        {links.map((item) => (
          <div
            className="flex min-w-0 items-center gap-3 px-3 py-3"
            key={`${item.label}-${item.url}`}
          >
            <Link2 className="size-4 shrink-0 text-muted-foreground" />
            <div className="min-w-0 flex-1">
              <p className="text-muted-foreground text-[11px]">{item.label}</p>
              <p className="mt-1 truncate font-mono text-xs">
                {revealed ? item.url : maskSubscriptionUrl(item.url)}
              </p>
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  aria-label={t("copy", "Copy")}
                  onClick={async () => {
                    await copyText(item.url);
                    toast.success(t("copySuccess", "Copied successfully"));
                  }}
                  size="icon-sm"
                  variant="ghost"
                >
                  <Copy />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("copy", "Copy")}</TooltipContent>
            </Tooltip>
          </div>
        ))}
      </div>
    </div>
  );
}

function ActivityPanel({
  row,
  userId,
}: {
  row: UserSubscribeRow;
  userId: number;
}) {
  const { t } = useTranslation("user");
  const links = [
    {
      icon: FileClock,
      label: t("subscriptionLogs", "Access logs"),
      description: t(
        "subscriptionLogsDescription",
        "Subscription access history"
      ),
      to: "/dashboard/log/subscribe" as const,
    },
    {
      icon: RotateCcw,
      label: t("resetLogs", "Reset logs"),
      description: t("resetLogsDescription", "Traffic reset audit history"),
      to: "/dashboard/log/reset-subscribe" as const,
    },
    {
      icon: Activity,
      label: t("trafficStats", "Traffic statistics"),
      description: t(
        "trafficStatsDescription",
        "Aggregated subscription usage"
      ),
      to: "/dashboard/log/subscribe-traffic" as const,
    },
    {
      icon: Gauge,
      label: t("trafficDetails", "Traffic details"),
      description: t("trafficDetailsDescription", "Raw server traffic records"),
      to: "/dashboard/log/traffic-details" as const,
    },
  ];
  return (
    <div className="grid gap-2 sm:grid-cols-2">
      {links.map((item) => {
        const Icon = item.icon;
        return (
          <Link
            className="group flex items-center gap-3 rounded-md border p-3 transition-colors hover:bg-muted/40"
            key={item.to}
            search={{ user_id: userId, user_subscribe_id: row.id }}
            to={item.to}
          >
            <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
              <Icon className="size-4" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block font-medium text-sm">{item.label}</span>
              <span className="mt-1 block truncate text-muted-foreground text-xs">
                {item.description}
              </span>
            </span>
            <ExternalLink className="size-3.5 text-muted-foreground group-hover:text-foreground" />
          </Link>
        );
      })}
    </div>
  );
}

function HiddenActions({
  deleteRef,
  refresh,
  resetTokenRef,
  resetTrafficRef,
  row,
  toggleStatusRef,
}: {
  row: UserSubscribeRow;
  refresh: () => void;
  resetTokenRef: RefObject<HTMLButtonElement | null>;
  resetTrafficRef: RefObject<HTMLButtonElement | null>;
  toggleStatusRef: RefObject<HTMLButtonElement | null>;
  deleteRef: RefObject<HTMLButtonElement | null>;
}) {
  const { t } = useTranslation("user");
  return (
    <>
      <ConfirmButton
        cancelText={t("cancel", "Cancel")}
        confirmText={t("confirm", "Confirm")}
        description={t(
          "resetTokenDescription",
          "The current subscription links will stop working immediately."
        )}
        onConfirm={async () => {
          await resetUserSubscribeToken({ user_subscribe_id: row.id });
          toast.success(t("resetTokenSuccess", "Access token reset"));
          refresh();
        }}
        title={t("confirmResetToken", "Reset access token")}
        trigger={<Button className="hidden" ref={resetTokenRef} />}
      />
      <ConfirmButton
        cancelText={t("cancel", "Cancel")}
        confirmText={t("confirm", "Confirm")}
        description={t(
          "resetSubscriptionTrafficDescription",
          "Current upload and download counters will be cleared. Historical traffic remains available."
        )}
        onConfirm={async () => {
          await resetUserSubscribeTraffic({ user_subscribe_id: row.id });
          toast.success(t("trafficResetSuccess", "Traffic reset successfully"));
          refresh();
        }}
        title={t("resetTraffic", "Reset traffic")}
        trigger={<Button className="hidden" ref={resetTrafficRef} />}
      />
      <ConfirmButton
        cancelText={t("cancel", "Cancel")}
        confirmText={t("confirm", "Confirm")}
        description={
          row.status === 5
            ? t("resumeSubscribeDescription", "Resume this subscription?")
            : t(
                "stopSubscribeDescription",
                "The user will lose access until the subscription is resumed."
              )
        }
        onConfirm={async () => {
          await toggleUserSubscribeStatus({ user_subscribe_id: row.id });
          toast.success(t("updateSuccess", "Updated successfully"));
          refresh();
        }}
        title={
          row.status === 5
            ? t("confirmResumeSubscribe", "Resume subscription")
            : t("confirmStopSubscribe", "Stop subscription")
        }
        trigger={<Button className="hidden" ref={toggleStatusRef} />}
      />
      <ConfirmButton
        cancelText={t("cancel", "Cancel")}
        confirmText={t("delete", "Delete")}
        description={t(
          "deleteSubscriptionDescription",
          "This permanently deletes the subscription and cannot be undone."
        )}
        onConfirm={async () => {
          await deleteUserSubscribe({ user_subscribe_id: row.id });
          toast.success(t("deleteSuccess", "Deleted successfully"));
          refresh();
        }}
        title={t("confirmDelete", "Delete subscription")}
        trigger={<Button className="hidden" ref={deleteRef} />}
      />
    </>
  );
}

function StatusBadge({ status }: { status: number }) {
  const { t } = useTranslation("user");
  const meta = STATUS_META[status] ?? DEFAULT_STATUS_META;
  return (
    <Badge className={cn("font-normal", meta.className)} variant="outline">
      {t(meta.key, meta.fallback)}
    </Badge>
  );
}

function IconAction({
  icon: Icon,
  label,
  onClick,
}: {
  icon: typeof Copy;
  label: string;
  onClick: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          aria-label={label}
          onClick={onClick}
          size="icon-sm"
          variant="outline"
        >
          <Icon />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

function Metric({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof Activity;
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="min-w-0 p-3">
      <p className="flex items-center gap-1.5 text-muted-foreground text-[11px]">
        <Icon className="size-3.5" />
        {label}
      </p>
      <div className="mt-1 truncate font-medium text-sm tabular-nums">
        {value}
      </div>
    </div>
  );
}

function Definition({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof CalendarClock;
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="flex min-w-0 items-start gap-3 bg-background p-3">
      <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
      <div className="min-w-0">
        <p className="text-muted-foreground text-[11px]">{label}</p>
        <div className="mt-1 truncate text-sm">{value}</div>
      </div>
    </div>
  );
}

function WorkspaceSkeleton() {
  return (
    <div className="overflow-hidden rounded-lg border">
      <div className="flex items-center justify-between border-b p-4">
        <div className="space-y-2">
          <Skeleton className="h-5 w-44" />
          <Skeleton className="h-3 w-64" />
        </div>
        <Skeleton className="h-8 w-36" />
      </div>
      <div className="grid min-h-[610px] md:grid-cols-[280px_minmax(0,1fr)]">
        <div className="space-y-2 border-r p-3">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton className="h-20 w-full" key={index} />
          ))}
        </div>
        <DetailSkeleton />
      </div>
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="space-y-5 p-5">
      <Skeleton className="h-7 w-56" />
      <Skeleton className="h-3 w-80" />
      <Skeleton className="h-11 w-full" />
      <Skeleton className="h-44 w-full" />
      <Skeleton className="h-28 w-full" />
    </div>
  );
}

function subscriptionName(row: UserSubscribeRow) {
  return row.product_name || row.subscribe?.name || `#${row.id}`;
}

function statusLabel(
  status: number,
  t: (key: string, fallback: string) => string
) {
  const meta = STATUS_META[status] ?? DEFAULT_STATUS_META;
  return t(meta.key, meta.fallback);
}

function usagePercent(used = 0, limit = 0) {
  if (limit <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((used / limit) * 100)));
}

function maskSubscriptionUrl(value: string) {
  try {
    const url = new URL(value);
    const token = url.searchParams.get("token");
    if (token) {
      url.searchParams.set("token", `${token.slice(0, 6)}••••••••`);
    }
    return url.toString();
  } catch {
    return value.length > 24 ? `${value.slice(0, 18)}••••••••` : value;
  }
}

async function copyText(value?: string) {
  if (!value) throw new Error("Nothing to copy");
  await navigator.clipboard.writeText(value);
}
