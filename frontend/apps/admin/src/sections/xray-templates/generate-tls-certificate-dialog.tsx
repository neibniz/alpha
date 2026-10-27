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
  Field,
  FieldGroup,
  FieldLabel,
  FieldDescription,
} from "@workspace/ui/components/field";
import { Input } from "@workspace/ui/components/input";
import { Textarea } from "@workspace/ui/components/textarea";
import { LoaderCircle, Sparkles } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import {
  generateTLSCertificate,
  type GeneratedTLSCertificate,
} from "./generate-tls-certificate";

export function GenerateTLSCertificateDialog({
  serverName,
  onGenerated,
}: {
  serverName?: string;
  onGenerated: (value: GeneratedTLSCertificate) => void;
}) {
  const [open, setOpen] = useState(false);
  const [names, setNames] = useState("");
  const [days, setDays] = useState("365");
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState("");
  const active = useRef(true);
  const inFlight = useRef(false);
  const id = useId();
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  async function generate() {
    if (inFlight.current) return;
    inFlight.current = true;
    setGenerating(true);
    setError("");
    try {
      const result = await generateTLSCertificate(names, Number(days));
      if (!active.current) return;
      onGenerated(result);
      setOpen(false);
    } catch (cause) {
      if (active.current)
        setError(
          cause instanceof Error ? cause.message : "证书生成失败，请重试。"
        );
    } finally {
      inFlight.current = false;
      if (active.current) setGenerating(false);
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (inFlight.current) return;
        if (next) {
          setNames(serverName || "localhost");
          setDays("365");
          setError("");
        }
        setOpen(next);
      }}
    >
      <DialogTrigger asChild>
        <Button type="button" size="sm" variant="outline">
          <Sparkles />
          生成自签名证书
        </Button>
      </DialogTrigger>
      <DialogContent aria-busy={generating}>
        <DialogHeader>
          <DialogTitle>生成自签名证书</DialogTitle>
          <DialogDescription>
            生成证书、私钥与 SHA-256 指纹。客户端连接时需固定此指纹。
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor={`${id}-names`}>域名或 IP</FieldLabel>
            <Textarea
              id={`${id}-names`}
              disabled={generating}
              value={names}
              onChange={(event) => setNames(event.target.value)}
              placeholder="example.com"
            />
            <FieldDescription>
              写入证书 SAN，多个值用逗号或换行分隔。
            </FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor={`${id}-days`}>有效期（天）</FieldLabel>
            <Input
              id={`${id}-days`}
              disabled={generating}
              type="number"
              min={1}
              max={3650}
              step={1}
              value={days}
              onChange={(event) => setDays(event.target.value)}
            />
          </Field>
        </FieldGroup>
        {error ? (
          <p role="alert" className="break-words text-destructive text-sm">
            {error}
          </p>
        ) : null}
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={generating}
            onClick={() => setOpen(false)}
          >
            取消
          </Button>
          <Button type="button" disabled={generating} onClick={generate}>
            {generating ? (
              <LoaderCircle className="animate-spin" />
            ) : (
              <Sparkles />
            )}
            生成并填入
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
