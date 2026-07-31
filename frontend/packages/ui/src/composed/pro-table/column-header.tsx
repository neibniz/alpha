import { flexRender, type Header } from "@tanstack/react-table";
import { Button } from "@workspace/ui/components/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@workspace/ui/components/dropdown-menu";
import { cn } from "@workspace/ui/lib/utils";
import {
  ArrowDownIcon,
  ArrowDownUpIcon,
  ArrowUpIcon,
  EyeOffIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";

interface ColumnHeaderProps<TData, TValue>
  extends React.HTMLAttributes<HTMLDivElement> {
  header: Header<TData, TValue>;
  text?: Partial<{
    asc: string;
    desc: string;
    hide: string;
  }>;
}

export function ColumnHeader<TData, TValue>({
  header,
  className,
  text,
}: ColumnHeaderProps<TData, TValue>) {
  const { t } = useTranslation("components");
  const column = header.column;
  const title = header.isPlaceholder
    ? null
    : flexRender(header.column.columnDef.header, header.getContext());
  if (!column.getCanSort()) {
    return <div className={cn(className)}>{title}</div>;
  }

  return (
    <div className={cn("flex w-full items-center", className)}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            className="!bg-transparent flex h-8 w-full justify-start p-0 text-sm"
            variant="ghost"
          >
            <span>{title}</span>
            {column.getIsSorted() === "desc" ? (
              <ArrowDownIcon className="ml-2 h-4 w-4" />
            ) : column.getIsSorted() === "asc" ? (
              <ArrowUpIcon className="ml-2 h-4 w-4" />
            ) : (
              <ArrowDownUpIcon className="ml-2 h-4 w-4" />
            )}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={() => column.toggleSorting(false)}>
            <ArrowUpIcon className="mr-2 h-3.5 w-3.5 text-muted-foreground/70" />
            {text?.asc || t("table.asc", "ASC")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => column.toggleSorting(true)}>
            <ArrowDownIcon className="mr-2 h-3.5 w-3.5 text-muted-foreground/70" />
            {text?.desc || t("table.desc", "DESC")}
          </DropdownMenuItem>
          {column.getCanHide() && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => column.toggleVisibility(false)}>
                <EyeOffIcon className="mr-2 h-3.5 w-3.5 text-muted-foreground/70" />
                {text?.hide || t("table.hide", "Hide")}
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
