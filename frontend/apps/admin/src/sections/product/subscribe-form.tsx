"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@workspace/ui/components/accordion";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { Checkbox } from "@workspace/ui/components/checkbox";
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
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
} from "@workspace/ui/components/empty";
import { Label } from "@workspace/ui/components/label";
import { Input } from "@workspace/ui/components/input";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
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
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Switch } from "@workspace/ui/components/switch";
import { Textarea } from "@workspace/ui/components/textarea";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import { Combobox } from "@workspace/ui/composed/combobox";
import { ArrayInput } from "@workspace/ui/composed/dynamic-Inputs";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import {
  evaluateWithPrecision,
  unitConversion,
} from "@workspace/ui/utils/unit-conversions";
import {
  CreditCard,
  ListPlus,
  GalleryHorizontal,
  Plus,
  Server,
  Settings,
  Trash2,
} from "lucide-react";
import { shake } from "radash";
import {
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { useGlobalStore } from "@/stores/global";
import { useNode } from "@/stores/node";
import {
  parseProductDescription,
  type ProductDescription,
  type ProductFeature,
  type ProductFeatureType,
  serializeProductDescription,
} from "./product-description";

interface SubscribeFormProps<T> {
  onSubmit: (data: T) => Promise<boolean> | boolean;
  initialValues?: Partial<T>;
  loadInitialValues?: () => Promise<Partial<T> | undefined>;
  loading?: boolean;
  trigger: ReactNode;
  title: string;
}

const defaultValues = {
  inventory: undefined,
  speed_limit: 0,
  device_limit: 0,
  traffic: 0,
  quota: 0,
  discount: [],
  language: "",
  node_tags: [],
  nodes: [],
  unit_time: "Month",
  deduction_ratio: 0,
  purchase_with_discount: false,
  reset_cycle: 0,
  allow_renewal: true,
  allow_reset_traffic: true,
  renewal_reset: false,
  show_original_price: false,
  deduction_mode: "auto",
  show: false,
  user_visible: false,
  sell: false,
  display_template: "standard" as const,
  badge: "",
  min_quantity: 1,
  max_quantity: 0,
  sale_starts_at: 0,
  sale_ends_at: 0,
};

function timestampToLocalInput(value?: number) {
  if (!value) return "";
  const milliseconds = value < 1_000_000_000_000 ? value * 1000 : value;
  const date = new Date(milliseconds);
  if (Number.isNaN(date.getTime())) return "";
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
    .toISOString()
    .slice(0, 16);
}

function localInputToTimestamp(value: string) {
  if (!value) return 0;
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

function ProductDescriptionEditor({
  onChange,
  value,
}: {
  onChange: (value: string) => void;
  value?: string;
}) {
  const { t } = useTranslation("product");
  const description = useMemo(() => parseProductDescription(value), [value]);

  const update = (next: ProductDescription) => {
    onChange(serializeProductDescription(next));
  };
  const updateFeature = (index: number, patch: Partial<ProductFeature>) => {
    update({
      ...description,
      features: description.features.map((feature, featureIndex) =>
        featureIndex === index ? { ...feature, ...patch } : feature
      ),
    });
  };

  return (
    <div className="space-y-5">
      <div className="space-y-2">
        <FormLabel>{t("form.description", "Product summary")}</FormLabel>
        <Textarea
          className="min-h-24 resize-y"
          maxLength={240}
          onChange={(event) =>
            update({ ...description, description: event.target.value })
          }
          placeholder={t(
            "form.descriptionPlaceholder",
            "Explain who this plan is for and its main value."
          )}
          value={description.description}
        />
        <p className="text-muted-foreground text-xs">
          {t(
            "form.descriptionHelp",
            "Shown on the user-facing product card and checkout page."
          )}
        </p>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between gap-3">
          <div>
            <FormLabel>{t("form.features", "Selling points")}</FormLabel>
            <p className="mt-1 text-muted-foreground text-xs">
              {t(
                "form.featuresHelp",
                "Keep benefits concise and ordered by importance."
              )}
            </p>
          </div>
          <Button
            onClick={() =>
              update({
                ...description,
                features: [
                  ...description.features,
                  { icon: "uil:check", label: "", type: "default" },
                ],
              })
            }
            size="sm"
            type="button"
            variant="outline"
          >
            <Plus className="size-4" />
            {t("form.addFeature", "Add selling point")}
          </Button>
        </div>

        {description.features.length > 0 ? (
          <div className="divide-y overflow-hidden rounded-md border">
            {description.features.map((feature, index) => (
              <div
                className="grid gap-3 p-3 sm:grid-cols-[minmax(0,1fr)_180px_132px_32px] sm:items-center"
                key={`feature-${index}`}
              >
                <Input
                  aria-label={t("form.featureLabel", "Selling point")}
                  onChange={(event) =>
                    updateFeature(index, { label: event.target.value })
                  }
                  placeholder={t(
                    "form.featureLabelPlaceholder",
                    "Example: Priority network access"
                  )}
                  value={feature.label}
                />
                <div className="relative">
                  {feature.icon ? (
                    <Icon
                      className="-translate-y-1/2 absolute top-1/2 left-3 size-4 text-muted-foreground"
                      icon={feature.icon}
                    />
                  ) : null}
                  <Input
                    aria-label={t("form.featureIcon", "Icon")}
                    className={feature.icon ? "pl-9" : undefined}
                    onChange={(event) =>
                      updateFeature(index, { icon: event.target.value })
                    }
                    placeholder="uil:check"
                    value={feature.icon}
                  />
                </div>
                <Select
                  onValueChange={(type) =>
                    updateFeature(index, {
                      type: type as ProductFeatureType,
                    })
                  }
                  value={feature.type}
                >
                  <SelectTrigger aria-label={t("form.featureTone", "Tone")}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="default">
                      {t("form.featureToneDefault", "Default")}
                    </SelectItem>
                    <SelectItem value="success">
                      {t("form.featureToneSuccess", "Highlight")}
                    </SelectItem>
                    <SelectItem value="destructive">
                      {t("form.featureToneWarning", "Warning")}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  aria-label={t("form.removeFeature", "Remove selling point")}
                  onClick={() =>
                    update({
                      ...description,
                      features: description.features.filter(
                        (_, featureIndex) => featureIndex !== index
                      ),
                    })
                  }
                  size="icon-sm"
                  type="button"
                  variant="ghost"
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <Empty className="border px-4 py-6">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ListPlus />
              </EmptyMedia>
              <EmptyDescription>
                {t("form.featuresEmpty", "No selling points yet")}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
      </div>
    </div>
  );
}

export default function SubscribeForm<T extends object>({
  onSubmit,
  initialValues,
  loadInitialValues,
  loading,
  trigger,
  title,
}: Readonly<SubscribeFormProps<T>>) {
  const { common } = useGlobalStore();
  const { currency } = common;

  const { t } = useTranslation("product");
  const [open, setOpen] = useState(false);
  const [resolvedInitialValues, setResolvedInitialValues] = useState<
    Partial<T> | undefined
  >(initialValues);
  const [loadingInitialValues, setLoadingInitialValues] = useState(false);
  const updateTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const formSchema = z
    .object({
      name: z.string(),
      description: z.string().optional(),
      unit_price: z.number().int().nonnegative(),
      unit_time: z.string(),
      replacement: z.number().int().nonnegative().optional(),
      discount: z
        .array(
          z.object({
            quantity: z.number().int().positive(),
            discount: z.number().min(0).max(100),
          })
        )
        .optional(),
      inventory: z.number().int().min(-1).optional(),
      speed_limit: z.number().int().nonnegative().optional(),
      device_limit: z.number().int().nonnegative().optional(),
      traffic: z.number().int().nonnegative().optional(),
      quota: z.number().int().nonnegative().optional(),
      language: z.string().optional(),
      node_tags: z.array(z.string()).optional(),
      nodes: z.array(z.number().int().positive()).optional(),
      deduction_ratio: z.number().int().min(0).max(100).optional(),
      allow_deduction: z.boolean().optional(),
      allow_renewal: z.boolean().optional(),
      allow_reset_traffic: z.boolean().optional(),
      reset_cycle: z.number().int().min(0).max(3).optional(),
      renewal_reset: z.boolean().optional(),
      show_original_price: z.boolean().optional(),
      show: z.boolean().optional(),
      user_visible: z.boolean().optional(),
      sell: z.boolean().optional(),
      display_template: z
        .enum(["standard", "popular", "premium", "compact"])
        .optional(),
      badge: z.string().max(64).optional(),
      min_quantity: z.number().int().min(1).max(1000).optional(),
      max_quantity: z.number().int().min(0).max(1000).optional(),
      sale_starts_at: z.number().int().min(0).optional(),
      sale_ends_at: z.number().int().min(0).optional(),
    })
    .superRefine((values, context) => {
      const minimum = values.min_quantity ?? 1;
      const maximum = values.max_quantity ?? 0;
      if (maximum > 0 && maximum < minimum) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: t(
            "form.quantityRangeInvalid",
            "Maximum quantity must be greater than or equal to minimum quantity"
          ),
          path: ["max_quantity"],
        });
      }
      if (
        values.sale_starts_at &&
        values.sale_ends_at &&
        values.sale_ends_at <= values.sale_starts_at
      ) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: t(
            "form.saleWindowInvalid",
            "End time must be after start time"
          ),
          path: ["sale_ends_at"],
        });
      }
    });
  type SubscribeFormValues = z.infer<typeof formSchema>;
  type DiscountFormItem = API.SubscribeDiscount & { price?: number };
  const activeInitialValues = resolvedInitialValues ?? initialValues;
  const normalizedInitialValues = useMemo(
    () =>
      shake(
        activeInitialValues,
        (value) => value === null
      ) as Partial<SubscribeFormValues>,
    [activeInitialValues]
  );

  const form = useForm<SubscribeFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      ...defaultValues,
      ...normalizedInitialValues,
    },
  });

  const debouncedCalculateDiscount = useCallback(
    (
      values: DiscountFormItem[],
      fieldName: "discount",
      lastChangedField?: string,
      changedIndex?: number
    ) => {
      if (updateTimeoutRef.current) {
        clearTimeout(updateTimeoutRef.current);
      }

      updateTimeoutRef.current = setTimeout(() => {
        const { unit_price } = form.getValues();
        if (!(unit_price && values?.length)) return;

        let hasChanges = false;
        const calculatedValues = values.map((item, index) => {
          const result = { ...item };

          if (changedIndex !== undefined && index !== changedIndex) {
            return result;
          }

          const quantity = Number(item.quantity) || 0;
          const discount = Number(item.discount) || 0;
          const price = Number(item.price) || 0;

          switch (lastChangedField) {
            case "quantity":
            case "discount":
              if (quantity > 0 && discount > 0) {
                const newPrice = evaluateWithPrecision(
                  (unit_price * quantity * discount) / 100
                );
                if (Math.abs(newPrice - price) > 0.01) {
                  result.price = newPrice;
                  hasChanges = true;
                }
              }
              break;

            case "price":
              if (quantity > 0 && price > 0) {
                const newDiscount = evaluateWithPrecision(
                  (price / quantity / unit_price) * 100
                );
                if (Math.abs(newDiscount - discount) > 0.01) {
                  result.discount = Math.min(100, Math.max(0, newDiscount));
                  hasChanges = true;
                }
              } else if (discount > 0 && price > 0) {
                const newQuantity = evaluateWithPrecision(
                  (price / unit_price / discount) * 100
                );
                if (
                  Math.abs(newQuantity - quantity) > 0.01 &&
                  newQuantity > 0
                ) {
                  result.quantity = Math.max(1, Math.round(newQuantity));
                  hasChanges = true;
                }
              }
              break;

            default:
              if (quantity > 0 && discount > 0 && price === 0) {
                result.price = evaluateWithPrecision(
                  (unit_price * quantity * discount) / 100
                );
                hasChanges = true;
              } else if (quantity > 0 && price > 0 && discount === 0) {
                const newDiscount = evaluateWithPrecision(
                  (price / quantity / unit_price) * 100
                );
                result.discount = Math.min(100, Math.max(0, newDiscount));
                hasChanges = true;
              } else if (discount > 0 && price > 0 && quantity === 0) {
                const newQuantity = evaluateWithPrecision(
                  (price / unit_price / discount) * 100
                );
                if (newQuantity > 0) {
                  result.quantity = Math.max(1, Math.round(newQuantity));
                  hasChanges = true;
                }
              }
              break;
          }

          return result;
        });

        if (hasChanges) {
          form.setValue(fieldName, calculatedValues, {
            shouldDirty: true,
          });
        }
      }, 300);
    },
    [form]
  );

  useEffect(() => {
    setResolvedInitialValues(initialValues);
  }, [initialValues]);

  useEffect(() => {
    form?.reset({
      ...defaultValues,
      ...normalizedInitialValues,
    });
    const discount = form.getValues("discount") || [];
    if (discount.length > 0) {
      debouncedCalculateDiscount(discount, "discount");
    }
  }, [form, normalizedInitialValues, open, debouncedCalculateDiscount]);

  useEffect(
    () => () => {
      if (updateTimeoutRef.current) {
        clearTimeout(updateTimeoutRef.current);
      }
    },
    []
  );

  async function handleSubmit(data: SubscribeFormValues) {
    const bool = await onSubmit(data as T);
    if (bool) setOpen(false);
  }

  async function handleOpen() {
    const initialFormValues = shake(initialValues, (value) => value === null);
    setResolvedInitialValues(initialValues);
    form.reset({
      ...defaultValues,
      ...(initialFormValues as Partial<SubscribeFormValues>),
    });
    setOpen(true);

    if (!loadInitialValues) {
      return;
    }

    setLoadingInitialValues(true);
    try {
      const loadedInitialValues = await loadInitialValues();
      if (loadedInitialValues) {
        setResolvedInitialValues(loadedInitialValues);
      }
    } catch {
      toast.error(t("loadFailed", "Load failed"));
    } finally {
      setLoadingInitialValues(false);
    }
  }

  function renderTrigger() {
    if (isValidElement(trigger)) {
      const element = trigger as ReactElement<{
        disabled?: boolean;
        onClick?: () => void;
      }>;
      return cloneElement(element, {
        disabled: loadingInitialValues || element.props.disabled,
        onClick: handleOpen,
      });
    }
    return <Button onClick={handleOpen}>{trigger}</Button>;
  }

  const { getAllAvailableTags, getNodesByTag, getNodesWithoutTags } = useNode();

  const tagGroups = getAllAvailableTags();

  const unit_time = form.watch("unit_time");

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <SheetTrigger asChild>{renderTrigger()}</SheetTrigger>
      <SheetContent
        className="w-[min(920px,calc(100vw-24px))] max-w-full gap-0 md:max-w-4xl"
        onClose={() => setOpen(false)}
      >
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription className="sr-only">
            {t(
              "form.editorDescription",
              "Configure product details, pricing, and availability."
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <form className="pt-4" onSubmit={form.handleSubmit(handleSubmit)}>
              <Tabs className="w-full" defaultValue="basic">
                <TabsList className="mb-6 grid w-full grid-cols-4">
                  <TabsTrigger
                    className="flex items-center gap-1 px-1 text-xs sm:gap-2 sm:text-sm"
                    value="basic"
                  >
                    <Settings className="hidden h-4 w-4 sm:block" />
                    {t("form.basic")}
                  </TabsTrigger>
                  <TabsTrigger
                    className="flex items-center gap-1 px-1 text-xs sm:gap-2 sm:text-sm"
                    value="pricing"
                  >
                    <CreditCard className="hidden h-4 w-4 sm:block" />
                    {t("form.pricing")}
                  </TabsTrigger>
                  <TabsTrigger
                    className="flex items-center gap-1 px-1 text-xs sm:gap-2 sm:text-sm"
                    value="storefront"
                  >
                    <GalleryHorizontal className="hidden h-4 w-4 sm:block" />
                    {t("form.storefront", "上架与展示")}
                  </TabsTrigger>
                  <TabsTrigger
                    className="flex items-center gap-1 px-1 text-xs sm:gap-2 sm:text-sm"
                    value="servers"
                  >
                    <Server className="hidden h-4 w-4 sm:block" />
                    {t("form.nodes")}
                  </TabsTrigger>
                </TabsList>

                <TabsContent className="space-y-4" value="basic">
                  <div className="grid gap-6">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.name")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                {...field}
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name="language"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("form.language")}
                              <span className="ml-1 text-[0.8rem] text-muted-foreground">
                                {t("form.languageDescription")}
                              </span>
                            </FormLabel>
                            <FormControl>
                              <EnhancedInput
                                {...field}
                                onValueChange={(v) =>
                                  form.setValue(field.name, v as string)
                                }
                                placeholder={t("form.languagePlaceholder")}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                      <FormField
                        control={form.control}
                        name="traffic"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.traffic")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                placeholder={t("form.noLimit")}
                                type="number"
                                {...field}
                                formatInput={(value) =>
                                  unitConversion("bytesToGb", value)
                                }
                                formatOutput={(value) =>
                                  unitConversion("gbToBytes", value)
                                }
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                                suffix="GB"
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="speed_limit"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.speedLimit")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                placeholder={t("form.noLimit")}
                                type="number"
                                {...field}
                                formatInput={(value) =>
                                  unitConversion("bitsToMb", value)
                                }
                                formatOutput={(value) =>
                                  unitConversion("mbToBits", value)
                                }
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                                suffix="Mbps"
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="device_limit"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.deviceLimit")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                placeholder={t("form.noLimit")}
                                step={1}
                                type="number"
                                {...field}
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <FormField
                        control={form.control}
                        name="inventory"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.inventory")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                {...field}
                                formatInput={(value) =>
                                  value === -1 ? "" : value
                                }
                                formatOutput={(value) =>
                                  value === "" || value === undefined
                                    ? -1
                                    : Number(value)
                                }
                                min={-1}
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                                placeholder={t("form.unlimitedInventory")}
                                step={1}
                                type="number"
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="quota"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.quota")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                placeholder={t("form.noLimit")}
                                step={1}
                                type="number"
                                {...field}
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    <FormField
                      control={form.control}
                      name="description"
                      render={({ field }) => (
                        <FormItem>
                          <FormControl>
                            <ProductDescriptionEditor
                              onChange={(value) =>
                                form.setValue(field.name, value, {
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
                </TabsContent>

                <TabsContent className="space-y-4" value="pricing">
                  <div className="grid gap-6">
                    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
                      <FormField
                        control={form.control}
                        name="unit_price"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.unitPrice")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                type="number"
                                {...field}
                                formatInput={(value) =>
                                  unitConversion("centsToDollars", value)
                                }
                                formatOutput={(value) =>
                                  unitConversion("dollarsToCents", value)
                                }
                                min={0}
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="unit_time"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.unitTime")}</FormLabel>
                            <FormControl>
                              <Combobox
                                placeholder={t("form.selectUnitTime")}
                                {...field}
                                onChange={(value) => {
                                  if (value) {
                                    form.setValue(field.name, value);
                                  }
                                }}
                                options={[
                                  {
                                    label: t("form.NoLimit"),
                                    value: "NoLimit",
                                  },
                                  { label: t("form.Year"), value: "Year" },
                                  { label: t("form.Month"), value: "Month" },
                                  { label: t("form.Day"), value: "Day" },
                                  { label: t("form.Hour"), value: "Hour" },
                                  { label: t("form.Minute"), value: "Minute" },
                                ]}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="replacement"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.replacement")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                type="number"
                                {...field}
                                formatInput={(value) =>
                                  unitConversion("centsToDollars", value)
                                }
                                formatOutput={(value) =>
                                  unitConversion("dollarsToCents", value)
                                }
                                min={0}
                                onValueChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name="reset_cycle"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("form.resetCycle")}</FormLabel>
                            <FormControl>
                              <Combobox<number, false>
                                placeholder={t("form.selectResetCycle")}
                                {...field}
                                onChange={(value) => {
                                  if (typeof value === "number") {
                                    form.setValue(field.name, value);
                                  }
                                }}
                                options={[
                                  { label: t("form.noReset"), value: 0 },
                                  { label: t("form.resetOn1st"), value: 1 },
                                  { label: t("form.monthlyReset"), value: 2 },
                                  { label: t("form.annualReset"), value: 3 },
                                ]}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>
                    <FormField
                      control={form.control}
                      name="discount"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("form.discount")}</FormLabel>
                          <FormControl>
                            <ArrayInput<
                              API.SubscribeDiscount & { price?: number }
                            >
                              fields={[
                                {
                                  name: "quantity",
                                  type: "number",
                                  step: 1,
                                  min: 1,
                                  suffix: unit_time && t(`form.${unit_time}`),
                                },
                                {
                                  name: "discount",
                                  type: "number",
                                  min: 1,
                                  max: 100,
                                  step: 0.01,
                                  placeholder: t("form.discountPercent"),
                                  suffix: "%",
                                },
                                {
                                  name: "price",
                                  placeholder: t("form.discount_price"),
                                  type: "number",
                                  min: 0,
                                  step: 0.01,
                                  prefix: currency.currency_symbol,
                                  formatInput: (value) =>
                                    unitConversion("centsToDollars", value),
                                  formatOutput: (value) =>
                                    String(
                                      unitConversion("dollarsToCents", value)
                                    ),
                                },
                              ]}
                              onChange={(newValues: DiscountFormItem[]) => {
                                const oldValues: DiscountFormItem[] =
                                  field.value || [];
                                let lastChangedField: string | undefined;
                                let changedIndex: number | undefined;

                                for (
                                  let i = 0;
                                  i <
                                  Math.max(newValues.length, oldValues.length);
                                  i++
                                ) {
                                  const newItem = newValues[i];
                                  const oldItem = oldValues[i];

                                  if (newItem?.quantity !== oldItem?.quantity) {
                                    lastChangedField = "quantity";
                                    changedIndex = i;
                                    break;
                                  }
                                  if (newItem?.discount !== oldItem?.discount) {
                                    lastChangedField = "discount";
                                    changedIndex = i;
                                    break;
                                  }
                                  if (newItem?.price !== oldItem?.price) {
                                    lastChangedField = "price";
                                    changedIndex = i;
                                    break;
                                  }
                                }
                                form.setValue(field.name, newValues, {
                                  shouldDirty: true,
                                });
                                if (newValues?.length > 0) {
                                  debouncedCalculateDiscount(
                                    newValues,
                                    field.name,
                                    lastChangedField,
                                    changedIndex
                                  );
                                }
                              }}
                              value={field.value}
                            />
                          </FormControl>
                          <FormDescription>
                            {t("form.discountDescription")}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="deduction_ratio"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("form.deductionRatio")}</FormLabel>
                          <FormControl>
                            <EnhancedInput
                              type="number"
                              {...field}
                              max={100}
                              min={0}
                              onValueChange={(value) => {
                                form.setValue(field.name, value);
                              }}
                              placeholder={t("form.auto", "Auto")}
                              step={1}
                              suffix="%"
                            />
                          </FormControl>
                          <FormMessage />
                          <FormDescription>
                            {t("form.deductionRatioDescription")}
                          </FormDescription>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="allow_renewal"
                      render={({ field }) => (
                        <FormItem>
                          <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                              <FormLabel>
                                {t("form.allowRenewal", "允许续费")}
                              </FormLabel>
                              <FormDescription>
                                {t(
                                  "form.allowRenewalDescription",
                                  "用户可以对该商品的订阅进行续费"
                                )}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                checked={field.value !== false}
                                onCheckedChange={(value) =>
                                  form.setValue(field.name, value)
                                }
                              />
                            </FormControl>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="allow_reset_traffic"
                      render={({ field }) => (
                        <FormItem>
                          <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                              <FormLabel>
                                {t("form.allowResetTraffic", "允许重置流量")}
                              </FormLabel>
                              <FormDescription>
                                {t(
                                  "form.allowResetTrafficDescription",
                                  "用户可以购买该商品的流量重置"
                                )}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                checked={field.value !== false}
                                onCheckedChange={(value) =>
                                  form.setValue(field.name, value)
                                }
                              />
                            </FormControl>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="renewal_reset"
                      render={({ field }) => (
                        <FormItem>
                          <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                              <FormLabel>{t("form.renewalReset")}</FormLabel>
                              <FormDescription>
                                {t("form.renewalResetDescription")}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="show_original_price"
                      render={({ field }) => (
                        <FormItem>
                          <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                              <FormLabel>
                                {t(
                                  "form.showOriginalPrice",
                                  "Show Original Price"
                                )}
                              </FormLabel>
                              <FormDescription>
                                {t(
                                  "form.showOriginalPriceDescription",
                                  "Display original price in the storefront"
                                )}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                checked={!!field.value}
                                onCheckedChange={(value) =>
                                  form.setValue(field.name, value)
                                }
                              />
                            </FormControl>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="allow_deduction"
                      render={({ field }) => (
                        <FormItem>
                          <div className="flex items-center justify-between">
                            <div className="space-y-0.5">
                              <FormLabel>
                                {t("form.purchaseWithDiscount")}
                              </FormLabel>
                              <FormDescription>
                                {t("form.purchaseWithDiscountDescription")}
                              </FormDescription>
                            </div>
                            <FormControl>
                              <Switch
                                checked={!!field.value}
                                onCheckedChange={(value) => {
                                  form.setValue(field.name, value);
                                }}
                              />
                            </FormControl>
                          </div>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </TabsContent>

                <TabsContent className="space-y-6" value="storefront">
                  <div className="overflow-hidden rounded-lg border">
                    <FormField
                      control={form.control}
                      name="show"
                      render={({ field }) => (
                        <FormItem className="flex items-center justify-between gap-4 border-b p-4">
                          <div className="space-y-1">
                            <FormLabel>
                              {t("guestVisible", "访客可见")}
                            </FormLabel>
                            <FormDescription>
                              {t(
                                "guestVisibleDescription",
                                "允许未登录访客在公开商品页查看"
                              )}
                            </FormDescription>
                          </div>
                          <FormControl>
                            <Switch
                              checked={!!field.value}
                              onCheckedChange={field.onChange}
                            />
                          </FormControl>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="user_visible"
                      render={({ field }) => (
                        <FormItem className="flex items-center justify-between gap-4 border-b p-4">
                          <div className="space-y-1">
                            <FormLabel>
                              {t("userVisible", "用户可见")}
                            </FormLabel>
                            <FormDescription>
                              {t(
                                "userVisibleDescription",
                                "允许登录用户查看，关闭销售后仍可作为预告展示"
                              )}
                            </FormDescription>
                          </div>
                          <FormControl>
                            <Switch
                              checked={!!field.value}
                              onCheckedChange={field.onChange}
                            />
                          </FormControl>
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="sell"
                      render={({ field }) => (
                        <FormItem className="flex items-center justify-between gap-4 p-4">
                          <div className="space-y-1">
                            <FormLabel>{t("sellable", "允许新购")}</FormLabel>
                            <FormDescription>
                              {t(
                                "sellableDescription",
                                "新订单还会同时校验销售时间、库存和购买数量"
                              )}
                            </FormDescription>
                          </div>
                          <FormControl>
                            <Switch
                              checked={!!field.value}
                              onCheckedChange={field.onChange}
                            />
                          </FormControl>
                        </FormItem>
                      )}
                    />
                  </div>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <FormField
                      control={form.control}
                      name="display_template"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.displayTemplate", "商品卡片模板")}
                          </FormLabel>
                          <Select
                            onValueChange={field.onChange}
                            value={field.value || "standard"}
                          >
                            <FormControl>
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                              <SelectItem value="standard">
                                {t("form.templateStandard", "标准")}
                              </SelectItem>
                              <SelectItem value="popular">
                                {t("form.templatePopular", "最受欢迎")}
                              </SelectItem>
                              <SelectItem value="premium">
                                {t("form.templatePremium", "高级")}
                              </SelectItem>
                              <SelectItem value="compact">
                                {t("form.templateCompact", "紧凑")}
                              </SelectItem>
                            </SelectContent>
                          </Select>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="badge"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("form.badge", "自定义角标")}</FormLabel>
                          <FormControl>
                            <Input
                              {...field}
                              maxLength={64}
                              placeholder={t(
                                "form.badgePlaceholder",
                                "例如：推荐、新品、限时"
                              )}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <FormField
                      control={form.control}
                      name="min_quantity"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.minQuantity", "最少购买周期")}
                          </FormLabel>
                          <FormControl>
                            <EnhancedInput
                              {...field}
                              min={1}
                              max={1000}
                              onValueChange={(value) =>
                                form.setValue(field.name, Number(value) || 1)
                              }
                              step={1}
                              type="number"
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="max_quantity"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.maxQuantity", "最多购买周期")}
                          </FormLabel>
                          <FormControl>
                            <EnhancedInput
                              {...field}
                              min={0}
                              max={1000}
                              onValueChange={(value) =>
                                form.setValue(field.name, Number(value) || 0)
                              }
                              placeholder={t(
                                "form.systemLimit",
                                "0 表示使用系统上限"
                              )}
                              step={1}
                              type="number"
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <FormField
                      control={form.control}
                      name="sale_starts_at"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.saleStartsAt", "开售时间")}
                          </FormLabel>
                          <FormControl>
                            <Input
                              onChange={(event) =>
                                field.onChange(
                                  localInputToTimestamp(event.target.value)
                                )
                              }
                              type="datetime-local"
                              value={timestampToLocalInput(field.value)}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="sale_ends_at"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t("form.saleEndsAt", "停售时间")}
                          </FormLabel>
                          <FormControl>
                            <Input
                              onChange={(event) =>
                                field.onChange(
                                  localInputToTimestamp(event.target.value)
                                )
                              }
                              type="datetime-local"
                              value={timestampToLocalInput(field.value)}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <Alert>
                    <AlertTitle>
                      {t("form.renewalPricing", "续费计价规则")}
                    </AlertTitle>
                    <AlertDescription>
                      {t(
                        "form.renewalCurrentPriceNotice",
                        "下架或停售只影响新购；允许续费时，续费订单始终按商品当前价格、折扣和购买周期规则结算。"
                      )}
                    </AlertDescription>
                  </Alert>
                </TabsContent>

                <TabsContent className="space-y-4" value="servers">
                  <div className="space-y-6">
                    <FormField
                      control={form.control}
                      name="node_tags"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("form.nodeGroup")}</FormLabel>
                          <FormControl>
                            <Accordion
                              className="w-full"
                              collapsible
                              type="single"
                            >
                              {tagGroups.map((tag) => {
                                const value = field.value || [];
                                const tagId = tag;
                                const nodesWithTag = getNodesByTag(tag);

                                return (
                                  <AccordionItem key={tag} value={String(tag)}>
                                    <AccordionTrigger>
                                      <div className="flex items-center gap-2">
                                        <Checkbox
                                          checked={value.includes(tagId)}
                                          onCheckedChange={(checked) =>
                                            checked
                                              ? form.setValue(field.name, [
                                                  ...value,
                                                  tagId,
                                                ])
                                              : form.setValue(
                                                  field.name,
                                                  value.filter(
                                                    (v) => v !== tagId
                                                  )
                                                )
                                          }
                                        />
                                        <Label>
                                          {tag}
                                          <span className="ml-2 text-muted-foreground text-xs">
                                            ({nodesWithTag.length})
                                          </span>
                                        </Label>
                                      </div>
                                    </AccordionTrigger>
                                    <AccordionContent>
                                      <ul className="space-y-1">
                                        {getNodesByTag(tag).map((node) => (
                                          <li
                                            className="flex items-center justify-between gap-3"
                                            key={node.id}
                                          >
                                            <span className="flex-1">
                                              {node.name}
                                            </span>
                                            <span className="flex-1">
                                              {node.address}:{node.port}
                                            </span>
                                            <span className="flex-1 text-right">
                                              {node.inbound_alias ||
                                                node.protocol}
                                            </span>
                                          </li>
                                        ))}
                                      </ul>
                                    </AccordionContent>
                                  </AccordionItem>
                                );
                              })}
                            </Accordion>
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="nodes"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t("form.node")}</FormLabel>
                          <FormControl>
                            <div className="flex flex-col gap-2">
                              {getNodesWithoutTags().map((item) => {
                                const value = field.value || [];

                                return (
                                  <div
                                    className="flex items-center gap-2"
                                    key={item.id}
                                  >
                                    <Checkbox
                                      checked={value.includes(item.id!)}
                                      onCheckedChange={(checked) =>
                                        checked
                                          ? form.setValue(field.name, [
                                              ...value,
                                              item.id,
                                            ])
                                          : form.setValue(
                                              field.name,
                                              value.filter(
                                                (value: number) =>
                                                  value !== item.id
                                              )
                                            )
                                      }
                                    />
                                    <Label className="flex w-full items-center justify-between gap-3">
                                      <span className="flex-1">
                                        {item.name}
                                      </span>
                                      <span className="flex-1">
                                        {item.address}:{item.port}
                                      </span>
                                      <span className="flex-1 text-right">
                                        {item.inbound_alias || item.protocol}
                                      </span>
                                    </Label>
                                  </div>
                                );
                              })}
                            </div>
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                </TabsContent>
              </Tabs>
            </form>
          </Form>
        </ScrollArea>
        <SheetFooter className="flex-row justify-end gap-2 pt-3">
          <Button
            disabled={loading || loadingInitialValues}
            onClick={() => {
              setOpen(false);
            }}
            variant="outline"
          >
            {t("form.cancel")}
          </Button>
          <Button
            disabled={loading || loadingInitialValues}
            onClick={form.handleSubmit(handleSubmit, (errors) => {
              const keys = Object.keys(errors) as Array<
                keyof SubscribeFormValues
              >;
              for (const key of keys) {
                const formattedKey = key.replace(/_([a-z])/g, (_, letter) =>
                  letter.toUpperCase()
                );
                const error = errors[key];
                toast.error(
                  `${t(`form.${formattedKey}`)} is ${error?.message}`
                );
                return false;
              }
            })}
          >
            {(loading || loadingInitialValues) && (
              <Icon className="mr-2 animate-spin" icon="mdi:loading" />
            )}
            {t("form.confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
