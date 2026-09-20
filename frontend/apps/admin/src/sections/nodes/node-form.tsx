"use client";

import { useFormCloseGuard } from "@/components/use-form-close-guard";
import { AdminForm } from "@/components/admin-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@workspace/ui/components/form";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Combobox } from "@workspace/ui/composed/combobox";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import TagInput from "@workspace/ui/composed/tag-input";
import { queryServerXrayTemplateList } from "@workspace/ui/services/admin/server";
import { normalizeListResult } from "@workspace/ui/utils/pagination";
import type { TFunction } from "i18next";
import { LoaderCircle } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { useNode } from "@/stores/node";
import { useServer } from "@/stores/server";

type InboundOption = {
  alias: string;
  label: string;
  port: number;
};

type NodeInitialValues = Partial<NodeFormValues> & {
  protocol?: string;
  tags?: unknown;
};

function isJsonRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function readInboundPort(binding?: API.ServerXrayTemplate) {
  if (!binding) return 0;
  const variables = {
    ...(binding.template?.default_variables || {}),
    ...(binding.variables || {}),
  } as Record<string, unknown>;
  const config = binding.template?.config;
  const settings = isJsonRecord(config?.settings) ? config.settings : undefined;
  const port = variables.port ?? config?.port ?? settings?.port;
  const value = Number(port);
  return Number.isFinite(value) ? value : 0;
}

function normalizeInitialValues(values?: NodeInitialValues) {
  return {
    ...values,
    inbound_alias: values?.inbound_alias || values?.protocol || "",
    tags: Array.isArray(values?.tags) ? values.tags : [],
  };
}

function buildDefaultValues(values?: NodeInitialValues): NodeFormValues {
  const normalized = normalizeInitialValues(values);
  return {
    name: normalized.name || "",
    server_id: normalized.server_id || 0,
    inbound_alias: normalized.inbound_alias || "",
    address: normalized.address || "",
    port: normalized.port || 0,
    tags: Array.isArray(normalized.tags) ? normalized.tags : [],
  };
}

const buildSchema = (t: TFunction) =>
  z.object({
    name: z
      .string()
      .trim()
      .min(1, t("errors.nameRequired", "Please enter a name")),
    server_id: z
      .number({ message: t("errors.serverRequired", "Please select a server") })
      .int()
      .gt(0, t("errors.serverRequired", "Please select a server")),
    inbound_alias: z
      .string()
      .min(1, t("errors.inboundRequired", "请选择 inbound")),
    address: z
      .string()
      .trim()
      .min(1, t("errors.serverAddrRequired", "Please enter an entry address")),
    port: z
      .number({
        message: t("errors.portRange", "Port must be between 1 and 65535"),
      })
      .int()
      .min(1, t("errors.portRange", "Port must be between 1 and 65535"))
      .max(65_535, t("errors.portRange", "Port must be between 1 and 65535")),
    tags: z.array(z.string()),
  });

export type NodeFormValues = z.infer<ReturnType<typeof buildSchema>>;

export default function NodeForm(props: {
  trigger: ReactNode;
  title: string;
  loading?: boolean;
  initialValues?: NodeInitialValues;
  onSubmit: (values: NodeFormValues) => Promise<boolean> | boolean;
}) {
  const { trigger, title, loading, initialValues, onSubmit } = props;
  const { t } = useTranslation("nodes");
  const Scheme = useMemo(() => buildSchema(t), [t]);
  const [open, setOpen] = useState(false);

  const [autoFilledFields, setAutoFilledFields] = useState<Set<string>>(
    new Set()
  );

  const addAutoFilledField = (fieldName: string) => {
    setAutoFilledFields((prev) => new Set(prev).add(fieldName));
  };

  const removeAutoFilledField = (fieldName: string) => {
    setAutoFilledFields((prev) => {
      const newSet = new Set(prev);
      newSet.delete(fieldName);
      return newSet;
    });
  };

  const form = useForm<NodeFormValues>({
    resolver: zodResolver(Scheme),
    defaultValues: buildDefaultValues(initialValues),
  });

  const serverId = form.watch("server_id");

  const { servers, getServerById } = useServer();
  const { tags } = useNode();

  const existingTags: string[] = tags || [];
  const serverOptions = useMemo(
    () =>
      servers.map((server) => ({
        value: server.id,
        label: `${server.name} (${server.address || ""})`,
      })),
    [servers]
  );

  const { data: xrayBindingData } = useQuery({
    enabled: open && !!serverId,
    queryKey: ["node-form-server-xray-inbounds", serverId],
    queryFn: async () => {
      const { data } = await queryServerXrayTemplateList({
        server_id: Number(serverId),
        include_template: true,
      });
      return normalizeListResult(data.data);
    },
  });

  const inboundOptions = useMemo<InboundOption[]>(() => {
    const bindings: API.ServerXrayTemplate[] = [];
    for (const binding of xrayBindingData || []) {
      if (binding.enabled && binding.template?.type === "inbound") {
        bindings.push(binding);
      }
    }
    bindings.sort((a, b) => a.sort - b.sort);

    const options: InboundOption[] = [];
    for (const binding of bindings) {
      const alias = binding.alias || binding.template?.name || "";
      if (!alias) continue;
      const port = readInboundPort(binding);
      options.push({
        alias,
        port,
        label: `${alias || binding.template?.name || "inbound"}${
          port ? ` (${port})` : ""
        }`,
      });
    }
    return options;
  }, [xrayBindingData]);
  const inboundOptionByAlias = useMemo(
    () => new Map(inboundOptions.map((option) => [option.alias, option])),
    [inboundOptions]
  );
  const inboundComboboxOptions = useMemo(
    () =>
      inboundOptions.map((option) => ({
        value: option.alias,
        label: option.label,
      })),
    [inboundOptions]
  );

  useEffect(() => {
    if (initialValues && !form.formState.isDirty) {
      form.reset(buildDefaultValues(initialValues));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialValues]);

  function handleServerChange(nextId?: number | null) {
    const id = nextId || 0;
    form.setValue("server_id", id, { shouldDirty: true });

    if (!id) {
      setAutoFilledFields(new Set());
      return;
    }

    const selectedServer = getServerById(id);
    if (!selectedServer) return;

    const currentValues = form.getValues();
    const fieldsToFill: string[] = [];

    if (!currentValues.name || autoFilledFields.has("name")) {
      form.setValue("name", selectedServer.name as string, {
        shouldDirty: false,
      });
      fieldsToFill.push("name");
    }

    if (!currentValues.address || autoFilledFields.has("address")) {
      form.setValue("address", selectedServer.address as string, {
        shouldDirty: false,
      });
      fieldsToFill.push("address");
    }

    setAutoFilledFields(new Set(fieldsToFill));
  }

  useEffect(() => {
    if (!(serverId && inboundOptions.length)) return;
    const currentValues = form.getValues();
    const currentExists = inboundOptionByAlias.has(
      currentValues.inbound_alias || ""
    );
    if (currentExists && !autoFilledFields.has("inbound_alias")) return;

    const firstInbound = inboundOptions[0];
    if (!firstInbound) return;

    const fieldsToFill = new Set(autoFilledFields);
    if (!currentValues.inbound_alias || autoFilledFields.has("inbound_alias")) {
      form.setValue("inbound_alias", firstInbound.alias, {
        shouldDirty: false,
      });
      fieldsToFill.add("inbound_alias");
    }
    if (
      !currentValues.port ||
      currentValues.port === 0 ||
      autoFilledFields.has("port")
    ) {
      form.setValue("port", firstInbound.port || 0, { shouldDirty: false });
      fieldsToFill.add("port");
    }
    setAutoFilledFields(fieldsToFill);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverId, inboundOptions, inboundOptionByAlias]);

  const handleTextFieldChange = (
    fieldName: "name" | "address",
    value: string
  ) => {
    form.setValue(fieldName, value, { shouldDirty: true });
    removeAutoFilledField(fieldName);
  };

  function handleInboundChange(nextInbound?: string | null) {
    const inbound = nextInbound || "";
    form.setValue("inbound_alias", inbound, { shouldDirty: true });

    if (!(inbound && serverId)) {
      removeAutoFilledField("inbound_alias");
      return;
    }

    const currentValues = form.getValues();
    const isPortAutoFilled = autoFilledFields.has("port");

    removeAutoFilledField("inbound_alias");

    if (!currentValues.port || currentValues.port === 0 || isPortAutoFilled) {
      const protocolData = inboundOptionByAlias.get(inbound);

      if (protocolData) {
        const port = protocolData.port || 0;
        form.setValue("port", port, { shouldDirty: false });
        addAutoFilledField("port");
      }
    }
  }

  const closeGuard = useFormCloseGuard({
    open,
    dirty: form.formState.isDirty,
    busy: Boolean(loading || form.formState.isSubmitting),
    onOpenChange: (next) => {
      if (!next) form.reset();
      setOpen(next);
    },
  });

  async function handleSubmit(values: NodeFormValues) {
    const result = await onSubmit({
      ...values,
      tags: Array.isArray(values.tags) ? values.tags : [],
    });
    if (result) {
      setOpen(false);
      setAutoFilledFields(new Set());
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (nextOpen) {
      form.reset(buildDefaultValues(initialValues));
      setAutoFilledFields(new Set());
    }
    setOpen(nextOpen);
  }

  return (
    <Sheet
      onOpenChange={(next) =>
        next ? handleOpenChange(true) : closeGuard.requestOpenChange(false)
      }
      open={open}
    >
      {closeGuard.dialog}
      <SheetTrigger asChild>
        {typeof trigger === "string" ? <Button>{trigger}</Button> : trigger}
      </SheetTrigger>

      <SheetContent className="w-[560px] max-w-full gap-0 sm:max-w-[560px]">
        <SheetHeader className="border-b px-6 py-5 pr-12">
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription>
            {t(
              "node_form_description",
              "配置节点展示信息，并关联该服务器已绑定的 Xray inbound。"
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="min-h-0 flex-1 px-6">
          <Form {...form}>
            <AdminForm
              className="grid grid-cols-1"
              onSubmit={form.handleSubmit(handleSubmit)}
            >
              <section className="grid gap-4 border-b py-5">
                <p className="font-medium text-xs">
                  {t("form.bindingSection", "Server and Inbound")}
                </p>
                <FormField
                  control={form.control}
                  name="server_id"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("server", "Server")}</FormLabel>
                      <FormControl>
                        <Combobox<number, false>
                          onChange={(v) => handleServerChange(v)}
                          options={serverOptions}
                          placeholder={t("select_server", "Select server…")}
                          value={field.value || undefined}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="inbound_alias"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("inbound", "Inbound")}</FormLabel>
                      <FormControl>
                        <Combobox<string, false>
                          onChange={(v) => handleInboundChange(v || null)}
                          options={inboundComboboxOptions}
                          placeholder={t("select_inbound", "Select inbound…")}
                          value={field.value}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          "inbound_description",
                          "Loaded from the Xray template bindings on this server."
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </section>

              <section className="grid gap-4 border-b py-5">
                <p className="font-medium text-xs">
                  {t("form.endpointSection", "Node and access endpoint")}
                </p>
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("name", "Name")}</FormLabel>
                      <FormControl>
                        <EnhancedInput
                          {...field}
                          onValueChange={(v) =>
                            handleTextFieldChange("name", v as string)
                          }
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_140px]">
                  <FormField
                    control={form.control}
                    name="address"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t("address", "Address")}</FormLabel>
                        <FormControl>
                          <EnhancedInput
                            {...field}
                            onValueChange={(v) =>
                              handleTextFieldChange("address", v as string)
                            }
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="port"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t("port", "Port")}</FormLabel>
                        <FormControl>
                          <EnhancedInput
                            {...field}
                            max={65_535}
                            min={1}
                            onValueChange={(v) => {
                              form.setValue("port", Number(v), {
                                shouldDirty: true,
                              });
                              removeAutoFilledField("port");
                            }}
                            placeholder="1-65535"
                            step={1}
                            type="number"
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </section>

              <section className="grid gap-4 py-5">
                <p className="font-medium text-xs">
                  {t("form.deliverySection", "Delivery tags")}
                </p>
                <FormField
                  control={form.control}
                  name="tags"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("tags", "Tags")}</FormLabel>
                      <FormControl>
                        <TagInput
                          onChange={(v) =>
                            form.setValue(field.name, v || [], {
                              shouldDirty: true,
                            })
                          }
                          options={existingTags}
                          placeholder={t(
                            "tags_placeholder",
                            "Use Enter or comma (,) to add multiple tags"
                          )}
                          value={field.value || []}
                        />
                      </FormControl>
                      <FormDescription>
                        {t(
                          "tags_description",
                          "Used by plans and delivery policies."
                        )}
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </section>
            </AdminForm>
          </Form>
        </ScrollArea>

        <SheetFooter className="flex-row justify-end gap-2 border-t px-6 py-4">
          <Button
            disabled={loading}
            onClick={() => closeGuard.requestOpenChange(false)}
            variant="outline"
          >
            {t("cancel", "Cancel")}
          </Button>
          <Button
            disabled={loading}
            onClick={form.handleSubmit(handleSubmit, (errors) => {
              const key = Object.keys(errors)[0] as keyof typeof errors;
              if (key) toast.error(String(errors[key]?.message));
              return false;
            })}
          >
            {loading ? <LoaderCircle className="size-4 animate-spin" /> : null}
            {t("confirm", "Confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
