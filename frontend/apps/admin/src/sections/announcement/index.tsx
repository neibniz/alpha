import { Button } from "@workspace/ui/components/button";
import { Switch } from "@workspace/ui/components/switch";
import { ConfirmButton } from "@workspace/ui/composed/confirm-button";
import {
  ProTable,
  type ProTableActions,
} from "@workspace/ui/composed/pro-table/pro-table";
import {
  createAnnouncement,
  deleteAnnouncement,
  getAnnouncement,
  getAnnouncementList,
  updateAnnouncement,
} from "@workspace/ui/services/admin/announcement";
import { formatDate } from "@workspace/ui/utils/formatting";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import NoticeForm from "./notice-form";

export default function Page() {
  const { t } = useTranslation("announcement");
  const [loading, setLoading] = useState(false);
  const ref = useRef<ProTableActions>(null);

  return (
    <ProTable<API.AnnouncementListItem, { search: string; show: boolean }>
      action={ref}
      actions={{
        render(row) {
          return [
            <NoticeForm<API.UpdateAnnouncementRequest>
              initialValues={row}
              key="edit"
              loadInitialValues={async () => {
                try {
                  const { data } = await getAnnouncement({ id: row.id });
                  return {
                    ...row,
                    ...(data.data || {}),
                  };
                } catch {
                  toast.error(
                    t("detailLoadFailed", "Failed to load announcement detail")
                  );
                  return;
                }
              }}
              loading={loading}
              onSubmit={async (values) => {
                setLoading(true);
                try {
                  await updateAnnouncement({
                    ...row,
                    ...values,
                  });
                  toast.success(t("updateSuccess", "Updated successfully"));
                  ref.current?.refresh();
                  setLoading(false);
                  return true;
                } catch {
                  setLoading(false);
                  return false;
                }
              }}
              title={t("editAnnouncement", "Edit Announcement")}
              trigger={t("edit", "Edit")}
            />,
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "deleteDescription",
                "This action cannot be undone."
              )}
              key="delete"
              onConfirm={async () => {
                await deleteAnnouncement({
                  id: row.id,
                });
                toast.success(t("deleteSuccess", "Deleted successfully"));
                ref.current?.refresh();
              }}
              title={t("confirmDelete", "Confirm Delete")}
              trigger={
                <Button variant="destructive">{t("delete", "Delete")}</Button>
              }
            />,
          ];
        },
        batchRender(rows) {
          return [
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "deleteDescription",
                "This action cannot be undone."
              )}
              key="delete"
              onConfirm={async () => {
                for (const element of rows) {
                  await deleteAnnouncement({
                    id: element.id!,
                  });
                }
                toast.success(t("deleteSuccess", "Deleted successfully"));
                ref.current?.refresh();
              }}
              title={t("confirmDelete", "Confirm Delete")}
              trigger={
                <Button variant="destructive">{t("delete", "Delete")}</Button>
              }
            />,
          ];
        },
      }}
      columns={[
        {
          accessorKey: "show",
          header: t("show", "Show"),
          cell: ({ row }) => (
            <Switch
              aria-label={t("toggleShow", "Toggle visibility for {{title}}", {
                title: row.original.title,
              })}
              checked={Boolean(row.getValue("show"))}
              onCheckedChange={async (checked) => {
                await updateAnnouncement({
                  id: row.original.id,
                  show: checked,
                });
                ref.current?.refresh();
              }}
            />
          ),
        },
        {
          accessorKey: "pinned",
          header: t("pinned", "Pinned"),
          cell: ({ row }) => (
            <Switch
              aria-label={t("togglePinned", "Toggle pin for {{title}}", {
                title: row.original.title,
              })}
              checked={Boolean(row.getValue("pinned"))}
              onCheckedChange={async (checked) => {
                await updateAnnouncement({
                  id: row.original.id,
                  pinned: checked,
                });
                ref.current?.refresh();
              }}
            />
          ),
        },
        {
          accessorKey: "popup",
          header: t("popup", "Popup"),
          cell: ({ row }) => (
            <Switch
              aria-label={t("togglePopup", "Toggle popup for {{title}}", {
                title: row.original.title,
              })}
              checked={Boolean(row.getValue("popup"))}
              onCheckedChange={async (checked) => {
                await updateAnnouncement({
                  id: row.original.id,
                  popup: checked,
                });
                ref.current?.refresh();
              }}
            />
          ),
        },
        {
          accessorKey: "title",
          header: t("title", "Title"),
        },
        {
          accessorKey: "updated_at",
          header: t("updatedAt", "Updated At"),
          cell: ({ row }) => formatDate(row.getValue("updated_at")) || "—",
        },
      ]}
      header={{
        title: t("announcementList", "Announcement List"),
        toolbar: (
          <NoticeForm<API.CreateAnnouncementRequest>
            loading={loading}
            onSubmit={async (values) => {
              setLoading(true);
              try {
                await createAnnouncement(values);
                toast.success(t("createSuccess", "Created successfully"));
                ref.current?.refresh();
                setLoading(false);
                return true;
              } catch {
                setLoading(false);
                return false;
              }
            }}
            title={t("createAnnouncement", "Create Announcement")}
            trigger={t("create", "Create")}
          />
        ),
      }}
      params={[
        {
          key: "show",
          placeholder: t("show", "Show"),
          options: [
            { label: t("show", "Show"), value: "true" },
            { label: t("hide", "Hide"), value: "false" },
          ],
        },
        { key: "search" },
      ]}
      request={async (pagination, filter) => {
        const { data } = await getAnnouncementList({
          ...pagination,
          ...filter,
        });
        return normalizePageResult(data.data);
      }}
    />
  );
}
