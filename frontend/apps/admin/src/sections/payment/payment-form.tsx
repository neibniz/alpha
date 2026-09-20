import { AdminForm } from "@/components/admin-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@workspace/ui/components/form";
import {
  RadioGroup,
  RadioGroupItem,
} from "@workspace/ui/components/radio-group";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import { Switch } from "@workspace/ui/components/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import { LazyMarkdownEditor } from "@/components/lazy-markdown-editor";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import { getPaymentPlatform } from "@workspace/ui/services/admin/payment";
import { normalizeListResult } from "@workspace/ui/utils/pagination";
import { unitConversion } from "@workspace/ui/utils/unit-conversions";
import {
  cloneElement,
  isValidElement,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useMemo,
  useState,
} from "react";
import { CircleAlert } from "lucide-react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import * as z from "zod";
import { useGlobalStore } from "@/stores/global";
import { DiscardChangesDialog } from "@/components/discard-changes-dialog";
import {
  canonicalPaymentPlatform,
  isHttpUrl,
  normalizePaymentFees,
  PAYMENT_NAME_MAX_LENGTH,
  PAYMENT_TEXT_MAX_LENGTH,
  paymentConfigFieldIsRequired,
} from "./payment-form-data";

type PaymentConfigMap = Record<string, unknown>;

type PaymentFormValues = {
  name: string;
  platform: string;
  icon?: string;
  domain?: string;
  config: PaymentConfigMap;
  fee_mode: number;
  fee_percent?: number;
  fee_amount?: number;
  description?: string;
};

type PaymentInitialValues = Omit<Partial<PaymentFormValues>, "config"> & {
  config?: unknown;
};

interface PaymentFormProps<T extends { platform?: string }> {
  trigger: ReactNode;
  title: string;
  loading?: boolean;
  initialValues?: Partial<T> & PaymentInitialValues;
  loadInitialValues?: () => Promise<
    (Partial<T> & PaymentInitialValues) | undefined
  >;
  onSubmit: (values: T) => Promise<boolean>;
  isEdit?: boolean;
}

function isRecord(value: unknown): value is PaymentConfigMap {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function buildPaymentFormValues(
  values?: PaymentInitialValues
): PaymentFormValues {
  return {
    name: values?.name ?? "",
    platform: canonicalPaymentPlatform(values?.platform),
    icon: values?.icon ?? "",
    domain: values?.domain ?? "",
    config: isRecord(values?.config) ? values.config : {},
    fee_mode: values?.fee_mode ?? 0,
    fee_percent: values?.fee_percent ?? 0,
    fee_amount: values?.fee_amount ?? 0,
    description: values?.description ?? "",
  };
}

function normalizeFieldDescriptions(
  value: Record<string, unknown> | undefined
): Record<string, string> {
  if (!value) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(value).filter(
      (entry): entry is [string, string] =>
        typeof entry[1] === "string" && entry[1].length > 0
    )
  );
}

export default function PaymentForm<T extends { platform?: string }>({
  trigger,
  title,
  loading,
  initialValues,
  loadInitialValues,
  onSubmit,
  isEdit,
}: PaymentFormProps<T>) {
  const { t } = useTranslation("payment");
  const { common } = useGlobalStore();
  const { currency } = common;
  const [open, setOpen] = useState(false);
  const [opening, setOpening] = useState(false);
  const [discardOpen, setDiscardOpen] = useState(false);

  const {
    data: platformData,
    isError: platformError,
    isPending: platformPending,
    refetch: refetchPlatforms,
  } = useQuery({
    queryKey: ["getPaymentPlatform"],
    queryFn: async () => {
      const { data } = await getPaymentPlatform();
      return normalizeListResult(data?.data);
    },
  });

  const platformByName = useMemo(
    () =>
      new Map(platformData?.map((platform) => [platform.platform, platform])),
    [platformData]
  );

  const formSchema = useMemo(
    () =>
      z
        .object({
          name: z
            .string()
            .trim()
            .min(1, { message: t("nameRequired", "Name is required") })
            .max(PAYMENT_NAME_MAX_LENGTH, {
              message: t("nameTooLong", "Name must be 100 characters or fewer"),
            }),
          platform: z
            .string()
            .trim()
            .min(1, {
              message: t("platformRequired", "Select a payment platform"),
            }),
          icon: z.string().trim().max(255).optional(),
          domain: z.string().trim().max(255).optional(),
          config: z.record(z.string(), z.unknown()),
          fee_mode: z.number().int().min(0).max(3),
          fee_percent: z.number().int().min(0).max(100).optional(),
          fee_amount: z.number().int().min(0).optional(),
          description: z.string().max(PAYMENT_TEXT_MAX_LENGTH).optional(),
        })
        .superRefine((values, ctx) => {
          if (values.platform !== "balance") {
            if (!values.domain) {
              ctx.addIssue({
                code: "custom",
                message: t("domainRequired", "Callback domain is required"),
                path: ["domain"],
              });
            } else if (!isHttpUrl(values.domain)) {
              ctx.addIssue({
                code: "custom",
                message: t(
                  "domainInvalid",
                  "Enter an absolute HTTP or HTTPS URL"
                ),
                path: ["domain"],
              });
            }
          }
          const fields = normalizeFieldDescriptions(
            platformByName.get(values.platform)?.platform_field_description
          );
          for (const field of Object.keys(fields)) {
            if (!paymentConfigFieldIsRequired(field)) continue;
            const value = values.config[field];
            if (typeof value !== "string" || value.trim() === "") {
              ctx.addIssue({
                code: "custom",
                message: t("configRequired", {
                  field: fields[field],
                  defaultValue: "{{field}} is required",
                }),
                path: ["config"],
              });
              break;
            }
          }
        }),
    [platformByName, t]
  );

  const form = useForm<PaymentFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: buildPaymentFormValues(initialValues),
  });

  const feeMode = form.watch("fee_mode");
  const platformValue = form.watch("platform");
  const configValues = form.watch("config");

  const currentPlatform = platformValue
    ? platformByName.get(platformValue)
    : undefined;
  const currentFieldDescriptions = normalizeFieldDescriptions(
    currentPlatform?.platform_field_description
  );
  const configFields = Object.keys(currentFieldDescriptions);
  const platformUrl = currentPlatform?.platform_url || "";
  const hasSelectedPlatform =
    !platformValue || platformByName.has(platformValue);

  useEffect(() => {
    if (feeMode === 0) {
      form.setValue("fee_amount", 0);
      form.setValue("fee_percent", 0);
    } else if (feeMode === 1) {
      form.setValue("fee_amount", 0);
    } else if (feeMode === 2) {
      form.setValue("fee_percent", 0);
    }
  }, [feeMode, form]);

  const closeAndReset = () => {
    form.reset();
    setOpen(false);
  };

  const requestClose = () => {
    if (form.formState.isDirty) {
      setDiscardOpen(true);
      return;
    }
    closeAndReset();
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setOpen(true);
    } else {
      requestClose();
    }
  };

  async function handleOpen() {
    setOpen(true);
    setOpening(true);
    try {
      const values = loadInitialValues
        ? await loadInitialValues()
        : initialValues;
      if (loadInitialValues && !values) {
        setOpen(false);
        return;
      }
      form.reset(buildPaymentFormValues(values));
    } finally {
      setOpening(false);
    }
  }

  function renderTrigger() {
    if (isValidElement(trigger)) {
      const element = trigger as ReactElement<{
        disabled?: boolean;
        onClick?: (event: MouseEvent<HTMLElement>) => void;
      }>;
      return cloneElement(element, {
        disabled: opening || element.props.disabled,
        onClick: (event: MouseEvent<HTMLElement>) => {
          element.props.onClick?.(event);
          if (!event.defaultPrevented) {
            handleOpen();
          }
        },
      });
    }
    return (
      <Button disabled={opening} onClick={handleOpen} type="button">
        {opening && <Icon className="mr-2 animate-spin" icon="mdi:loading" />}
        {trigger}
      </Button>
    );
  }

  const handleSubmit = async (values: PaymentFormValues) => {
    const cleanedValues = {
      ...values,
      name: values.name.trim(),
      icon: values.icon?.trim(),
      domain: values.domain?.trim(),
      ...normalizePaymentFees(values),
    };

    const success = await onSubmit(cleanedValues as unknown as T);
    if (success) {
      closeAndReset();
    }
  };

  const openPlatformUrl = () => {
    if (platformUrl) {
      window.open(platformUrl, "_blank");
    }
  };

  return (
    <>
      <Sheet onOpenChange={handleOpenChange} open={open}>
        {renderTrigger()}
        <SheetContent className="w-[550px] max-w-full md:max-w-screen-md">
          <SheetHeader>
            <SheetTitle>{title}</SheetTitle>
          </SheetHeader>
          <ScrollArea className="h-[calc(100vh-48px-36px-36px-24px-env(safe-area-inset-top))]">
            {opening ? (
              <div className="flex min-h-48 items-center justify-center text-muted-foreground text-sm">
                <Icon className="mr-2 animate-spin" icon="mdi:loading" />
                {t("detailLoading", "Loading payment method...")}
              </div>
            ) : (
              <Form {...form}>
                <AdminForm
                  className="space-y-6 px-6 pt-4"
                  onSubmit={form.handleSubmit(handleSubmit)}
                >
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                      <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("name", "Name")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "namePlaceholder",
                                  "Enter payment method name"
                                )}
                                value={field.value}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name="icon"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("icon", "Icon")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "iconPlaceholder",
                                  "Enter icon URL"
                                )}
                                value={field.value}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    <FormField
                      control={form.control}
                      name="domain"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("domain", "Domain")}</FormLabel>
                          <FormControl>
                            <EnhancedInput
                              onValueChange={field.onChange}
                              placeholder="http(s)://example.com"
                              value={field.value}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <div className="space-y-4">
                    <FormField
                      control={form.control}
                      name="fee_mode"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("handlingFee", "Handling Fee")}
                          </FormLabel>
                          <FormControl>
                            <RadioGroup
                              className="flex flex-wrap gap-4"
                              onValueChange={(value) =>
                                field.onChange(Number.parseInt(value, 10))
                              }
                              value={field.value.toString()}
                            >
                              <FormItem className="flex items-center space-x-2">
                                <FormControl>
                                  <RadioGroupItem value="0" />
                                </FormControl>
                                <FormLabel className="!mt-0 cursor-pointer">
                                  {t("noFee", "No Fee")}
                                </FormLabel>
                              </FormItem>
                              <FormItem className="flex items-center space-x-2">
                                <FormControl>
                                  <RadioGroupItem value="3" />
                                </FormControl>
                                <FormLabel className="!mt-0 cursor-pointer">
                                  {t("combinedFee", "Percentage + fixed")}
                                </FormLabel>
                              </FormItem>
                              <FormItem className="flex items-center space-x-2">
                                <FormControl>
                                  <RadioGroupItem value="1" />
                                </FormControl>
                                <FormLabel className="!mt-0 cursor-pointer">
                                  {t("percentFee", "Percentage")}
                                </FormLabel>
                              </FormItem>
                              <FormItem className="flex items-center space-x-2">
                                <FormControl>
                                  <RadioGroupItem value="2" />
                                </FormControl>
                                <FormLabel className="!mt-0 cursor-pointer">
                                  {t("fixedFee", "Fixed Amount")}
                                </FormLabel>
                              </FormItem>
                            </RadioGroup>
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    {(feeMode === 1 || feeMode === 3) && (
                      <div className="grid grid-cols-1 sm:w-1/2">
                        <FormField
                          control={form.control}
                          name="fee_percent"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t("feePercent", "Fee Percentage")}
                              </FormLabel>
                              <FormControl>
                                <EnhancedInput
                                  max={100}
                                  min={0}
                                  onValueChange={field.onChange}
                                  step="1"
                                  suffix="%"
                                  type="number"
                                  value={field.value}
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                    )}

                    {(feeMode === 2 || feeMode === 3) && (
                      <div className="grid grid-cols-1 sm:w-1/2">
                        <FormField
                          control={form.control}
                          name="fee_amount"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t("feeAmount", "Fixed Amount")}
                              </FormLabel>
                              <FormControl>
                                <EnhancedInput
                                  formatInput={(value) =>
                                    unitConversion("centsToDollars", value)
                                  }
                                  formatOutput={(value) =>
                                    unitConversion("dollarsToCents", value)
                                  }
                                  min={0}
                                  onValueChange={(value) =>
                                    field.onChange(value)
                                  }
                                  prefix={currency.currency_symbol}
                                  step="0.01"
                                  suffix={currency.currency_unit}
                                  type="number"
                                  value={field.value}
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                    )}
                  </div>

                  <div className="space-y-4">
                    {hasSelectedPlatform && (
                      <FormField
                        control={form.control}
                        name="platform"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("platform", "Platform")}</FormLabel>
                            <Select
                              defaultValue={field.value}
                              disabled={
                                isEdit && Boolean(initialValues?.platform)
                              }
                              onValueChange={(value) => {
                                field.onChange(value);
                                form.setValue(
                                  "config",
                                  {},
                                  { shouldDirty: true }
                                );
                              }}
                              value={field.value}
                            >
                              <FormControl>
                                <SelectTrigger>
                                  <SelectValue
                                    placeholder={t(
                                      "selectPlatform",
                                      "Select Platform"
                                    )}
                                  />
                                </SelectTrigger>
                              </FormControl>
                              <SelectContent>
                                {platformData?.map((platform) => (
                                  <SelectItem
                                    key={platform.platform}
                                    value={platform.platform}
                                  >
                                    {platform.platform}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            {platformUrl ? (
                              <div className="mt-1 flex justify-end">
                                <Button
                                  className="h-6 px-2 text-xs"
                                  onClick={openPlatformUrl}
                                  size="sm"
                                  type="button"
                                  variant="ghost"
                                >
                                  <Icon
                                    className="mr-1 h-3 w-3"
                                    icon="tabler:external-link"
                                  />
                                  {t("applyForPayment", "Apply for Payment")}
                                </Button>
                              </div>
                            ) : (
                              <div className="mt-1 h-6" />
                            )}
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    )}

                    {platformPending ? (
                      <p className="text-muted-foreground text-sm">
                        {t("platformLoading", "Loading payment platforms...")}
                      </p>
                    ) : null}
                    {platformError ? (
                      <Alert variant="destructive">
                        <CircleAlert />
                        <AlertTitle>
                          {t(
                            "platformLoadFailed",
                            "Payment platforms could not be loaded"
                          )}
                        </AlertTitle>
                        <AlertDescription>
                          {t(
                            "platformLoadFailedDescription",
                            "Retry before selecting a payment provider."
                          )}
                        </AlertDescription>
                        <Button
                          className="ml-auto"
                          onClick={() => refetchPlatforms()}
                          size="sm"
                          type="button"
                          variant="outline"
                        >
                          {t("retry", "Retry")}
                        </Button>
                      </Alert>
                    ) : null}

                    {configFields.length > 0 && (
                      <div className="mt-4 space-y-4">
                        {configFields.map((fieldKey) => (
                          <FormItem key={fieldKey}>
                            <FormLabel>
                              {currentFieldDescriptions[fieldKey]}
                            </FormLabel>
                            <FormControl>
                              {fieldKey === "sandbox" ? (
                                <Switch
                                  checked={Boolean(configValues?.[fieldKey])}
                                  onCheckedChange={(value) =>
                                    form.setValue(
                                      "config",
                                      { ...configValues, [fieldKey]: value },
                                      { shouldDirty: true }
                                    )
                                  }
                                />
                              ) : (
                                <EnhancedInput
                                  disabled={fieldKey === "webhook_secret"}
                                  onValueChange={(value) =>
                                    form.setValue(
                                      "config",
                                      { ...configValues, [fieldKey]: value },
                                      { shouldDirty: true }
                                    )
                                  }
                                  placeholder={t("configPlaceholder", {
                                    field: currentFieldDescriptions[fieldKey],
                                    defaultValue:
                                      "Please fill in the provided {{field}} configuration",
                                  })}
                                  type={
                                    fieldKey.includes("secret") ||
                                    fieldKey.includes("key")
                                      ? "password"
                                      : "text"
                                  }
                                  value={configValues?.[fieldKey] ?? ""}
                                />
                              )}
                            </FormControl>
                          </FormItem>
                        ))}
                        {form.formState.errors.config?.message ? (
                          <p className="font-medium text-destructive text-sm">
                            {String(form.formState.errors.config.message)}
                          </p>
                        ) : null}
                      </div>
                    )}

                    <FormField
                      control={form.control}
                      name="description"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("description", "Description")}
                          </FormLabel>
                          <FormControl>
                            <LazyMarkdownEditor
                              onChange={(value: string | undefined) =>
                                form.setValue(field.name, value as string, {
                                  shouldDirty: true,
                                })
                              }
                              value={field.value}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </AdminForm>
              </Form>
            )}
          </ScrollArea>

          <SheetFooter className="flex-row justify-end gap-2 pt-3">
            <span className="mr-auto hidden text-muted-foreground text-xs sm:block">
              {form.formState.isDirty
                ? t("unsavedChanges", "Unsaved changes")
                : t("noChanges", "No unsaved changes")}
            </span>
            <Button
              disabled={loading}
              onClick={requestClose}
              type="button"
              variant="outline"
            >
              {t("cancel", "Cancel")}
            </Button>
            <Button
              disabled={loading || opening || platformPending}
              onClick={form.handleSubmit(handleSubmit)}
              type="button"
            >
              {loading && (
                <Icon className="mr-2 animate-spin" icon="mdi:loading" />
              )}
              {t("submit", "Submit")}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <DiscardChangesDialog
        onDiscard={closeAndReset}
        onOpenChange={setDiscardOpen}
        open={discardOpen}
      />
    </>
  );
}
