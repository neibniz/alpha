import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Switch } from "@workspace/ui/components/switch";
import { ConfirmButton } from "@/components/confirm-button";
import {
  ProTable,
  type ProTableActions,
} from "@/components/pro-table/pro-table";
import {
  deleteUserDevice,
  getUserSubscribeDevices,
  kickOfflineByUserDevice,
  updateUserDevice,
} from "@workspace/ui/services/admin/user";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { LogOut, Trash2 } from "lucide-react";
import { type ReactNode, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { IpLink, IpLocation } from "@/components/ip-link";
import { formatDate } from "@/utils/common";

export function SubscriptionDetail({
  trigger,
  userId,
  subscriptionId,
}: {
  trigger: ReactNode;
  userId: number;
  subscriptionId: number;
}) {
  const { t } = useTranslation("user");
  const [open, setOpen] = useState(false);
  const [pendingDeviceId, setPendingDeviceId] = useState<number>();
  const tableRef = useRef<ProTableActions>(null);

  async function toggleDevice(device: API.UserDevice, enabled: boolean) {
    setPendingDeviceId(device.id);
    try {
      await updateUserDevice({ ...device, enabled });
      toast.success(
        enabled
          ? t("deviceEnableSuccess", "Device enabled")
          : t("deviceDisableSuccess", "Device disabled")
      );
      tableRef.current?.refresh();
    } catch {
      toast.error(t("deviceUpdateFailed", "Failed to update device"));
    } finally {
      setPendingDeviceId(undefined);
    }
  }

  async function kickDevice(device: API.UserDevice) {
    setPendingDeviceId(device.id);
    try {
      await kickOfflineByUserDevice({ id: device.id });
      toast.success(t("kickOfflineSuccess", "Device kicked offline"));
      tableRef.current?.refresh();
    } catch {
      toast.error(t("kickOfflineFailed", "Failed to kick device offline"));
    } finally {
      setPendingDeviceId(undefined);
    }
  }

  async function deleteDevice(device: API.UserDevice) {
    setPendingDeviceId(device.id);
    try {
      await deleteUserDevice({ id: device.id });
      toast.success(t("deviceDeleteSuccess", "Device deleted"));
      tableRef.current?.refresh();
    } catch {
      toast.error(t("deviceDeleteFailed", "Failed to delete device"));
    } finally {
      setPendingDeviceId(undefined);
    }
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <SheetTrigger asChild>{trigger}</SheetTrigger>
      <SheetContent
        className="w-[700px] max-w-full md:max-w-screen-md"
        side="right"
      >
        <SheetHeader>
          <SheetTitle>{t("onlineDevices", "Online Devices")}</SheetTitle>
          <SheetDescription>
            {t(
              "onlineDevicesDescription",
              "Manage device access, current session state, and subscription usage"
            )}
          </SheetDescription>
        </SheetHeader>
        <div className="mt-4 max-h-[calc(100dvh-120px)] overflow-y-auto">
          <ProTable<API.UserDevice, Record<string, unknown>>
            action={tableRef}
            actions={{
              columnSize: 96,
              render: (row) => {
                const actions: ReactNode[] = [];
                if (row.identifier && row.online) {
                  actions.push(
                    <ConfirmButton
                      cancelText={t("cancel", "Cancel")}
                      confirmText={t("confirm", "Confirm")}
                      description={t(
                        "kickOfflineConfirm",
                        `Kick device ${row.ip} offline?`
                      )}
                      key="offline"
                      onConfirm={() => kickDevice(row)}
                      title={t("confirmOffline", "Confirm Offline")}
                      trigger={
                        <Button
                          aria-label={t("confirmOffline", "Confirm Offline")}
                          disabled={pendingDeviceId === row.id}
                          size="icon-sm"
                          title={t("confirmOffline", "Confirm Offline")}
                          variant="ghost"
                        >
                          <LogOut aria-hidden="true" className="size-4" />
                        </Button>
                      }
                    />
                  );
                }
                actions.push(
                  <ConfirmButton
                    cancelText={t("cancel", "Cancel")}
                    confirmText={t("confirm", "Confirm")}
                    description={t(
                      "deleteDeviceConfirm",
                      "Delete this device record? This cannot be undone."
                    )}
                    key="delete"
                    onConfirm={() => deleteDevice(row)}
                    title={t("deleteDevice", "Delete device")}
                    trigger={
                      <Button
                        aria-label={t("deleteDevice", "Delete device")}
                        disabled={pendingDeviceId === row.id}
                        size="icon-sm"
                        title={t("deleteDevice", "Delete device")}
                        variant="ghost"
                      >
                        <Trash2
                          aria-hidden="true"
                          className="size-4 text-destructive"
                        />
                      </Button>
                    }
                  />
                );
                return actions;
              },
            }}
            columns={[
              {
                accessorKey: "enabled",
                header: t("enable", "Enable"),
                size: 76,
                cell: ({ row }) => (
                  <Switch
                    aria-label={`${t("enable", "Enable")} ${row.original.identifier || row.original.id}`}
                    checked={row.original.enabled}
                    disabled={pendingDeviceId === row.original.id}
                    onCheckedChange={(checked) =>
                      toggleDevice(row.original, checked)
                    }
                  />
                ),
              },
              { accessorKey: "id", header: "ID", size: 72 },
              {
                accessorKey: "identifier",
                header: t("deviceIdentifier", "Device identifier"),
                size: 180,
              },
              {
                accessorKey: "user_agent",
                header: t("userAgent", "User Agent"),
                size: 180,
              },
              {
                accessorKey: "ip",
                header: "IP",
                cell: ({ row }) => <IpLink ip={row.getValue("ip")} />,
                size: 140,
              },
              {
                accessorKey: "location",
                header: t("location", "Location"),
                cell: ({ row }) => (
                  <IpLocation location={row.original.location} />
                ),
                size: 150,
              },
              {
                accessorKey: "online",
                header: t("loginStatus", "Login Status"),
                cell: ({ row }) => (
                  <Badge
                    variant={row.getValue("online") ? "default" : "destructive"}
                  >
                    {row.getValue("online")
                      ? t("online", "Online")
                      : t("offline", "Offline")}
                  </Badge>
                ),
                size: 96,
              },
              {
                accessorKey: "updated_at",
                header: t("lastSeen", "Last Seen"),
                cell: ({ row }) => formatDate(row.getValue("updated_at")),
                size: 160,
              },
            ]}
            request={async (pagination) => {
              const { data } = await getUserSubscribeDevices({
                user_id: userId,
                subscribe_id: subscriptionId,
                ...pagination,
              });
              return normalizePageResult(data.data);
            }}
          />
        </div>
      </SheetContent>
    </Sheet>
  );
}
