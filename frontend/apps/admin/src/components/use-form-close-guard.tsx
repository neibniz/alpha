import { useEffect, useState } from "react";
import { DiscardChangesDialog } from "./discard-changes-dialog";

export function useFormCloseGuard({
  open,
  dirty,
  busy,
  onOpenChange,
}: {
  open: boolean;
  dirty: boolean;
  busy: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  useEffect(() => {
    if (!(open && dirty)) return;
    const preventUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", preventUnload);
    return () => window.removeEventListener("beforeunload", preventUnload);
  }, [open, dirty]);

  function requestOpenChange(next: boolean) {
    if (busy) return;
    if (!next && dirty) {
      setConfirmOpen(true);
      return;
    }
    onOpenChange(next);
  }

  return {
    requestOpenChange,
    dialog: (
      <DiscardChangesDialog
        open={open && confirmOpen}
        onOpenChange={setConfirmOpen}
        disabled={busy}
        onDiscard={() => {
          setConfirmOpen(false);
          onOpenChange(false);
        }}
      />
    ),
  };
}
