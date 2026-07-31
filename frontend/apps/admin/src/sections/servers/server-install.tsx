"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
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
import { Field, FieldGroup, FieldLabel } from "@workspace/ui/components/field";
import { Textarea } from "@workspace/ui/components/textarea";
import { queryServerNodeToken } from "@workspace/ui/services/admin/server";
import { AlertTriangle, Copy, Server as ServerIcon } from "lucide-react";
import {
  type ChangeEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

type Props = {
  server: API.ServerListItem;
  trigger?: ReactNode;
};

export default function ServerInstall({ server, trigger }: Props) {
  const { t } = useTranslation("servers");
  const [open, setOpen] = useState(false);
  const [domain, setDomain] = useState("");

  const { data: nodeToken, isFetching: tokenLoading } = useQuery({
    queryKey: ["serverNodeToken", server.id],
    queryFn: async () => {
      const { data } = await queryServerNodeToken({ id: server.id });
      return data.data?.token ?? "";
    },
    enabled: open,
  });

  useEffect(() => {
    if (open) {
      const host = localStorage.getItem("API_HOST") ?? window.location.origin;
      setDomain(host);
    }
  }, [open]);

  const installCommand = useMemo(
    () =>
      [
        "docker run -d --name alpha-xray-agent --restart unless-stopped --network host \\",
        "  -v /var/lib/alpha/xray-agent:/var/lib/alpha/xray-agent \\",
        `  -e ALPHA_SERVER_URL=${domain} \\`,
        `  -e ALPHA_SERVER_ID=${server.id} \\`,
        `  -e ALPHA_NODE_TOKEN=${nodeToken ?? ""} \\`,
        "  alpha-xray-agent:lan",
      ].join("\n"),
    [domain, server.id, nodeToken]
  );

  async function handleCopy() {
    if (!nodeToken) {
      toast.error(t("tokenUnavailable", "Node token is not available"));
      return;
    }
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(installCommand);
      } else {
        // fallback for environments without clipboard API
        const el = document.createElement("textarea");
        el.value = installCommand;
        document.body.appendChild(el);
        el.select();
        document.execCommand("copy");
        document.body.removeChild(el);
      }
      toast.success(t("copied", "Copied"));
      setOpen(false);
    } catch {
      toast.error(t("copyFailed", "Copy failed"));
    }
  }

  const onDomainChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setDomain(e.target.value);
    localStorage.setItem("API_HOST", e.target.value);
  }, []);
  return (
    <Dialog onOpenChange={setOpen} open={open}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="secondary">{t("connect", "Connect")}</Button>
        )}
      </DialogTrigger>

      <DialogContent className="w-[680px] max-w-[calc(100%-2rem)] gap-5 md:max-w-2xl">
        <DialogHeader className="gap-0">
          <div className="flex items-start gap-3">
            <div className="flex size-9 shrink-0 items-center justify-center rounded-md border bg-muted/30 text-muted-foreground">
              <ServerIcon className="size-4" />
            </div>
            <div className="min-w-0">
              <DialogTitle>
                {t("oneClickInstall", "Install xray-agent")}
              </DialogTitle>
              <DialogDescription className="mt-1 truncate">
                {server.name} · {server.address}
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <FieldGroup className="gap-5">
          <Field>
            <FieldLabel htmlFor={`server-${server.id}-api-host`}>
              {t("apiHost", "API Host")}
            </FieldLabel>
            <Input
              id={`server-${server.id}-api-host`}
              onChange={onDomainChange}
              placeholder={t("apiHostPlaceholder", "http(s)://example.com")}
              value={domain}
            />
          </Field>

          <Field>
            <FieldLabel htmlFor={`server-${server.id}-install-command`}>
              {t("installCommand", "Install command")}
            </FieldLabel>
            <Textarea
              aria-label={t("installCommand", "Install command")}
              className="h-28 resize-none bg-muted/25 font-mono text-xs leading-5"
              id={`server-${server.id}-install-command`}
              readOnly
              spellCheck={false}
              value={installCommand}
            />
          </Field>

          <Alert>
            <AlertTriangle />
            <AlertTitle>
              {t("credentialWarning", "Credential warning")}
            </AlertTitle>
            <AlertDescription>
              {t(
                "installCredentialWarning",
                "This command contains the node credential. Treat it as a secret and rotate it if exposed."
              )}
            </AlertDescription>
          </Alert>
        </FieldGroup>

        <DialogFooter className="flex-row justify-end gap-2 pt-3">
          <Button onClick={() => setOpen(false)} variant="outline">
            {t("close", "Close")}
          </Button>
          <Button disabled={tokenLoading || !nodeToken} onClick={handleCopy}>
            <Copy className="size-4" />
            {t("copyAndClose", "Copy and Close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
