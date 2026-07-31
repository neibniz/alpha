import { Button } from "@workspace/ui/components/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@workspace/ui/components/dialog";
import { Input } from "@workspace/ui/components/input";
import { Label } from "@workspace/ui/components/label";
import { batchResetUserSubscribeTraffic } from "@workspace/ui/services/admin/user";
import { LoaderCircle, RotateCcw, TriangleAlert } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import {
  canSubmitTrafficReset,
  RESET_ALL_TRAFFIC_CONFIRMATION,
} from "./traffic-reset-dialog-data";

interface TrafficResetDialogProps {
  scope: "all" | "users";
  userIds?: number[];
  trigger: ReactNode;
  onReset?: () => void;
}

export function TrafficResetDialog({
  scope,
  userIds = [],
  trigger,
  onReset,
}: TrafficResetDialogProps) {
  const { t } = useTranslation("user");
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const all = scope === "all";
  const canSubmit = canSubmitTrafficReset(scope, confirmation, userIds);

  const handleOpenChange = (nextOpen: boolean) => {
    if (submitting) return;
    setOpen(nextOpen);
    if (!nextOpen) setConfirmation("");
  };

  const submit = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      const { data } = await batchResetUserSubscribeTraffic({
        scope,
        user_ids: all ? undefined : userIds,
        confirmation: all ? confirmation.trim() : undefined,
      });
      toast.success(
        t(
          "batchTrafficResetSuccess",
          "Reset {{count}} subscriptions; {{reactivated}} reactivated",
          {
            count: data.data?.reset_count || 0,
            reactivated: data.data?.reactivated || 0,
          }
        )
      );
      handleOpenChange(false);
      onReset?.();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="mb-2 flex size-10 items-center justify-center rounded-md bg-destructive/10 text-destructive">
            <TriangleAlert className="size-5" />
          </div>
          <DialogTitle>
            {all
              ? t(
                  "resetAllTrafficRiskTitle",
                  "High-risk operation: reset all subscription traffic"
                )
              : t("resetSelectedTraffic", "Reset selected users' traffic")}
          </DialogTitle>
          <DialogDescription>
            {all
              ? t(
                  "resetAllTrafficDescription",
                  "This clears current counters for every subscription. Historical traffic is retained, and only non-expired traffic-exhausted subscriptions are reactivated."
                )
              : t(
                  "resetSelectedTrafficDescription",
                  "This clears current counters for every subscription owned by the selected users. Historical traffic is retained."
                )}
          </DialogDescription>
        </DialogHeader>
        {all ? (
          <div className="space-y-2 py-2">
            <Label htmlFor="reset-all-confirmation">
              {t("typeToConfirm", "Type {{text}} to confirm", {
                text: RESET_ALL_TRAFFIC_CONFIRMATION,
              })}
            </Label>
            <Input
              autoComplete="off"
              id="reset-all-confirmation"
              onChange={(event) => setConfirmation(event.target.value)}
              placeholder={RESET_ALL_TRAFFIC_CONFIRMATION}
              value={confirmation}
            />
          </div>
        ) : (
          <div className="rounded-md border bg-muted/30 px-3 py-2 text-sm">
            {t("selectedUserCount", "{{count}} users selected", {
              count: userIds.length,
            })}
          </div>
        )}
        <DialogFooter>
          <Button onClick={() => handleOpenChange(false)} variant="outline">
            {t("cancel", "Cancel")}
          </Button>
          <Button
            disabled={!canSubmit || submitting}
            onClick={submit}
            variant="destructive"
          >
            {submitting ? (
              <LoaderCircle className="size-4 animate-spin" />
            ) : (
              <RotateCcw className="size-4" />
            )}
            {t("resetTraffic", "Reset traffic")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
