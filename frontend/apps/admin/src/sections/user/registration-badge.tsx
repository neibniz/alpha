import { Badge } from "@workspace/ui/components/badge";
import { Github, Mail, UserRound, CircleHelp } from "lucide-react";
import { useTranslation } from "react-i18next";

export function RegistrationBadge({ method }: { method?: string }) {
  const { t } = useTranslation("user");
  if (method === "github")
    return (
      <Badge variant="secondary">
        <Github />
        {t("registrationGithub", "GitHub 注册")}
      </Badge>
    );
  if (method === "email")
    return (
      <Badge variant="outline">
        <Mail />
        {t("registrationEmail", "邮箱注册")}
      </Badge>
    );
  if (method === "admin")
    return (
      <Badge variant="outline">
        <UserRound />
        {t("registrationAdmin", "后台创建")}
      </Badge>
    );
  return (
    <Badge variant="outline">
      <CircleHelp />
      {t("registrationUnknown", "来源未知")}
    </Badge>
  );
}
