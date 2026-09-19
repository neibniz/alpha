import { useTranslation } from "react-i18next";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import { Database, Globe, ShieldCheck } from "lucide-react";
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

      <Tabs className="admin-settings-tabs gap-7" defaultValue="general">
        <TabsList className="admin-page-tabs">
          <TabsTrigger value="general">
            <Globe />
            {t("basicSettings", "Basic Settings")}
          </TabsTrigger>
          <TabsTrigger value="security">
            <ShieldCheck />
            {t("userSecuritySettings", "User & Security")}
          </TabsTrigger>
          <TabsTrigger value="retention">
            <Database />
            {t("logSettings", "Log Settings")}
          </TabsTrigger>
        </TabsList>
        <TabsContent
          className="data-[state=inactive]:hidden"
          forceMount
          value="general"
        >
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
        </TabsContent>
        <TabsContent
          className="data-[state=inactive]:hidden"
          forceMount
          value="security"
        >
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
        </TabsContent>
        <TabsContent
          className="data-[state=inactive]:hidden"
          forceMount
          value="retention"
        >
          <AdminSettingsGroup
            description={t(
              "logSettingsDescription",
              "Retention and cleanup policy for operational records."
            )}
            title={t("logSettings", "Log Settings")}
          >
            <LogCleanupForm />
          </AdminSettingsGroup>
        </TabsContent>
      </Tabs>
    </AdminPage>
  );
}
