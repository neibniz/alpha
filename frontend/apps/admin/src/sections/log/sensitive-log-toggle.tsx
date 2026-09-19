import { Button } from "@workspace/ui/components/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import { Label } from "@workspace/ui/components/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Switch } from "@workspace/ui/components/switch";
import type { ProTableActions } from "@/components/pro-table/pro-table";
import { Eye, ShieldCheck } from "lucide-react";
import {
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";

export type SensitiveLogReason = NonNullable<
  API.FilterLoginLogParams["reveal_reason"]
>;

export type SensitiveLogVisibility = {
  enabled: boolean;
  reason?: SensitiveLogReason;
};

export const MASKED_LOG_VISIBILITY: SensitiveLogVisibility = {
  enabled: false,
};

const REVEAL_DURATION_MS = 60_000;
const REVEAL_REASONS: SensitiveLogReason[] = [
  "customer_support",
  "account_verification",
  "security_incident",
  "billing_dispute",
  "compliance",
];

export function sensitiveLogQuery(
  visibility: SensitiveLogVisibility
): Pick<API.FilterLoginLogParams, "reveal_sensitive" | "reveal_reason"> {
  if (!(visibility.enabled && visibility.reason)) return {};
  return {
    reveal_sensitive: true,
    reveal_reason: visibility.reason,
  };
}

export function useSensitiveLogVisibility(
  tableAction: RefObject<ProTableActions | null>
) {
  const [visibility, setVisibility] = useState<SensitiveLogVisibility>(
    MASKED_LOG_VISIBILITY
  );
  const visibilityRef = useRef(visibility);
  const updateVisibility = useCallback(
    (next: SensitiveLogVisibility) => {
      visibilityRef.current = next;
      tableAction.current?.clear();
      setVisibility(next);
      window.setTimeout(() => tableAction.current?.refresh(), 0);
    },
    [tableAction]
  );
  return { updateVisibility, visibility, visibilityRef };
}

export function SensitiveLogToggle({
  onChange,
  value,
}: {
  onChange: (value: SensitiveLogVisibility) => void;
  value: SensitiveLogVisibility;
}) {
  const { t } = useTranslation("log");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [reason, setReason] = useState<SensitiveLogReason>();
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    if (!value.enabled) return;
    const timer = window.setTimeout(
      () => onChangeRef.current(MASKED_LOG_VISIBILITY),
      REVEAL_DURATION_MS
    );
    return () => window.clearTimeout(timer);
  }, [value.enabled]);

  const handleCheckedChange = (checked: boolean) => {
    if (!checked) {
      onChange(MASKED_LOG_VISIBILITY);
      return;
    }
    setReason(undefined);
    setDialogOpen(true);
  };

  const reveal = () => {
    if (!reason) return;
    onChange({ enabled: true, reason });
    setDialogOpen(false);
  };

  return (
    <>
      <div className="flex h-8 items-center gap-2 rounded-md border bg-background px-2.5">
        <ShieldCheck className="size-3.5 text-muted-foreground" />
        <Label
          className="cursor-pointer text-xs"
          htmlFor="sensitive-log-toggle"
        >
          {t("sensitive.fullInformation", "完整信息")}
        </Label>
        <Switch
          aria-label={t("sensitive.fullInformation", "完整信息")}
          checked={value.enabled}
          id="sensitive-log-toggle"
          onCheckedChange={handleCheckedChange}
        />
      </div>

      <Dialog onOpenChange={setDialogOpen} open={dialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="mb-1 flex size-10 items-center justify-center rounded-md border bg-muted/50 text-muted-foreground">
              <ShieldCheck className="size-5" />
            </div>
            <DialogTitle>
              {t("sensitive.title", "查看完整日志信息")}
            </DialogTitle>
            <DialogDescription>
              {t(
                "sensitive.description",
                "完整字段仅显示 60 秒。本次访问将记录管理员、来源 IP、访问原因和记录数量。"
              )}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="sensitive-log-reason">
              {t("sensitive.reason", "访问原因")}
            </Label>
            <Select
              onValueChange={(next) => setReason(next as SensitiveLogReason)}
              value={reason}
            >
              <SelectTrigger className="w-full" id="sensitive-log-reason">
                <SelectValue
                  placeholder={t("sensitive.selectReason", "选择访问原因")}
                />
              </SelectTrigger>
              <SelectContent>
                {REVEAL_REASONS.map((item) => (
                  <SelectItem key={item} value={item}>
                    {t(
                      `sensitive.reasons.${item}`,
                      {
                        customer_support: "客户支持",
                        account_verification: "账户核验",
                        security_incident: "安全事件",
                        billing_dispute: "账单争议",
                        compliance: "合规审查",
                      }[item]
                    )}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <DialogFooter>
            <Button
              onClick={() => setDialogOpen(false)}
              type="button"
              variant="outline"
            >
              {t("cancel", "取消")}
            </Button>
            <Button disabled={!reason} onClick={reveal} type="button">
              <Eye />
              {t("sensitive.reveal", "显示完整信息")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
