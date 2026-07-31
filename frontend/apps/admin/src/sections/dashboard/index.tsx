import { Badge } from "@workspace/ui/components/badge";
import { Activity } from "lucide-react";
import { useTranslation } from "react-i18next";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import Statistics from "./components/statistics";

export default function Dashboard() {
  const { t } = useTranslation("dashboard");

  return (
    <AdminPage>
      <AdminPageHeader
        actions={
          <Badge className="gap-1.5" variant="outline">
            <Activity aria-hidden />
            {t("liveMonitoring", "Live monitoring")}
          </Badge>
        }
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
