import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { useTheme } from "@workspace/ui/integrations/theme";
import { cn } from "@workspace/ui/lib/utils";
import { Check, Moon, Sun } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";

export function ThemeSwitch() {
  const { t } = useTranslation("components");
  const { theme, setTheme } = useTheme();

  /* Update theme-color meta tag
   * when theme is updated */
  useEffect(() => {
    const themeColor = theme === "dark" ? "#020817" : "#fff";
    const metaThemeColor = document.querySelector("meta[name='theme-color']");
    if (metaThemeColor) metaThemeColor.setAttribute("content", themeColor);
  }, [theme]);

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button className="scale-95 rounded-full" size="icon" variant="ghost">
          <Sun className="dark:-rotate-90 size-[1.2rem] rotate-0 scale-100 transition-all dark:scale-0" />
          <Moon className="absolute size-[1.2rem] rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
          <span className="sr-only">{t("theme.toggle", "Toggle theme")}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => setTheme("light")}>
          {t("theme.light", "Light")}{" "}
          <Check
            className={cn("ms-auto", theme !== "light" && "hidden")}
            size={14}
          />
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("dark")}>
          {t("theme.dark", "Dark")}
          <Check
            className={cn("ms-auto", theme !== "dark" && "hidden")}
            size={14}
          />
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme("system")}>
          {t("theme.system", "System")}
          <Check
            className={cn("ms-auto", theme !== "system" && "hidden")}
            size={14}
          />
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
