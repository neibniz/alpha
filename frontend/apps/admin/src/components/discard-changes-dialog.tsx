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
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDiscard: () => void;
}) {
  const { t } = useTranslation("subscribe");

  return (
    <AlertDialog onOpenChange={onOpenChange} open={open}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t("discard.title", "Discard unsaved changes?")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "discard.description",
              "Your changes have not been saved. This action cannot be undone."
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>
            {t("discard.continueEditing", "Continue editing")}
          </AlertDialogCancel>
          <AlertDialogAction onClick={onDiscard}>
            {t("discard.confirm", "Discard changes")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
