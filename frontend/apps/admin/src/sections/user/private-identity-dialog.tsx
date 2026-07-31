import { Badge } from "@workspace/ui/components/badge";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { revealUserPrivateIdentity } from "@workspace/ui/services/admin/user";
import {
  Copy,
  Eye,
  Github,
  KeyRound,
  LoaderCircle,
  Mail,
  ShieldCheck,
} from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

type RevealReason = API.RevealUserPrivateIdentityRequest["reason"];

const REVEAL_REASONS: RevealReason[] = [
  "customer_support",
  "account_verification",
  "security_incident",
  "billing_dispute",
  "compliance",
];

function authMethodIcon(authType: string) {
  if (authType === "email") return Mail;
  if (authType === "github") return Github;
  return KeyRound;
}

export function PrivateIdentityDialog({
  trigger,
  userId,
}: {
  trigger: ReactNode;
  userId: number;
}) {
  const { t } = useTranslation("user");
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<RevealReason>();
  const [loading, setLoading] = useState(false);
  const [privateIdentity, setPrivateIdentity] =
    useState<API.RevealUserPrivateIdentityResponse>();

  useEffect(() => {
    if (!privateIdentity) return;
    const remaining = Math.max(0, privateIdentity.expires_at - Date.now());
    const timer = window.setTimeout(
      () => setPrivateIdentity(undefined),
      remaining
    );
    return () => window.clearTimeout(timer);
  }, [privateIdentity]);

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setPrivateIdentity(undefined);
      setReason(undefined);
    }
  };

  const reveal = async () => {
    if (!reason) return;
    setLoading(true);
    try {
      const response = await revealUserPrivateIdentity({
        user_id: userId,
        reason,
      });
      if (!response.data.data)
        throw new Error("Private identity is unavailable");
      setPrivateIdentity(response.data.data);
    } finally {
      setLoading(false);
    }
  };

  const copy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(t("copySuccess", "Copied successfully"));
    } catch {
      toast.error(t("copyFailed", "Copy failed"));
    }
  };

  return (
    <Dialog onOpenChange={handleOpenChange} open={open}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <div className="mb-1 flex size-10 items-center justify-center rounded-md border bg-muted/50 text-muted-foreground">
            <ShieldCheck className="size-5" />
          </div>
          <DialogTitle>
            {t("privateIdentityTitle", "View private identity")} · #{userId}
          </DialogTitle>
          <DialogDescription>
            {t(
              "privateIdentityDescription",
              "This action is audited. Complete identifiers are shown only in this dialog and cleared after 60 seconds."
            )}
          </DialogDescription>
        </DialogHeader>

        {privateIdentity ? (
          <div className="space-y-3">
            {privateIdentity.email ? (
              <div className="rounded-md border bg-muted/30 px-4 py-3">
                <p className="text-muted-foreground text-xs">
                  {t("email", "Email")}
                </p>
                <div className="mt-1 flex items-center gap-2">
                  <p className="min-w-0 flex-1 truncate font-medium text-sm">
                    {privateIdentity.email}
                  </p>
                  <Button
                    aria-label={t("copyEmail", "Copy email")}
                    onClick={() => copy(privateIdentity.email)}
                    size="icon-sm"
                    type="button"
                    variant="ghost"
                  >
                    <Copy />
                  </Button>
                </div>
              </div>
            ) : null}

            <div className="divide-y overflow-hidden rounded-md border">
              {privateIdentity.auth_methods.map((method) => {
                const Icon = authMethodIcon(method.auth_type);
                return (
                  <div
                    className="flex items-center gap-3 px-4 py-3"
                    key={`${method.auth_type}:${method.auth_identifier}`}
                  >
                    <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                      <Icon className="size-4" />
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="font-medium text-xs uppercase">
                          {method.auth_type}
                        </p>
                        <Badge
                          variant={method.verified ? "secondary" : "outline"}
                        >
                          {method.verified
                            ? t("verified", "Verified")
                            : t("unverified", "Unverified")}
                        </Badge>
                      </div>
                      <p className="mt-1 break-all font-mono text-sm">
                        {method.auth_identifier}
                      </p>
                    </div>
                    <Button
                      aria-label={t("copyIdentity", "Copy identifier")}
                      onClick={() => copy(method.auth_identifier)}
                      size="icon-sm"
                      type="button"
                      variant="ghost"
                    >
                      <Copy />
                    </Button>
                  </div>
                );
              })}
              {privateIdentity.auth_methods.length === 0 ? (
                <p className="px-4 py-8 text-center text-muted-foreground text-sm">
                  {t("noAuthMethods", "No sign-in identities found")}
                </p>
              ) : null}
            </div>
          </div>
        ) : (
          <div className="space-y-2">
            <label className="font-medium text-sm" htmlFor="reveal-reason">
              {t("revealReasonLabel", "Access reason")}
            </label>
            <Select
              onValueChange={(value) => setReason(value as RevealReason)}
              value={reason}
            >
              <SelectTrigger className="w-full" id="reveal-reason">
                <SelectValue
                  placeholder={t("selectRevealReason", "Select a reason")}
                />
              </SelectTrigger>
              <SelectContent>
                {REVEAL_REASONS.map((value) => (
                  <SelectItem key={value} value={value}>
                    {t(
                      `revealReasons.${value}`,
                      {
                        customer_support: "Customer support",
                        account_verification: "Account verification",
                        security_incident: "Security incident",
                        billing_dispute: "Billing dispute",
                        compliance: "Compliance review",
                      }[value]
                    )}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <DialogFooter>
          {privateIdentity ? (
            <Button onClick={() => handleOpenChange(false)} type="button">
              {t("hidePrivateData", "Hide private data")}
            </Button>
          ) : (
            <Button
              disabled={!reason || loading}
              onClick={reveal}
              type="button"
            >
              {loading ? <LoaderCircle className="animate-spin" /> : <Eye />}
              {t("revealPrivateData", "Reveal private data")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
