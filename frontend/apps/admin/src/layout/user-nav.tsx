import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@workspace/ui/components/avatar";
import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { LanguageSwitch } from "@workspace/ui/composed/language-switch";
import { ChevronDown, LogOut } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useGlobalStore } from "@/stores/global";
import { Logout } from "@/utils/common";
import TimezoneSwitch from "./timezone-switch";

export function UserNav() {
  const { t } = useTranslation("auth");
  const user = useGlobalStore((store) => store.user);
  if (!user) return null;
  const identifier =
    user.auth_methods?.[0]?.auth_identifier ||
    t("consoleBadge", "Admin Console");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          aria-label={t("accountMenu", "Account menu")}
          className="h-9 gap-2 px-1.5"
          variant="ghost"
        >
          <Avatar className="size-7">
            <AvatarImage alt="" src={user.avatar || undefined} />
            <AvatarFallback className="bg-muted text-xs">
              {identifier.slice(0, 1).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <span className="hidden max-w-28 truncate text-xs 2xl:block">
            {identifier}
          </span>
          <ChevronDown className="hidden size-3 text-muted-foreground sm:block" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel className="grid gap-1">
          <span className="text-xs font-normal text-muted-foreground">
            {t("consoleBadge", "Admin Console")}
          </span>
          <span className="break-all text-sm">{identifier}</span>
        </DropdownMenuLabel>
        <div className="flex flex-wrap items-center gap-1 p-1 xl:hidden">
          <div className="lg:hidden">
            <LanguageSwitch />
          </div>
          <TimezoneSwitch />
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={Logout}>
          <LogOut />
          {t("logout", "Logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
