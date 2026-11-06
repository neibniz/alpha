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
  SelectGroup,
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
import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { PrivacyRevealDuration } from "@/components/privacy-reveal-duration";
import {
  DEFAULT_REVEAL_DURATION_MINUTES,
  type RevealDurationMinutes,
} from "@/lib/privacy-reveal-policy";
import { formatDate } from "@/utils/common";

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
  const reasonId = useId();
  const requestGeneration = useRef(0);
  const [duration, setDuration] = useState<RevealDurationMinutes>(
    DEFAULT_REVEAL_DURATION_MINUTES
  );
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<RevealReason>();
  const [loading, setLoading] = useState(false);
  const [privateIdentity, setPrivateIdentity] =
    useState<API.RevealUserPrivateIdentityResponse>();

  useEffect(() => {
    setPrivateIdentity(undefined);
    setLoading(false);
    setReason(undefined);
    return () => {
      requestGeneration.current += 1;
    };
  }, [userId]);

  useEffect(() => {
    if (!privateIdentity) return;
    const remaining = Math.max(0, privateIdentity.expires_at - Date.now());
    const timer = window.setTimeout(
      () => setPrivateIdentity(undefined),
      remaining
    );
    const checkExpiry = () => {
      if (Date.now() >= privateIdentity.expires_at)
        setPrivateIdentity(undefined);
    };
    window.addEventListener("focus", checkExpiry);
    document.addEventListener("visibilitychange", checkExpiry);
    return () => {
      window.clearTimeout(timer);
      window.removeEventListener("focus", checkExpiry);
      document.removeEventListener("visibilitychange", checkExpiry);
    };
  }, [privateIdentity]);

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      requestGeneration.current += 1;
      setLoading(false);
      setPrivateIdentity(undefined);
      setReason(undefined);
    }
  };

  const reveal = async () => {
    if (!reason) return;
    const generation = ++requestGeneration.current;
    setLoading(true);
    try {
      const response = await revealUserPrivateIdentity({
        user_id: userId,
        reason,
        duration_minutes: duration,
      });
      if (!response.data.data)
        throw new Error("Private identity is unavailable");
      if (
        generation === requestGeneration.current &&
        response.data.data.expires_at > Date.now()
      ) {
        setPrivateIdentity(response.data.data);
      }
    } finally {
      if (generation === requestGeneration.current) setLoading(false);
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
              "This action is audited. Complete identifiers are shown only in this dialog and cleared when the selected time expires or the dialog closes."
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
                  <p className="min-w-0 flex-1 break-all font-medium text-sm">
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
            ) : (
              <p className="text-muted-foreground text-sm">
                {t("noEmailBound", "未绑定邮箱")}
              </p>
            )}

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
          <div className="flex flex-col gap-3">
            <label className="font-medium text-sm" htmlFor={reasonId}>
              {t("revealReasonLabel", "Access reason")}
            </label>
            <Select
              onValueChange={(value) => setReason(value as RevealReason)}
              value={reason || ""}
            >
              <SelectTrigger className="w-full" id={reasonId}>
                <SelectValue
                  placeholder={t("selectRevealReason", "Select a reason")}
                />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
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
                </SelectGroup>
              </SelectContent>
            </Select>
            <PrivacyRevealDuration value={duration} onChange={setDuration} />
          </div>
        )}

        <DialogFooter>
          {privateIdentity ? (
            <div className="flex w-full flex-wrap items-center justify-between gap-3">
              <p className="text-muted-foreground text-xs">
                {t("privateIdentityExpiresAt", {
                  defaultValue: "有效至 {{time}}",
                  time: formatDate(privateIdentity.expires_at),
                })}
              </p>
              <Button onClick={() => handleOpenChange(false)} type="button">
                {t("hidePrivateData", "Hide private data")}
              </Button>
            </div>
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
