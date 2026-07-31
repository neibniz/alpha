import { useQuery } from "@tanstack/react-query";
import { Link, useSearch } from "@tanstack/react-router";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemTitle,
} from "@workspace/ui/components/item";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Switch } from "@workspace/ui/components/switch";
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
import {
  ProTable,
  type ProTableActions,
} from "@workspace/ui/composed/pro-table/pro-table";
import { cn } from "@workspace/ui/lib/utils";
import {
  batchDeleteUser,
  createUser,
  deleteUser,
  getUserDetail,
  getUserList,
  updateUserBasicInfo,
} from "@workspace/ui/services/admin/user";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import {
  BadgeDollarSign,
  Check,
  Circle,
  Copy,
  Eye,
  EyeOff,
  Gift,
  LoaderCircle,
  LogIn,
  MoreHorizontal,
  Pencil,
  Plus,
  ReceiptText,
  RotateCcw,
  ShieldCheck,
  ShieldX,
  ShoppingBag,
  TriangleAlert,
  Trash2,
  UserCheck,
  UserRound,
  UserX,
  Users,
  WalletCards,
} from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { Display } from "@/components/display";
import { useSubscribe } from "@/stores/subscribe";
import { formatDate } from "@/utils/common";
import {
  buildUserPageSummary,
  getUserLifecycle,
  getUserWalletTotal,
  maskUserIdentifier,
} from "./user-directory-data";
import UserForm from "./user-form";
import { AuthMethodsForm } from "./user-profile/auth-methods-form";
import { BasicInfoForm } from "./user-profile/basic-info-form";
import { MembershipInfoForm } from "./user-profile/membership-info-form";
import { NotifySettingsForm } from "./user-profile/notify-settings-form";
import UserSubscription from "./user-subscription";
import { TrafficResetDialog } from "./traffic-reset-dialog";

type UserListRow = API.UserListItem;
type UserListStatus = NonNullable<API.GetUserListParams["status"]>;

const EMPTY_SUMMARY = {
  matchingTotal: 0,
  pageTotal: 0,
  enabled: 0,
  members: 0,
  attention: 0,
};

function buildBasicInfoPayload(
  user: API.User,
  enable: boolean
): API.UpdateUserBasiceInfoRequest {
  return {
    user_id: user.id,
    password: "",
    avatar: user.avatar,
    balance: user.balance,
    commission: user.commission,
    referral_percentage: user.referral_percentage,
    only_first_purchase: user.only_first_purchase,
    gift_amount: user.gift_amount,
    telegram: user.telegram,
    refer_code: user.refer_code,
    referer_id: user.referer_id,
    enable,
    is_admin: user.is_admin,
  };
}

function isUserListStatus(value?: string): value is UserListStatus {
  return value === "enabled" || value === "disabled" || value === "deleted";
}

function UserStatusCell({
  busy,
  onToggle,
  user,
}: {
  busy: boolean;
  onToggle: (checked: boolean) => Promise<void>;
  user: UserListRow;
}) {
  const { t } = useTranslation("user");
  const lifecycle = getUserLifecycle(user);
  const labels = {
    enabled: t("enabled", "Enabled"),
    disabled: t("disabled", "Disabled"),
    deleted: t("deleted", "Deleted"),
  };

  return (
    <div className="flex min-w-32 items-center justify-between gap-3">
      <span
        className={cn(
          "inline-flex items-center gap-2 text-xs",
          lifecycle === "enabled" && "text-emerald-500",
          lifecycle === "disabled" && "text-amber-500",
          lifecycle === "deleted" && "text-destructive"
        )}
      >
        <Circle className="size-2 fill-current" />
        {labels[lifecycle]}
      </span>
      <Switch
        aria-label={t("toggleStatus", "Toggle status")}
        checked={lifecycle === "enabled"}
        disabled={busy || lifecycle === "deleted"}
        onCheckedChange={onToggle}
      />
    </div>
  );
}

function UserIdentityCell({
  revealed,
  toggleReveal,
  user,
}: {
  revealed: boolean;
  toggleReveal: () => void;
  user: UserListRow;
}) {
  const { t } = useTranslation("user");
  const method = user.primary_auth_method;
  const identifier = method?.auth_identifier || `#${user.id}`;
  const authType = method?.auth_type || "user";
  const displayIdentifier = revealed
    ? identifier
    : maskUserIdentifier(identifier, authType);

  return (
    <div className="flex min-w-0 items-center gap-3">
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate font-medium text-sm">
            {displayIdentifier}
          </span>
          {method?.verified ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  aria-label={t("verified", "Verified")}
                  className="shrink-0"
                  size="icon-sm"
                  type="button"
                  variant="ghost"
                >
                  <Check />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("verified", "Verified")}</TooltipContent>
            </Tooltip>
          ) : null}
        </div>
        <div className="mt-1 flex items-center gap-2 text-[11px] text-muted-foreground">
          <span className="uppercase">{authType}</span>
          <span aria-hidden="true">·</span>
          <span className="tabular-nums">ID {user.id}</span>
          <span aria-hidden="true">·</span>
          <span className="tabular-nums">
            {formatDate(user.created_at, false)}
          </span>
        </div>
      </div>
      <div className="flex shrink-0 items-center">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              aria-label={
                revealed
                  ? t("hidePrivateData", "Hide private data")
                  : t("revealPrivateData", "Reveal private data")
              }
              onClick={toggleReveal}
              size="icon-sm"
              type="button"
              variant="ghost"
            >
              {revealed ? <EyeOff /> : <Eye />}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {revealed
              ? t("hidePrivateData", "Hide private data")
              : t("revealPrivateData", "Reveal private data")}
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              aria-label={t("copyIdentity", "Copy full identifier")}
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(identifier);
                  toast.success(t("copySuccess", "Copied successfully"));
                } catch {
                  toast.error(t("copyFailed", "Copy failed"));
                }
              }}
              size="icon-sm"
              type="button"
              variant="ghost"
            >
              <Copy />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {t("copyIdentity", "Copy full identifier")}
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}

function MembershipCell({ user }: { user: UserListRow }) {
  const { t } = useTranslation("user");
  const active = Boolean(user.is_member) && !user.deleted_at;
  const Icon = active ? ShieldCheck : ShieldX;
  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <Icon
        className={cn(
          "size-4 shrink-0",
          active ? "text-sky-500" : "text-muted-foreground"
        )}
      />
      <div className="min-w-0">
        <p className="truncate font-medium text-xs">
          {active
            ? t("memberActive", "Active member")
            : t("memberInactive", "No active membership")}
        </p>
        <p className="mt-1 truncate text-[11px] text-muted-foreground">
          {user.member_expired_at
            ? formatDate(user.member_expired_at, false)
            : t("notActivated", "Not activated")}
        </p>
      </div>
    </div>
  );
}

function WalletCell({ user }: { user: UserListRow }) {
  const { t } = useTranslation("user");
  return (
    <div className="min-w-44">
      <div className="font-medium text-sm tabular-nums">
        <Display type="currency" value={getUserWalletTotal(user)} />
      </div>
      <div className="mt-1 flex flex-wrap gap-x-2 text-[11px] text-muted-foreground tabular-nums">
        <span>
          {t("balanceShort", "Balance")}{" "}
          <Display type="currency" value={user.balance} />
        </span>
        <span>
          {t("giftShort", "Gift")}{" "}
          <Display type="currency" value={user.gift_amount} />
        </span>
        <span>
          {t("commissionShort", "Commission")}{" "}
          <Display type="currency" value={user.commission} />
        </span>
      </div>
    </div>
  );
}

function ReferralCell({ user }: { user: UserListRow }) {
  const { t } = useTranslation("user");
  return (
    <div className="min-w-0 text-xs">
      <p className="truncate font-medium">
        {user.refer_code || t("noInviteCode", "No invite code")}
      </p>
      <p className="mt-1 text-muted-foreground tabular-nums">
        {user.referer_id
          ? `${t("referer", "Referrer")} #${user.referer_id}`
          : t("directRegistration", "Direct registration")}
      </p>
    </div>
  );
}

export default function User() {
  const { t } = useTranslation("user");
  const [createLoading, setCreateLoading] = useState(false);
  const [statusUpdatingId, setStatusUpdatingId] = useState<number>();
  const [revealedUserIds, setRevealedUserIds] = useState<Set<number>>(
    () => new Set()
  );
  const [summary, setSummary] = useState(EMPTY_SUMMARY);
  const ref = useRef<ProTableActions>(null);
  const sp = useSearch({ strict: false }) as Record<string, string | undefined>;
  const { subscribes } = useSubscribe();

  const status = isUserListStatus(sp.status) ? sp.status : undefined;
  const initialFilters = {
    search: sp.search || sp.user_id || undefined,
    status,
    subscribe_id: sp.subscribe_id || undefined,
    user_subscribe_id: sp.user_subscribe_id || undefined,
  };

  const toggleIdentity = (userId: number) => {
    setRevealedUserIds((current) => {
      const next = new Set(current);
      if (next.has(userId)) next.delete(userId);
      else next.add(userId);
      return next;
    });
  };

  const updateStatus = async (userId: number, enabled: boolean) => {
    setStatusUpdatingId(userId);
    try {
      const { data } = await getUserDetail({ id: userId });
      if (!data.data) throw new Error("User detail not found");
      await updateUserBasicInfo(buildBasicInfoPayload(data.data, enabled));
      toast.success(t("updateSuccess", "Updated successfully"));
      ref.current?.refresh();
    } finally {
      setStatusUpdatingId(undefined);
    }
  };

  return (
    <AdminPage>
      <AdminPageHeader
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <TrafficResetDialog
              onReset={() => ref.current?.refresh()}
              scope="all"
              trigger={
                <Button size="sm" variant="destructive">
                  <TriangleAlert className="size-4" />
                  {t(
                    "resetAllSubscriptionTraffic",
                    "Reset all subscription traffic"
                  )}
                </Button>
              }
            />
            <UserForm<API.CreateUserRequest>
              loading={createLoading}
              onSubmit={async (values) => {
                setCreateLoading(true);
                try {
                  await createUser(values);
                  toast.success(t("createSuccess", "Created successfully"));
                  ref.current?.refresh();
                  return true;
                } catch {
                  return false;
                } finally {
                  setCreateLoading(false);
                }
              }}
              title={t("createUser", "Create user")}
              trigger={
                <span className="inline-flex items-center gap-2">
                  <Plus className="size-4" />
                  {t("createUser", "Create user")}
                </span>
              }
            />
          </div>
        }
        description={t(
          "pageDescription",
          "Manage account access, subscriptions, balances, and authentication from one directory."
        )}
        eyebrow={t("workspace", "Customers")}
        title={t("pageTitle", "User management")}
      />

      <ItemGroup className="admin-stat-strip grid grid-cols-2 overflow-hidden rounded-lg border bg-card xl:grid-cols-4">
        {[
          {
            icon: Users,
            label: t("summaryMatching", "Matching users"),
            value: summary.matchingTotal,
            tone: "text-foreground",
          },
          {
            icon: UserCheck,
            label: t("summaryEnabledPage", "Enabled on this page"),
            value: summary.enabled,
            tone: "text-emerald-500",
          },
          {
            icon: ShieldCheck,
            label: t("summaryMembersPage", "Members on this page"),
            value: summary.members,
            tone: "text-sky-500",
          },
          {
            icon: UserX,
            label: t("summaryAttentionPage", "Needs attention on this page"),
            value: summary.attention,
            tone: "text-amber-500",
          },
        ].map((item) => {
          const Icon = item.icon;
          return (
            <Item
              className="rounded-none border-0 border-border/70 border-b px-4 py-3 even:border-l [&:nth-last-child(-n+2)]:border-b-0 xl:border-b-0 xl:border-l xl:px-5 xl:py-4 xl:first:border-l-0"
              key={item.label}
              size="sm"
            >
              <ItemContent>
                <ItemDescription>{item.label}</ItemDescription>
                <ItemTitle
                  className={cn(
                    "font-semibold text-2xl tabular-nums",
                    item.tone
                  )}
                >
                  {item.value}
                </ItemTitle>
              </ItemContent>
              <ItemMedia>
                <Icon className={cn(item.tone)} />
              </ItemMedia>
            </Item>
          );
        })}
      </ItemGroup>

      <div className="min-w-0">
        <ProTable<UserListRow, API.GetUserListParams>
          action={ref}
          actions={{
            columnSize: 114,
            render: (row) => [
              <ProfileSheet
                key="profile"
                onUpdated={() => ref.current?.refresh()}
                userId={row.id}
              />,
              <SubscriptionSheet key="subscription" userId={row.id} />,
              <UserMoreActions
                key="more"
                onDeleted={() => ref.current?.refresh()}
                user={row}
              />,
            ],
            batchRender: (rows) => {
              const targets = rows.filter((row) => !row.deleted_at);
              return [
                <TrafficResetDialog
                  key="reset-traffic"
                  onReset={() => ref.current?.refresh()}
                  scope="users"
                  trigger={
                    <Button
                      disabled={targets.length === 0}
                      size="sm"
                      variant="outline"
                    >
                      <RotateCcw className="size-4" />
                      {t("resetSelectedTraffic", "Reset traffic")}
                    </Button>
                  }
                  userIds={targets.map((row) => row.id)}
                />,
                <ConfirmButton
                  cancelText={t("cancel", "Cancel")}
                  confirmText={t("confirm", "Confirm")}
                  description={t(
                    "batchDeleteDescription",
                    "The selected active accounts will be deleted. This action cannot be undone."
                  )}
                  key="delete"
                  onConfirm={async () => {
                    if (targets.length === 0) return;
                    await batchDeleteUser({
                      ids: targets.map((row) => row.id),
                    });
                    toast.success(
                      t("batchDeleteSuccess", "Selected users deleted")
                    );
                    ref.current?.refresh();
                  }}
                  title={t("batchDeleteTitle", "Delete selected users")}
                  trigger={
                    <Button
                      disabled={targets.length === 0}
                      size="sm"
                      variant="destructive"
                    >
                      <Trash2 className="size-4" />
                      {t("deleteSelected", "Delete selected")}
                    </Button>
                  }
                />,
              ];
            },
          }}
          columns={[
            {
              accessorKey: "enable",
              header: t("status", "Status"),
              size: 132,
              cell: ({ row }) => (
                <UserStatusCell
                  busy={statusUpdatingId === row.original.id}
                  onToggle={(checked) => updateStatus(row.original.id, checked)}
                  user={row.original}
                />
              ),
            },
            {
              accessorKey: "primary_auth_method",
              header: t("identity", "Identity"),
              size: 260,
              cell: ({ row }) => (
                <UserIdentityCell
                  revealed={revealedUserIds.has(row.original.id)}
                  toggleReveal={() => toggleIdentity(row.original.id)}
                  user={row.original}
                />
              ),
            },
            {
              accessorKey: "is_member",
              header: t("membership", "Membership"),
              size: 170,
              cell: ({ row }) => <MembershipCell user={row.original} />,
            },
            {
              id: "wallet",
              header: t("wallet", "Wallet"),
              size: 210,
              cell: ({ row }) => <WalletCell user={row.original} />,
            },
            {
              accessorKey: "refer_code",
              header: t("referral", "Referral"),
              size: 155,
              cell: ({ row }) => <ReferralCell user={row.original} />,
            },
          ]}
          header={{ title: t("directoryTitle", "User directory") }}
          initialFilters={initialFilters}
          key={[
            initialFilters.search || "",
            initialFilters.subscribe_id || "",
            initialFilters.user_subscribe_id || "",
            initialFilters.status || "",
          ].join("-")}
          params={[
            {
              key: "status",
              placeholder: t("allStatuses", "All statuses"),
              type: "select",
              options: [
                { label: t("enabled", "Enabled"), value: "enabled" },
                { label: t("disabled", "Disabled"), value: "disabled" },
                { label: t("deleted", "Deleted"), value: "deleted" },
              ],
            },
            {
              key: "search",
              className: "sm:w-72",
              placeholder: t(
                "searchPlaceholder",
                "Search email, phone, invite code, or user ID"
              ),
            },
            {
              key: "subscribe_id",
              placeholder: t("subscription", "Subscription"),
              options: subscribes?.map((item) => ({
                label: item.name!,
                value: String(item.id!),
              })),
            },
            {
              key: "user_subscribe_id",
              placeholder: t("userSubscriptionId", "User subscription ID"),
            },
          ]}
          request={async (pagination, filter) => {
            const { data } = await getUserList({
              ...pagination,
              ...filter,
            });
            const result = normalizePageResult<UserListRow>(data.data);
            setSummary(buildUserPageSummary(result.list, result.total));
            return result;
          }}
          texts={{ actions: t("actions", "Actions") }}
        />
      </div>
    </AdminPage>
  );
}

function UserMoreActions({
  onDeleted,
  user,
}: {
  onDeleted: () => void;
  user: UserListRow;
}) {
  const { t } = useTranslation("user");
  const logs = [
    {
      label: t("orderList", "Orders"),
      to: "/dashboard/order",
      icon: ShoppingBag,
    },
    {
      label: t("loginLogs", "Login logs"),
      to: "/dashboard/log/login",
      icon: LogIn,
    },
    {
      label: t("balanceLogs", "Balance logs"),
      to: "/dashboard/log/balance",
      icon: WalletCards,
    },
    {
      label: t("commissionLogs", "Commission logs"),
      to: "/dashboard/log/commission",
      icon: BadgeDollarSign,
    },
    {
      label: t("giftLogs", "Gift logs"),
      to: "/dashboard/log/gift",
      icon: Gift,
    },
  ] as const;

  return (
    <DropdownMenu modal={false}>
      <Tooltip>
        <TooltipTrigger asChild>
          <DropdownMenuTrigger asChild>
            <Button
              aria-label={t("moreActions", "More actions")}
              size="icon-sm"
              variant="ghost"
            >
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
        </TooltipTrigger>
        <TooltipContent>{t("moreActions", "More actions")}</TooltipContent>
      </Tooltip>
      <DropdownMenuContent align="end" className="w-52">
        {logs.map((item) => {
          const Icon = item.icon;
          return (
            <DropdownMenuItem asChild key={item.to}>
              <Link search={{ user_id: String(user.id) }} to={item.to}>
                <Icon className="size-4" />
                {item.label}
              </Link>
            </DropdownMenuItem>
          );
        })}
        {!user.deleted_at ? (
          <>
            <DropdownMenuSeparator />
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "deleteDescription",
                "This account will be removed and can no longer sign in."
              )}
              onConfirm={async () => {
                await deleteUser({ id: user.id });
                toast.success(t("deleteSuccess", "Deleted successfully"));
                onDeleted();
              }}
              title={t("confirmDelete", "Delete user")}
              trigger={
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  onSelect={(event) => event.preventDefault()}
                >
                  <Trash2 className="size-4" />
                  {t("delete", "Delete")}
                </DropdownMenuItem>
              }
            />
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ProfileSheet({
  userId,
  onUpdated,
}: {
  userId: number;
  onUpdated?: () => void;
}) {
  const { t } = useTranslation("user");
  const [open, setOpen] = useState(false);
  const {
    data: user,
    isLoading,
    refetch,
  } = useQuery({
    enabled: open,
    queryKey: ["user", userId],
    queryFn: async () => {
      const { data } = await getUserDetail({ id: userId });
      return data.data as API.User;
    },
  });

  const refetchAll = async () => {
    await refetch();
    onUpdated?.();
  };

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <Tooltip>
        <TooltipTrigger asChild>
          <SheetTrigger asChild>
            <Button
              aria-label={t("editProfile", "Edit profile")}
              size="icon-sm"
              variant="ghost"
            >
              <Pencil />
            </Button>
          </SheetTrigger>
        </TooltipTrigger>
        <TooltipContent>{t("editProfile", "Edit profile")}</TooltipContent>
      </Tooltip>
      <SheetContent
        className="w-[760px] max-w-full gap-0 md:max-w-screen-lg"
        side="right"
      >
        <SheetHeader className="border-b px-5 py-4">
          <SheetTitle>
            {t("userProfile", "User profile")} · #{userId}
          </SheetTitle>
          <SheetDescription>
            {t(
              "profileDescription",
              "Review account access, membership, notifications, and authentication methods."
            )}
          </SheetDescription>
        </SheetHeader>
        {isLoading ? (
          <div className="flex min-h-72 items-center justify-center text-muted-foreground">
            <LoaderCircle className="size-5 animate-spin" />
          </div>
        ) : null}
        {user ? (
          <ScrollArea className="h-[calc(100dvh-89px)]">
            <div className="space-y-5 p-5">
              <UserProfileSummary user={user} />
              <Tabs defaultValue="basic">
                <TabsList className="grid h-auto w-full grid-cols-2 gap-1 p-1 md:grid-cols-4">
                  <TabsTrigger value="basic">
                    {t("basicInfoTitle", "Basic info")}
                  </TabsTrigger>
                  <TabsTrigger value="membership">
                    {t("membership", "Membership")}
                  </TabsTrigger>
                  <TabsTrigger value="notify">
                    {t("notifySettingsTitle", "Notifications")}
                  </TabsTrigger>
                  <TabsTrigger value="auth">
                    {t("authMethodsTitle", "Authentication")}
                  </TabsTrigger>
                </TabsList>
                <TabsContent className="mt-4" value="basic">
                  <BasicInfoForm refetch={refetchAll} user={user} />
                </TabsContent>
                <TabsContent className="mt-4" value="membership">
                  <MembershipInfoForm refetch={refetchAll} user={user} />
                </TabsContent>
                <TabsContent className="mt-4" value="notify">
                  <NotifySettingsForm refetch={refetchAll} user={user} />
                </TabsContent>
                <TabsContent className="mt-4" value="auth">
                  <AuthMethodsForm refetch={refetchAll} user={user} />
                </TabsContent>
              </Tabs>
            </div>
          </ScrollArea>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

function UserProfileSummary({ user }: { user: API.User }) {
  const { t } = useTranslation("user");
  const lifecycle = getUserLifecycle(user);
  const primary = user.auth_methods?.[0];
  const identity = primary
    ? maskUserIdentifier(primary.auth_identifier, primary.auth_type)
    : `#${user.id}`;

  return (
    <section className="overflow-hidden rounded-lg border bg-card">
      <div className="flex min-w-0 items-center gap-3 border-b px-4 py-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
          <UserRound className="size-4" />
        </span>
        <div className="min-w-0">
          <p className="truncate font-medium text-sm">{identity}</p>
          <p className="mt-1 text-muted-foreground text-xs">
            {primary?.auth_type || t("unknown", "Unknown")} · ID {user.id}
          </p>
        </div>
        <Badge className="ml-auto" variant="outline">
          {lifecycle === "enabled"
            ? t("enabled", "Enabled")
            : lifecycle === "disabled"
              ? t("disabled", "Disabled")
              : t("deleted", "Deleted")}
        </Badge>
      </div>
      <div className="grid sm:grid-cols-3">
        <div className="border-b px-4 py-3 sm:border-r sm:border-b-0">
          <p className="text-muted-foreground text-[11px]">
            {t("wallet", "Wallet")}
          </p>
          <p className="mt-1 font-medium text-sm tabular-nums">
            <Display type="currency" value={getUserWalletTotal(user)} />
          </p>
        </div>
        <div className="border-b px-4 py-3 sm:border-r sm:border-b-0">
          <p className="text-muted-foreground text-[11px]">
            {t("membership", "Membership")}
          </p>
          <p className="mt-1 truncate font-medium text-sm">
            {user.is_member
              ? formatDate(user.member_expired_at, false)
              : t("memberInactive", "No active membership")}
          </p>
        </div>
        <div className="px-4 py-3">
          <p className="text-muted-foreground text-[11px]">
            {t("createdAt", "Created")}
          </p>
          <p className="mt-1 truncate font-medium text-sm">
            {formatDate(user.created_at)}
          </p>
        </div>
      </div>
    </section>
  );
}

function SubscriptionSheet({ userId }: { userId: number }) {
  const { t } = useTranslation("user");
  const [open, setOpen] = useState(false);
  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <Tooltip>
        <TooltipTrigger asChild>
          <SheetTrigger asChild>
            <Button
              aria-label={t("manageSubscriptions", "Manage subscriptions")}
              size="icon-sm"
              variant="ghost"
            >
              <ReceiptText />
            </Button>
          </SheetTrigger>
        </TooltipTrigger>
        <TooltipContent>
          {t("manageSubscriptions", "Manage subscriptions")}
        </TooltipContent>
      </Tooltip>
      <SheetContent
        className="w-[1180px] max-w-full gap-0 bg-muted/10 md:max-w-[min(1180px,96vw)]"
        side="right"
      >
        <SheetHeader className="border-b px-5 py-4">
          <SheetTitle>
            {t("subscriptionList", "Subscriptions")} · #{userId}
          </SheetTitle>
          <SheetDescription>
            {t(
              "subscriptionDescription",
              "Review access, usage, traffic history, devices, and subscription lifecycle."
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-89px)]">
          <div className="p-4 lg:p-5">
            <UserSubscription userId={userId} />
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
