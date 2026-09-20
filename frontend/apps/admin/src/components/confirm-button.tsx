import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@workspace/ui/components/alert-dialog";
import { Alert, AlertDescription } from "@workspace/ui/components/alert";
import { CircleAlert, LoaderCircle } from "lucide-react";
import { type ReactNode, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

export function ConfirmButton({
  trigger,
  title,
  description,
  onConfirm,
  cancelText,
  confirmText,
}: {
  trigger: ReactNode;
  title: string;
  description: string;
  onConfirm: () => void | Promise<void>;
  cancelText?: string;
  confirmText?: string;
}) {
  const { t } = useTranslation("components");
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);
  const inFlight = useRef(false);

  async function confirm() {
    if (inFlight.current) return;
    inFlight.current = true;
    setPending(true);
    setFailed(false);
    try {
      await onConfirm();
      setOpen(false);
    } catch {
      setFailed(true);
    } finally {
      inFlight.current = false;
      setPending(false);
    }
  }

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (inFlight.current) return;
        setOpen(next);
        setFailed(false);
      }}
    >
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent aria-busy={pending}>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        {failed && (
          <Alert variant="destructive">
            <CircleAlert />
            <AlertDescription>
              {t(
                "action.failed",
                "The operation failed. Check the current state before trying again."
              )}
            </AlertDescription>
          </Alert>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>
            {cancelText ?? t("common.cancel", "Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            onClick={(event) => {
              event.preventDefault();
              confirm();
            }}
          >
            {pending && <LoaderCircle className="size-4 animate-spin" />}
            {pending
              ? t("action.processing", "Processing")
              : (confirmText ?? t("common.confirm", "Confirm"))}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
