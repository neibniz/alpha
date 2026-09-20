import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@workspace/ui/components/alert-dialog";
import { useTranslation } from "react-i18next";

export function DiscardChangesDialog({
  open,
  onOpenChange,
  onDiscard,
  disabled = false,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDiscard: () => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation("components");

  return (
    <AlertDialog onOpenChange={onOpenChange} open={open}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t("form.discardTitle", "Discard unsaved changes?")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "form.discardDescription",
              "Your edits have not been saved. Keep editing or discard them and close."
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>
            {t("form.keepEditing", "Keep editing")}
          </AlertDialogCancel>
          <AlertDialogAction disabled={disabled} onClick={onDiscard}>
            {t("form.discard", "Discard changes")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
