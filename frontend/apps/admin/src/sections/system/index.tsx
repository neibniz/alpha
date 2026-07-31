import { useTranslation } from "react-i18next";
import {
  AdminPage,
  AdminPageHeader,
  AdminSettingsGroup,
} from "@/components/admin-page";
import CurrencyForm from "./basic-settings/currency-form";
import PrivacyPolicyForm from "./basic-settings/privacy-policy-form";
import SiteForm from "./basic-settings/site-form";
import TosForm from "./basic-settings/tos-form";
import LogCleanupForm from "./log-cleanup/log-cleanup-form";
import InviteForm from "./user-security/invite-form";
import RegisterForm from "./user-security/register-form";
import VerifyCodeForm from "./user-security/verify-code-form";
import VerifyForm from "./user-security/verify-form";

export default function System() {
  const { t } = useTranslation("system");

  return (
    <AdminPage className="gap-8">
      <AdminPageHeader
        description={t(
          "pageDescription",
          "Configure platform identity, account policy, verification and data retention."
        )}
        eyebrow={t("workspace", "Platform")}
        title={t("title", "System settings")}
      />

      <AdminSettingsGroup
        description={t(
          "basicSettingsDescription",
          "Identity, public metadata, currency and legal documents."
        )}
        title={t("basicSettings", "Basic Settings")}
      >
        {[
          <SiteForm key="site" />,
          <CurrencyForm key="currency" />,
          <TosForm key="tos" />,
          <PrivacyPolicyForm key="privacy" />,
        ]}
      </AdminSettingsGroup>

      <AdminSettingsGroup
        description={t(
          "userSecuritySettingsDescription",
          "Registration, invitations and verification requirements."
        )}
        title={t("userSecuritySettings", "User & Security")}
      >
        {[
          <RegisterForm key="register" />,
          <InviteForm key="invite" />,
          <VerifyForm key="verify" />,
          <VerifyCodeForm key="code" />,
        ]}
      </AdminSettingsGroup>

      <AdminSettingsGroup
        description={t(
          "logSettingsDescription",
          "Retention and cleanup policy for operational records."
        )}
        title={t("logSettings", "Log Settings")}
      >
        <LogCleanupForm />
      </AdminSettingsGroup>
    </AdminPage>
  );
}
