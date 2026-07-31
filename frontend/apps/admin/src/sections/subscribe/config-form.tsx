"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@workspace/ui/components/item";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Switch } from "@workspace/ui/components/switch";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import { Textarea } from "@workspace/ui/components/textarea";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import {
  getSubscribeConfig,
  updateSubscribeConfig,
} from "@workspace/ui/services/admin/system";
import { copyTextToClipboard } from "@workspace/ui/utils/index";
import {
  AlertTriangle,
  Check,
  ChevronRight,
  Copy,
  Globe2,
  Layers3,
  Link2,
  ListChecks,
  RefreshCcw,
  Route,
  ServerCog,
  Settings2,
  ShieldCheck,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { type FieldErrors, useForm, useWatch } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { DiscardChangesDialog } from "../../components/discard-changes-dialog";
import {
  DEFAULT_SUBSCRIBE_CONFIG,
  isValidDomainLine,
  isValidSubscribePath,
  isValidUserAgentLine,
  lineCount,
  normalizeLines,
  normalizeSubscribeConfig,
  normalizeSubscribePath,
  SUBSCRIBE_CONFIG_QUERY_KEY,
  subscriptionEndpointPreview,
  type SubscribeConfigTab,
  subscriptionConfigTabForFields,
} from "./subscription-config-data";

type SubscribeTranslator = (key: string, defaultValue: string) => string;

const createSubscribeConfigSchema = (t: SubscribeTranslator) =>
  z.object({
    single_model: z.boolean(),
    pan_domain: z.boolean(),
    subscribe_path: z
      .string()
      .trim()
      .refine(
        isValidSubscribePath,
        t(
          "config.validation.path",
          "Use a static path such as /v1/subscribe/config"
        )
      ),
    subscribe_domain: z
      .string()
      .max(
        8192,
        t("config.validation.domainListTooLong", "Domain list is too long")
      )
      .refine(
        (value) => {
          const normalized = normalizeLines(value);
          return !normalized || normalized.split("\n").every(isValidDomainLine);
        },
        t(
          "config.validation.domain",
          "Enter concrete hostnames only, one per line, without wildcards, a protocol, port, or path"
        )
      ),
    user_agent_limit: z.boolean(),
    user_agent_list: z
      .string()
      .max(
        16_384,
        t(
          "config.validation.userAgentListTooLong",
          "User-Agent list is too long"
        )
      )
      .refine(
        (value) => {
          const normalized = normalizeLines(value);
          return (
            !normalized || normalized.split("\n").every(isValidUserAgentLine)
          );
        },
        t(
          "config.validation.userAgent",
          "Each User-Agent entry must contain 1 to 255 printable characters"
        )
      ),
  });

type SubscribeConfigFormData = z.infer<
  ReturnType<typeof createSubscribeConfigSchema>
>;
function ConfigSummary({
  icon: SummaryIcon,
  label,
  value,
  detail,
  onClick,
}: {
  icon: typeof Layers3;
  label: string;
  value: string;
  detail: string;
  onClick: () => void;
}) {
  return (
    <Item
      asChild
      className="subscribe-config-summary-item rounded-none border-0"
      size="sm"
    >
      <button onClick={onClick} type="button">
        <ItemMedia variant="icon">
          <SummaryIcon aria-hidden="true" />
        </ItemMedia>
        <ItemContent>
          <ItemDescription>{label}</ItemDescription>
          <ItemTitle className="subscribe-config-summary-value">
            {value}
          </ItemTitle>
          <ItemDescription>{detail}</ItemDescription>
        </ItemContent>
        <ItemActions>
          <ChevronRight
            aria-hidden="true"
            className="subscribe-config-summary-chevron"
          />
        </ItemActions>
      </button>
    </Item>
  );
}

function SectionHeader({
  icon: SectionIcon,
  title,
  description,
}: {
  icon: typeof Layers3;
  title: string;
  description: string;
}) {
  return (
    <header className="subscribe-settings-section-header">
      <SectionIcon aria-hidden="true" className="size-4 text-primary" />
      <div>
        <h3 className="font-semibold text-sm">{title}</h3>
        <p className="text-muted-foreground text-xs leading-5">{description}</p>
      </div>
    </header>
  );
}

export default function ConfigForm() {
  const { t } = useTranslation("subscribe");
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [discardOpen, setDiscardOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [activeTab, setActiveTab] = useState<SubscribeConfigTab>("policy");
  const schema = useMemo(() => createSubscribeConfigSchema(t), [t]);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: SUBSCRIBE_CONFIG_QUERY_KEY,
    queryFn: async () => {
      const { data: response } = await getSubscribeConfig();
      return { ...DEFAULT_SUBSCRIBE_CONFIG, ...response.data };
    },
  });

  const form = useForm<SubscribeConfigFormData>({
    resolver: zodResolver(schema),
    defaultValues: DEFAULT_SUBSCRIBE_CONFIG,
    mode: "onChange",
  });
  const userAgentLimit = useWatch({
    control: form.control,
    name: "user_agent_limit",
  });
  const draftValues = useWatch({ control: form.control });
  const draftConfig = useMemo(
    () =>
      ({
        ...DEFAULT_SUBSCRIBE_CONFIG,
        ...draftValues,
      }) as API.SubscribeConfig,
    [draftValues]
  );
  const draftDomains = lineCount(draftConfig.subscribe_domain);
  const invalidDraftDomains = normalizeLines(draftConfig.subscribe_domain)
    .split("\n")
    .filter(
      (item) => item && (item.includes("*") || !isValidDomainLine(item))
    ).length;
  const draftAgents = lineCount(draftConfig.user_agent_list);
  const endpointPreview = subscriptionEndpointPreview(draftConfig);

  const { isDirty, isValid } = form.formState;

  useEffect(() => {
    if (!data || (open && isDirty)) return;
    form.reset(data);
    if (open) form.trigger().catch(() => undefined);
  }, [data, form, isDirty, open]);

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen && saving) return;
    if (!nextOpen && form.formState.isDirty) {
      setDiscardOpen(true);
      return;
    }
    if (!nextOpen && data) form.reset(data);
    if (nextOpen) setActiveTab("policy");
    setOpen(nextOpen);
  }

  function discardChanges() {
    form.reset(data || DEFAULT_SUBSCRIBE_CONFIG);
    setDiscardOpen(false);
    setOpen(false);
  }

  function openTab(tab: SubscribeConfigTab) {
    setActiveTab(tab);
    setOpen(true);
  }

  async function copyEndpoint(value: string) {
    const didCopy = await copyTextToClipboard(value);
    if (!didCopy) {
      toast.error(t("config.copyFailed", "Unable to copy URL preview"));
      return;
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  function handleInvalid(errors: FieldErrors<SubscribeConfigFormData>) {
    setActiveTab(subscriptionConfigTabForFields(Object.keys(errors)));
  }

  async function onSubmit(values: SubscribeConfigFormData) {
    setSaving(true);
    try {
      const normalized = normalizeSubscribeConfig(values);
      const pathChanged = normalized.subscribe_path !== data?.subscribe_path;
      await updateSubscribeConfig(normalized);
      queryClient.setQueryData(SUBSCRIBE_CONFIG_QUERY_KEY, normalized);
      form.reset(normalized);
      setOpen(false);
      if (pathChanged) {
        toast.success(
          t(
            "config.pathUpdateSuccess",
            "Settings saved. The service is applying the new endpoint"
          )
        );
      } else {
        toast.success(
          t("config.updateSuccess", "Settings updated successfully")
        );
        queryClient
          .invalidateQueries({ queryKey: SUBSCRIBE_CONFIG_QUERY_KEY })
          .catch(() => undefined);
      }
    } catch (_error) {
      toast.error(t("config.updateError", "Update failed"));
    } finally {
      setSaving(false);
    }
  }

  const current = data || DEFAULT_SUBSCRIBE_CONFIG;
  const domains = lineCount(current.subscribe_domain);
  const allowedAgents = lineCount(current.user_agent_list);
  const currentEndpointPreview = subscriptionEndpointPreview(current);

  return (
    <>
      <Sheet onOpenChange={handleOpenChange} open={open}>
        <section className="subscribe-config-panel">
          <header className="subscribe-config-panel-header">
            <div>
              <h2 className="font-semibold text-base">
                {t("config.overviewTitle", "Subscription policy")}
              </h2>
              <p className="mt-1 text-muted-foreground text-sm">
                {t(
                  "config.overviewDescription",
                  "Review endpoint and access-control settings before editing"
                )}
              </p>
            </div>
            <Button
              disabled={isLoading || (isError && !data)}
              onClick={() => openTab("policy")}
              size="sm"
            >
              <Settings2 aria-hidden="true" className="size-4" />
              {t("config.edit", "Edit settings")}
            </Button>
          </header>

          {isLoading ? (
            <div className="subscribe-config-loading grid gap-3 md:grid-cols-3">
              {[0, 1, 2].map((item) => (
                <Skeleton className="h-20 rounded-md" key={item} />
              ))}
            </div>
          ) : isError && !data ? (
            <div className="subscribe-config-error">
              <span>
                {t("config.loadError", "Unable to load subscription settings")}
              </span>
              <Button onClick={() => refetch()} size="sm" variant="outline">
                <RefreshCcw aria-hidden="true" className="size-4" />
                {t("actions.retry", "Retry")}
              </Button>
            </div>
          ) : (
            <div className="subscribe-config-summary-grid">
              <ConfigSummary
                detail={
                  current.pan_domain
                    ? t("config.wildcardOn", "Wildcard resolution enabled")
                    : t("config.wildcardOff", "Standard domain resolution")
                }
                icon={Layers3}
                label={t("config.mode", "Subscription mode")}
                value={
                  current.single_model
                    ? t("config.singleMode", "Single subscription")
                    : t("config.multiMode", "Multiple subscriptions")
                }
                onClick={() => openTab("policy")}
              />
              <ConfigSummary
                detail={t("config.domainCount", "{{count}} custom domains", {
                  count: domains,
                })}
                icon={Route}
                label={t("config.endpoint", "Subscription endpoint")}
                value={
                  current.subscribe_path ||
                  t("config.systemDefault", "System default")
                }
                onClick={() => openTab("endpoint")}
              />
              <ConfigSummary
                detail={t(
                  "config.agentCount",
                  "{{count}} manual allow-list entries",
                  {
                    count: allowedAgents,
                  }
                )}
                icon={ShieldCheck}
                label="User-Agent"
                value={
                  current.user_agent_limit
                    ? t("config.restricted", "Restricted")
                    : t("config.unrestricted", "Unrestricted")
                }
                onClick={() => openTab("access")}
              />
            </div>
          )}
          {!isLoading && (!isError || data) ? (
            <div className="subscribe-config-public-url">
              <div className="min-w-0">
                <span>
                  {t("config.publicUrlPreview", "Public URL preview")}
                </span>
                <code>{currentEndpointPreview}</code>
              </div>
              <Button
                aria-label={t(
                  "config.copyPublicUrl",
                  "Copy public URL preview"
                )}
                onClick={() => copyEndpoint(currentEndpointPreview)}
                size="icon"
                type="button"
                variant="ghost"
              >
                {copied ? (
                  <Check aria-hidden="true" className="size-4" />
                ) : (
                  <Copy aria-hidden="true" className="size-4" />
                )}
              </Button>
            </div>
          ) : null}
        </section>

        <SheetContent className="subscribe-config-sheet w-[680px] max-w-full md:max-w-3xl">
          <SheetHeader>
            <SheetTitle>
              {t("config.title", "Subscription Configuration")}
            </SheetTitle>
            <SheetDescription>
              {t(
                "config.sheetDescription",
                "Configure subscription policy, public endpoints, and client access rules"
              )}
            </SheetDescription>
          </SheetHeader>
          <ScrollArea className="min-h-0 flex-1 px-6">
            <Form {...form}>
              <form
                className="subscribe-settings-form py-5"
                id="subscribe-config-form"
                onSubmit={form.handleSubmit(onSubmit, handleInvalid)}
              >
                <Tabs
                  className="subscribe-config-tabs"
                  onValueChange={(value) =>
                    setActiveTab(value as SubscribeConfigTab)
                  }
                  value={activeTab}
                >
                  <TabsList className="grid w-full grid-cols-3">
                    <TabsTrigger value="policy">
                      <Layers3 aria-hidden="true" className="size-4" />
                      {t("config.tabs.policy", "Policy")}
                    </TabsTrigger>
                    <TabsTrigger value="endpoint">
                      <Globe2 aria-hidden="true" className="size-4" />
                      {t("config.tabs.endpoint", "Endpoint")}
                    </TabsTrigger>
                    <TabsTrigger value="access">
                      <ShieldCheck aria-hidden="true" className="size-4" />
                      {t("config.tabs.access", "Access")}
                    </TabsTrigger>
                  </TabsList>

                  <section
                    aria-live="polite"
                    className="subscribe-config-live-preview"
                  >
                    <div className="subscribe-config-live-preview-heading">
                      <Link2 aria-hidden="true" className="size-4" />
                      <div>
                        <div className="font-medium text-sm">
                          {t("config.previewTitle", "Applied result")}
                        </div>
                        <p className="text-muted-foreground text-xs">
                          {t(
                            "config.previewDescription",
                            "Preview how the current draft affects generated subscription links"
                          )}
                        </p>
                      </div>
                    </div>
                    <code className="subscribe-config-endpoint-preview">
                      {endpointPreview}
                    </code>
                    <div className="subscribe-config-impact-grid">
                      <div>
                        <span>{t("config.mode", "Subscription mode")}</span>
                        <strong>
                          {draftConfig.single_model
                            ? t("config.singleMode", "Single subscription")
                            : t("config.multiMode", "Multiple subscriptions")}
                        </strong>
                      </div>
                      <div>
                        <span>
                          {t("config.domainSummary", "Domain routing")}
                        </span>
                        <strong>
                          {draftConfig.pan_domain
                            ? t("config.tokenSubdomain", "Token subdomain")
                            : t("config.standardPath", "Standard path")}
                        </strong>
                      </div>
                      <div>
                        <span>User-Agent</span>
                        <strong>
                          {draftConfig.user_agent_limit
                            ? t("config.restricted", "Restricted")
                            : t("config.unrestricted", "Unrestricted")}
                        </strong>
                      </div>
                    </div>
                  </section>

                  <TabsContent
                    className="pt-5 data-[state=inactive]:hidden"
                    forceMount
                    value="policy"
                  >
                    <section className="subscribe-settings-section">
                      <SectionHeader
                        description={t(
                          "config.policyDescription",
                          "Control how many subscriptions a user may keep and how domains resolve"
                        )}
                        icon={Layers3}
                        title={t("config.policy", "Subscription policy")}
                      />
                      <div className="subscribe-setting-switches">
                        <FormField
                          control={form.control}
                          name="single_model"
                          render={({ field }) => (
                            <FormItem className="subscribe-setting-switch">
                              <div>
                                <FormLabel>
                                  {t(
                                    "config.singleSubscriptionMode",
                                    "Single Subscription Mode"
                                  )}
                                </FormLabel>
                                <FormDescription>
                                  {t(
                                    "config.singleSubscriptionModeDescription",
                                    "Limit users to one active subscription. Existing subscriptions unaffected"
                                  )}
                                </FormDescription>
                              </div>
                              <FormControl>
                                <Switch
                                  aria-label={t(
                                    "config.singleSubscriptionMode",
                                    "Single Subscription Mode"
                                  )}
                                  checked={field.value}
                                  onCheckedChange={field.onChange}
                                />
                              </FormControl>
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name="pan_domain"
                          render={({ field }) => (
                            <FormItem className="subscribe-setting-switch">
                              <div>
                                <FormLabel>
                                  {t(
                                    "config.wildcardResolution",
                                    "Wildcard Resolution"
                                  )}
                                </FormLabel>
                                <FormDescription>
                                  {t(
                                    "config.wildcardResolutionDescription",
                                    "Enable wildcard domain resolution for subscriptions"
                                  )}
                                </FormDescription>
                              </div>
                              <FormControl>
                                <Switch
                                  aria-label={t(
                                    "config.wildcardResolution",
                                    "Wildcard Resolution"
                                  )}
                                  checked={field.value}
                                  onCheckedChange={field.onChange}
                                />
                              </FormControl>
                            </FormItem>
                          )}
                        />
                      </div>
                    </section>
                  </TabsContent>

                  <TabsContent
                    className="pt-5 data-[state=inactive]:hidden"
                    forceMount
                    value="endpoint"
                  >
                    <section className="subscribe-settings-section">
                      <SectionHeader
                        description={t(
                          "config.endpointDescription",
                          "Define the public path and optional hostnames used for subscription links"
                        )}
                        icon={Globe2}
                        title={t("config.endpoint", "Subscription endpoint")}
                      />
                      <div className="grid gap-5">
                        <FormField
                          control={form.control}
                          name="subscribe_path"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t(
                                  "config.subscriptionPath",
                                  "Subscription Path"
                                )}
                              </FormLabel>
                              <FormControl>
                                <EnhancedInput
                                  onValueChange={(value) =>
                                    field.onChange(
                                      normalizeSubscribePath(String(value))
                                    )
                                  }
                                  placeholder="/subscribe"
                                  prefix="/"
                                  value={field.value.replace(/^\//, "")}
                                />
                              </FormControl>
                              <FormDescription>
                                {t(
                                  "config.subscriptionPathDescription",
                                  "Custom path for subscription endpoints"
                                )}
                              </FormDescription>
                              <div className="subscribe-path-impact">
                                <ServerCog
                                  aria-hidden="true"
                                  className="size-3.5"
                                />
                                <span>
                                  {t(
                                    "config.subscriptionPathImpact",
                                    "Changing this path restarts the service to register the new route"
                                  )}
                                </span>
                              </div>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name="subscribe_domain"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t(
                                  "config.subscriptionDomain",
                                  "Subscription Domain"
                                )}
                              </FormLabel>
                              <FormControl>
                                <Textarea
                                  className="min-h-28 font-mono text-sm"
                                  {...field}
                                  onBlur={() => {
                                    field.onBlur();
                                    form.setValue(
                                      "subscribe_domain",
                                      normalizeLines(field.value),
                                      {
                                        shouldDirty: true,
                                        shouldValidate: true,
                                      }
                                    );
                                  }}
                                  placeholder="subscribe.example.com\nedge.example.com"
                                />
                              </FormControl>
                              <FormDescription>
                                {t(
                                  "config.subscriptionDomainDescription",
                                  "Enter concrete hostnames only, one per line. Wildcards are not valid subscription URLs"
                                )}
                              </FormDescription>
                              <div className="subscribe-field-count">
                                <ListChecks
                                  aria-hidden="true"
                                  className="size-3.5"
                                />
                                {t(
                                  "config.domainEntries",
                                  "{{count}} domain entries",
                                  { count: draftDomains }
                                )}
                              </div>
                              {invalidDraftDomains > 0 ? (
                                <div className="subscribe-field-warning">
                                  <AlertTriangle
                                    aria-hidden="true"
                                    className="size-3.5"
                                  />
                                  <span>
                                    {t(
                                      "config.invalidConfiguredDomains",
                                      "{{count}} configured domain entries are invalid and must be corrected before saving",
                                      { count: invalidDraftDomains }
                                    )}
                                  </span>
                                </div>
                              ) : null}
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                    </section>
                  </TabsContent>

                  <TabsContent
                    className="pt-5 data-[state=inactive]:hidden"
                    forceMount
                    value="access"
                  >
                    <section className="subscribe-settings-section">
                      <SectionHeader
                        description={t(
                          "config.accessDescription",
                          "Restrict unknown clients while keeping configured application identifiers available"
                        )}
                        icon={ShieldCheck}
                        title={t("config.accessControl", "Access control")}
                      />
                      <FormField
                        control={form.control}
                        name="user_agent_limit"
                        render={({ field }) => (
                          <FormItem className="subscribe-setting-switch">
                            <div>
                              <FormLabel>
                                {t(
                                  "config.userAgentLimit",
                                  "User-Agent Restriction",
                                  {
                                    userAgent: "User-Agent",
                                  }
                                )}
                              </FormLabel>
                              <FormDescription>
                                {t(
                                  "config.userAgentLimitDescription",
                                  "Enable access restrictions based on User-Agent",
                                  { userAgent: "User-Agent" }
                                )}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                aria-label={t(
                                  "config.userAgentLimit",
                                  "User-Agent Restriction",
                                  { userAgent: "User-Agent" }
                                )}
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name="user_agent_list"
                        render={({ field }) => (
                          <FormItem
                            className={
                              !userAgentLimit ? "opacity-55" : undefined
                            }
                          >
                            <FormLabel>
                              {t(
                                "config.userAgentList",
                                "User-Agent Whitelist",
                                {
                                  userAgent: "User-Agent",
                                }
                              )}
                            </FormLabel>
                            <FormControl>
                              <Textarea
                                className="min-h-32 font-mono text-sm"
                                disabled={!userAgentLimit}
                                {...field}
                                onBlur={() => {
                                  field.onBlur();
                                  form.setValue(
                                    "user_agent_list",
                                    normalizeLines(field.value),
                                    {
                                      shouldDirty: true,
                                      shouldValidate: true,
                                    }
                                  );
                                }}
                                placeholder="ClashX\nClash-verge\nsing-box"
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "config.userAgentListDescription",
                                "One identifier per line. Configured client identifiers are included automatically",
                                { userAgent: "User-Agent" }
                              )}
                            </FormDescription>
                            <div className="subscribe-field-count">
                              <ListChecks
                                aria-hidden="true"
                                className="size-3.5"
                              />
                              {draftConfig.user_agent_limit
                                ? t(
                                    "config.agentEntries",
                                    "{{count}} manual allow-list entries",
                                    { count: draftAgents }
                                  )
                                : t(
                                    "config.agentEntriesInactive",
                                    "Restriction is disabled"
                                  )}
                            </div>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </section>
                  </TabsContent>
                </Tabs>
              </form>
            </Form>
          </ScrollArea>
          <SheetFooter className="subscribe-config-footer flex-row justify-between gap-2">
            <span className="hidden text-muted-foreground text-xs sm:block">
              {form.formState.isDirty
                ? t("config.unsavedChanges", "Unsaved changes")
                : t("config.noChanges", "No unsaved changes")}
            </span>
            <div className="flex gap-2">
              <Button
                disabled={saving}
                onClick={() => handleOpenChange(false)}
                type="button"
                variant="outline"
              >
                {t("actions.cancel", "Cancel")}
              </Button>
              <Button
                disabled={saving || !isDirty || !isValid}
                onClick={() => form.handleSubmit(onSubmit, handleInvalid)()}
                type="button"
              >
                {saving ? <RefreshCcw className="size-4 animate-spin" /> : null}
                {t("actions.save", "Save")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <DiscardChangesDialog
        onDiscard={discardChanges}
        onOpenChange={setDiscardOpen}
        open={discardOpen}
      />
    </>
  );
}
