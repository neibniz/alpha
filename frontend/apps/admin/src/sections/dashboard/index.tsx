import { useTranslation } from "react-i18next";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import Statistics from "./components/statistics";

export default function Dashboard() {
  const { t } = useTranslation("dashboard");

  return (
    <AdminPage>
      <AdminPageHeader
        description={t(
          "operationsOverviewDescription",
          "Monitor service health, traffic, and pending work in realtime"
        )}
        eyebrow={t("operations", "Operations")}
        title={t("operationsOverview", "Operations overview")}
      />
      <Statistics />
    </AdminPage>
  );
}
