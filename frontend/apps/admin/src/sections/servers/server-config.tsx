"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import { Card, CardContent } from "@workspace/ui/components/card";
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
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import {
  getNodeConfig,
  updateNodeConfig,
} from "@workspace/ui/services/admin/system";
import { unitConversion } from "@workspace/ui/utils/unit-conversions";
import { type ReactNode, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

const nodeConfigSchema = z.object({
  node_pull_interval: z.number().optional(),
  node_push_interval: z.number().optional(),
  traffic_report_threshold: z.number().optional(),
});
type NodeConfigFormData = z.infer<typeof nodeConfigSchema>;

export default function ServerConfig({
  trigger,
  open: controlledOpen,
  onOpenChange,
}: {
  trigger?: ReactNode | null;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
} = {}) {
  const { t } = useTranslation("servers");
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const open = controlledOpen ?? uncontrolledOpen;
  const setOpen = (nextOpen: boolean) => {
    onOpenChange?.(nextOpen);
    if (controlledOpen === undefined) setUncontrolledOpen(nextOpen);
  };
  const [saving, setSaving] = useState(false);

  const { data: cfgResp, refetch: refetchCfg } = useQuery({
    queryKey: ["getNodeConfig"],
    queryFn: async () => {
      const { data } = await getNodeConfig();
      return data.data as API.NodeConfig | undefined;
    },
    enabled: open,
  });

  const form = useForm<NodeConfigFormData>({
    resolver: zodResolver(nodeConfigSchema),
    defaultValues: {
      node_pull_interval: undefined,
      node_push_interval: undefined,
      traffic_report_threshold: undefined,
    },
  });

  useEffect(() => {
    if (cfgResp) {
      form.reset({
        node_pull_interval: cfgResp.node_pull_interval as number | undefined,
        node_push_interval: cfgResp.node_push_interval as number | undefined,
        traffic_report_threshold: cfgResp.traffic_report_threshold as
          | number
          | undefined,
      });
    }
  }, [cfgResp, form]);

  async function onSubmit(values: NodeConfigFormData) {
    setSaving(true);
    try {
      const currentConfig = {
        ...(cfgResp || {}),
        ip_strategy: undefined,
      } as Partial<API.NodeConfig>;
      await updateNodeConfig({
        ...currentConfig,
        ...values,
      } as API.NodeConfig);
      toast.success(t("server_config.saveSuccess", "Saved successfully"));
      await refetchCfg();
      setOpen(false);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      {trigger !== null ? (
        <SheetTrigger asChild>
          {trigger || (
            <Card className="rounded-lg border bg-card transition-colors hover:bg-muted/30">
              <CardContent className="p-4">
                <div className="flex cursor-pointer items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                      <Icon
                        className="h-5 w-5 text-primary"
                        icon="mdi:resistor-nodes"
                      />
                    </div>
                    <div className="flex-1">
                      <p className="font-medium">
                        {t("server_config.title", "Node configuration")}
                      </p>
                      <p className="truncate text-muted-foreground text-sm">
                        {t(
                          "server_config.description",
                          "Manage node synchronization intervals and reporting."
                        )}
                      </p>
                    </div>
                  </div>
                  <Icon className="size-6" icon="mdi:chevron-right" />
                </div>
              </CardContent>
            </Card>
          )}
        </SheetTrigger>
      ) : null}

      <SheetContent className="w-[720px] max-w-full md:max-w-3xl">
        <SheetHeader>
          <SheetTitle>
            {t("server_config.title", "Node configuration")}
          </SheetTitle>
          <SheetDescription>
            {t(
              "server_config.panel_description",
              "Configure synchronization, traffic reporting, and access log collection."
            )}
          </SheetDescription>
        </SheetHeader>

        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <form
              className="space-y-4 pt-4"
              id="server-config-form"
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <FormField
                control={form.control}
                name="node_pull_interval"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t(
                        "server_config.fields.node_pull_interval",
                        "Node pull interval"
                      )}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={0}
                        onValueChange={field.onChange}
                        placeholder={t(
                          "server_config.fields.interval_placeholder",
                          "Please enter"
                        )}
                        suffix="S"
                        type="number"
                        value={field.value as number | undefined}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "server_config.fields.node_pull_interval_desc",
                        "How often the node pulls configuration (seconds)."
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="node_push_interval"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t(
                        "server_config.fields.node_push_interval",
                        "Node push interval"
                      )}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={0}
                        onValueChange={field.onChange}
                        placeholder={t(
                          "server_config.fields.interval_placeholder",
                          "Please enter"
                        )}
                        step={0.1}
                        suffix="S"
                        type="number"
                        value={field.value as number | undefined}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "server_config.fields.node_push_interval_desc",
                        "How often the node pushes stats (seconds)."
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="traffic_report_threshold"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t(
                        "server_config.fields.traffic_report_threshold",
                        "Traffic Report Threshold"
                      )}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={0}
                        onValueChange={(value) => {
                          field.onChange(unitConversion("mbToBits", value));
                        }}
                        placeholder="1"
                        suffix="MB"
                        type="number"
                        value={unitConversion(
                          "bitsToMb",
                          field.value as number | undefined
                        )}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "server_config.fields.traffic_report_threshold_desc",
                        "Set the minimum threshold for traffic reporting."
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </form>
          </Form>
        </ScrollArea>

        <SheetFooter className="flex-row justify-end gap-2 pt-3">
          <Button
            disabled={saving}
            onClick={() => setOpen(false)}
            variant="outline"
          >
            {t("actions.cancel", "Cancel")}
          </Button>
          <Button disabled={saving} form="server-config-form" type="submit">
            <Icon
              className={saving ? "mr-2 animate-spin" : "hidden"}
              icon="mdi:loading"
            />
            {t("actions.save", "Save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
