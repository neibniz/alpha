import { useNavigate } from "@tanstack/react-router";
import { Button } from "@workspace/ui/components/button";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@workspace/ui/components/command";
import { Search } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavs } from "./navs";

export function CommandMenu() {
  const [open, setOpen] = useState(false);
  const navs = useNavs();
  const navigate = useNavigate();
  const { t } = useTranslation("components");

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        setOpen((current) => !current);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const goTo = (url: string) => {
    setOpen(false);
    navigate({ to: url });
  };

  return (
    <>
      <Button
        aria-label={t("command.search", "Search pages")}
        className="admin-command-trigger"
        onClick={() => setOpen(true)}
        variant="outline"
      >
        <Search data-icon="inline-start" />
        <span className="hidden lg:inline">
          {t("command.search", "Search pages")}
        </span>
      </Button>
      <CommandDialog
        description={t(
          "command.description",
          "Search all administration pages"
        )}
        onOpenChange={setOpen}
        open={open}
        title={t("command.title", "Go to page")}
      >
        <CommandInput
          placeholder={t(
            "command.placeholder",
            "Search users, orders, logs..."
          )}
        />
        <CommandList>
          <CommandEmpty>{t("command.empty", "No matching page")}</CommandEmpty>
          {navs.map((group) => (
            <CommandGroup heading={group.title} key={group.title}>
              {group.items.map((item) => {
                const Icon = item.icon;
                return (
                  <CommandItem
                    key={item.url}
                    onSelect={() => goTo(item.url)}
                    value={`${group.title} ${item.title}`}
                  >
                    <Icon aria-hidden />
                    <span>{item.title}</span>
                    <CommandShortcut>{group.title}</CommandShortcut>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          ))}
        </CommandList>
      </CommandDialog>
    </>
  );
}
