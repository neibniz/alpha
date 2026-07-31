import { Button } from "@workspace/ui/components/button";
import { Switch } from "@workspace/ui/components/switch";
import { ConfirmButton } from "@workspace/ui/composed/confirm-button";
import {
  ProTable,
  type ProTableActions,
} from "@workspace/ui/composed/pro-table/pro-table";
import {
  batchDeleteDocument,
  createDocument,
  deleteDocument,
  getDocumentDetail,
  getDocumentList,
  updateDocument,
} from "@workspace/ui/services/admin/document";
import { normalizePageResult } from "@workspace/ui/utils/pagination";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { formatDate } from "@/utils/common";
import DocumentForm from "./document-form";

export default function Page() {
  const { t } = useTranslation("document");
  const [loading, setLoading] = useState(false);
  const [updatingDocumentId, setUpdatingDocumentId] = useState<number>();

  const ref = useRef<ProTableActions>(null);
  return (
    <ProTable<API.DocumentListItem, { tag: string; search: string }>
      action={ref}
      actions={{
        render(row) {
          return [
            <DocumentForm<API.UpdateDocumentRequest>
              initialValues={row}
              key="edit"
              loadInitialValues={async () => {
                try {
                  const { data } = await getDocumentDetail({ id: row.id });
                  return {
                    ...row,
                    ...(data.data || {}),
                  };
                } catch {
                  toast.error(
                    t("detailLoadFailed", "Failed to load document detail")
                  );
                  return;
                }
              }}
              loading={loading}
              onSubmit={async (values) => {
                setLoading(true);
                try {
                  await updateDocument({
                    ...row,
                    ...values,
                  });
                  toast.success(t("updateSuccess", "Updated successfully"));
                  ref.current?.refresh();
                  return true;
                } catch {
                  return false;
                } finally {
                  setLoading(false);
                }
              }}
              title={t("editDocument", "Edit Document")}
              trigger={t("edit", "Edit")}
            />,
            <ConfirmButton
              cancelText={t("cancel", "Cancel")}
              confirmText={t("confirm", "Confirm")}
              description={t(
                "deleteDescription",
                "Are you sure you want to delete this document? This action cannot be undone."
              )}
              key="delete"
              onConfirm={async () => {
                await deleteDocument({
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
                "Are you sure you want to delete this document? This action cannot be undone."
              )}
              key="delete"
              onConfirm={async () => {
                await batchDeleteDocument({
                  ids: rows.map((item) => item.id),
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
      }}
      columns={[
        {
          accessorKey: "show",
          header: t("show", "Show"),
          cell: ({ row }) => (
            <Switch
              aria-label={t("toggleShow", {
                defaultValue: "Toggle visibility for {{title}}",
                title: row.original.title,
              })}
              checked={row.original.show}
              disabled={updatingDocumentId === row.original.id}
              onCheckedChange={async (checked) => {
                setUpdatingDocumentId(row.original.id);
                try {
                  const { data } = await getDocumentDetail({
                    id: row.original.id,
                  });
                  const detail = data.data;
                  await updateDocument({
                    id: row.original.id,
                    title: detail?.title ?? row.original.title,
                    content: detail?.content ?? "",
                    tags: detail?.tags ?? row.original.tags,
                    show: checked,
                  });
                  ref.current?.refresh();
                  toast.success(
                    t("visibilityUpdateSuccess", "Visibility updated")
                  );
                } catch {
                  toast.error(t("updateFailed", "Update failed"));
                } finally {
                  setUpdatingDocumentId(undefined);
                }
              }}
            />
          ),
        },
        {
          accessorKey: "title",
          header: t("title", "Title"),
        },
        {
          accessorKey: "tags",
          header: t("tags", "Tags"),
          cell: ({ row }) => row.original.tags.join(", "),
        },
        {
          accessorKey: "updated_at",
          header: t("updatedAt", "Updated At"),
          cell: ({ row }) => formatDate(row.getValue("updated_at")),
        },
      ]}
      header={{
        title: t("DocumentList", "Document List"),
        toolbar: (
          <DocumentForm<API.CreateDocumentRequest>
            key="create"
            loading={loading}
            onSubmit={async (values) => {
              setLoading(true);
              try {
                await createDocument({
                  ...values,
                  show: false,
                });
                toast.success(t("createSuccess", "Created successfully"));
                ref.current?.refresh();
                return true;
              } catch {
                return false;
              } finally {
                setLoading(false);
              }
            }}
            title={t("createDocument", "Create Document")}
            trigger={t("create", "Create")}
          />
        ),
      }}
      params={[
        {
          key: "search",
        },
        {
          key: "tag",
          placeholder: t("tags", "Tags"),
        },
      ]}
      request={async (pagination, filter) => {
        const { data } = await getDocumentList({ ...pagination, ...filter });
        return normalizePageResult(data.data);
      }}
    />
  );
}
