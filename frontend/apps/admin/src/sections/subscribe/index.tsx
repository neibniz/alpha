"use client";

import { useTranslation } from "react-i18next";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import ConfigForm from "./config-form";
import { ProtocolForm } from "./protocol-form";

export default function Subscribe() {
  const { t } = useTranslation("subscribe");

  return (
    <AdminPage className="subscribe-page">
      <AdminPageHeader
        description={t(
          "page.description",
          "Manage subscription delivery policy, public endpoints, and client-specific output templates"
        )}
        eyebrow={t("page.workspace", "Infrastructure")}
        title={t("page.title", "Subscription configuration")}
      />

      <ConfigForm />
      <ProtocolForm />
    </AdminPage>
  );
}
