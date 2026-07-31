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
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { useTranslation } from "react-i18next";
import { useGlobalStore } from "@/stores/global";
import { Logout } from "@/utils/common";

export function UserNav() {
  const { t } = useTranslation("auth");
  const { user } = useGlobalStore();

  if (user) {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={t("accountMenu", "Account menu")}
            className="rounded-md border"
            size="icon"
            variant="ghost"
          >
            <Avatar className="size-7">
              <AvatarImage alt={user?.avatar ?? ""} src={user?.avatar ?? ""} />
              <AvatarFallback className="rounded-sm bg-primary/10 font-semibold text-primary text-xs">
                {user?.auth_methods?.[0]?.auth_identifier
                  .toUpperCase()
                  .charAt(0)}
              </AvatarFallback>
            </Avatar>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56" forceMount>
          <DropdownMenuLabel className="font-normal">
            <div className="flex flex-col space-y-1">
              <p className="font-medium text-sm leading-none">
                {user?.auth_methods?.[0]?.auth_identifier}
              </p>
              {/* <p className='text-xs leading-none text-muted-foreground'>ID: {user?.id}</p> */}
            </div>
          </DropdownMenuLabel>
          {/* <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem>
              Profile
              <DropdownMenuShortcut>⇧⌘P</DropdownMenuShortcut>
            </DropdownMenuItem>
            <DropdownMenuItem>
              Billing
              <DropdownMenuShortcut>⌘B</DropdownMenuShortcut>
            </DropdownMenuItem>
            <DropdownMenuItem>
              Settings
              <DropdownMenuShortcut>⌘S</DropdownMenuShortcut>
            </DropdownMenuItem>
            <DropdownMenuItem>New Team</DropdownMenuItem>
          </DropdownMenuGroup> */}
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={Logout}>
            {t("logout", "Logout")}
            <DropdownMenuShortcut>⇧⌘Q</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }
}
