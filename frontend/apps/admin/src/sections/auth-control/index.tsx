import { useTranslation } from "react-i18next";
import {
  AdminPage,
  AdminPageHeader,
  AdminSettingsGroup,
} from "@/components/admin-page";
import EmailSettingsForm from "./forms/email-settings-form";
import GithubForm from "./forms/github-form";

export default function AuthControl() {
  const { t } = useTranslation("auth-control");

  return (
    <AdminPage className="gap-8">
      <AdminPageHeader
        description={t(
          "pageDescription",
          "Manage email delivery and the sign-in methods available to users."
        )}
        eyebrow={t("workspace", "Access")}
        title={t("title", "Authentication")}
      />

      <AdminSettingsGroup
        description={t(
          "communicationDescription",
          "Configure verification email transport and templates."
        )}
        title={t("communicationMethods", "Communication Methods")}
      >
        <EmailSettingsForm />
      </AdminSettingsGroup>

      <AdminSettingsGroup
        description={t(
          "oauthDescription",
          "GitHub is the supported third-party sign-in provider."
        )}
        title={t("oauthMethods", "OAuth")}
      >
        <GithubForm />
      </AdminSettingsGroup>
    </AdminPage>
  );
}
