import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@workspace/ui/components/button";
import {
  Field,
  FieldContent,
  FieldLabel,
  FieldTitle,
} from "@workspace/ui/components/field";
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
  RadioGroup,
  RadioGroupItem,
} from "@workspace/ui/components/radio-group";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
import { Combobox } from "@workspace/ui/composed/combobox";
import { DatePicker } from "@workspace/ui/composed/date-picker";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import { unitConversion } from "@workspace/ui/utils/unit-conversions";
import {
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  useEffect,
  useId,
  useMemo,
  useState,
} from "react";
import {
  type FieldPath,
  type FieldPathValue,
  type FieldErrors,
  type UseFormReturn,
  useForm,
  useWatch,
} from "react-hook-form";
import { useTranslation } from "react-i18next";
import { useSubscribe } from "@/stores/subscribe";
import { DiscardChangesDialog } from "../../components/discard-changes-dialog";
import {
  couponErrorTab,
  couponToFormValues,
  createCouponFormSchema,
  type CouponFormTab,
  type CouponFormValues,
} from "./coupon-form-data";

interface CouponFormProps {
  onSubmit: (data: CouponFormValues) => Promise<boolean> | boolean;
  initialValues?: Partial<API.Coupon>;
  loadInitialValues?: () =>
    | Promise<Partial<API.Coupon> | undefined>
    | Partial<API.Coupon>
    | undefined;
  loading?: boolean;
  trigger: ReactNode;
  title: string;
}

function setFieldValue<K extends FieldPath<CouponFormValues>>(
  form: UseFormReturn<CouponFormValues>,
  field: K,
  value: FieldPathValue<CouponFormValues, K>
) {
  form.setValue(field, value, {
    shouldDirty: true,
    shouldTouch: true,
    shouldValidate: true,
  });
}

export default function CouponForm({
  onSubmit,
  initialValues,
  loadInitialValues,
  loading,
  trigger,
  title,
}: CouponFormProps) {
  const { t } = useTranslation("coupon");
  const { subscribes } = useSubscribe();
  const formId = useId();
  const percentageTypeId = `${formId}-percentage`;
  const amountTypeId = `${formId}-amount`;
  const [open, setOpen] = useState(false);
  const [opening, setOpening] = useState(false);
  const [discardOpen, setDiscardOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<CouponFormTab>("discount");
  const schema = useMemo(() => createCouponFormSchema(t), [t]);
  const form = useForm<CouponFormValues>({
    resolver: zodResolver(schema),
    defaultValues: couponToFormValues(initialValues),
    mode: "onChange",
  });
  const type = useWatch({ control: form.control, name: "type" });
  const usedCount = useWatch({ control: form.control, name: "used_count" });

  useEffect(() => {
    if (!(open || form.formState.isDirty)) {
      form.reset(couponToFormValues(initialValues));
    }
  }, [form, initialValues, open]);

  async function handleSubmit(data: CouponFormValues) {
    const success = await onSubmit(data);
    if (success) {
      form.reset(data);
      setOpen(false);
    }
  }

  function handleInvalid(errors: FieldErrors<CouponFormValues>) {
    setActiveTab(couponErrorTab(Object.keys(errors)));
  }

  async function handleOpen() {
    setOpen(true);
    setOpening(true);
    setActiveTab("discount");
    try {
      const values = loadInitialValues
        ? await loadInitialValues()
        : initialValues;
      if (loadInitialValues && !values) {
        setOpen(false);
        return;
      }
      form.reset(couponToFormValues(values ?? initialValues));
    } catch {
      setOpen(false);
    } finally {
      setOpening(false);
    }
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen && (loading || opening)) return;
    if (!nextOpen && form.formState.isDirty) {
      setDiscardOpen(true);
      return;
    }
    setOpen(nextOpen);
  }

  function renderTrigger() {
    if (isValidElement(trigger)) {
      const element = trigger as ReactElement<{
        disabled?: boolean;
        onClick?: () => void;
      }>;
      return cloneElement(element, {
        disabled: opening || element.props.disabled,
        onClick: handleOpen,
      });
    }
    return (
      <Button disabled={opening} onClick={handleOpen} type="button">
        {opening ? <Icon className="animate-spin" icon="mdi:loading" /> : null}
        {trigger}
      </Button>
    );
  }

  return (
    <>
      <Sheet onOpenChange={handleOpenChange} open={open}>
        {renderTrigger()}
        <SheetContent className="coupon-form-sheet w-[680px] max-w-full md:max-w-3xl">
          <SheetHeader>
            <SheetTitle>{title}</SheetTitle>
            <SheetDescription>
              {t(
                "form.description",
                "Configure discount value, product scope, validity period, and usage limits."
              )}
            </SheetDescription>
          </SheetHeader>
          <ScrollArea className="min-h-0 flex-1 px-6">
            {opening ? (
              <div className="flex h-56 items-center justify-center text-muted-foreground text-sm">
                <Icon className="mr-2 animate-spin" icon="mdi:loading" />
                {t("form.loading", "Loading coupon details")}
              </div>
            ) : (
              <Form {...form}>
                <form
                  className="space-y-5 py-5"
                  id={formId}
                  onSubmit={form.handleSubmit(handleSubmit, handleInvalid)}
                >
                  <Tabs
                    onValueChange={(value) =>
                      setActiveTab(value as CouponFormTab)
                    }
                    value={activeTab}
                  >
                    <TabsList className="grid w-full grid-cols-2">
                      <TabsTrigger value="discount">
                        {t("form.tabs.discount", "Discount & scope")}
                      </TabsTrigger>
                      <TabsTrigger value="limits">
                        {t("form.tabs.limits", "Validity & limits")}
                      </TabsTrigger>
                    </TabsList>

                    <TabsContent className="space-y-5 pt-5" value="discount">
                      <section className="coupon-form-section">
                        <header>
                          <h3 className="font-semibold text-sm">
                            {t("form.identity", "Coupon identity")}
                          </h3>
                          <p className="text-muted-foreground text-xs leading-5">
                            {t(
                              "form.identityDescription",
                              "Use a recognizable internal name and an optional customer-facing code."
                            )}
                          </p>
                        </header>
                        <div className="grid gap-4 md:grid-cols-2">
                          <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>{t("form.name", "Name")}</FormLabel>
                                <FormControl>
                                  <EnhancedInput
                                    onValueChange={field.onChange}
                                    placeholder={t(
                                      "form.enterCouponName",
                                      "Enter Coupon Name"
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
                            name="code"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  {t("form.customCouponCode", "Coupon Code")}
                                </FormLabel>
                                <FormControl>
                                  <EnhancedInput
                                    onValueChange={field.onChange}
                                    placeholder={t(
                                      "form.customCouponCodePlaceholder",
                                      "Leave blank to generate automatically"
                                    )}
                                    value={field.value}
                                  />
                                </FormControl>
                                <FormDescription>
                                  {t(
                                    "form.codeDescription",
                                    "3-64 letters, numbers, underscores, or hyphens."
                                  )}
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </section>

                      <section className="coupon-form-section">
                        <header>
                          <h3 className="font-semibold text-sm">
                            {t("form.discountRule", "Discount rule")}
                          </h3>
                          <p className="text-muted-foreground text-xs leading-5">
                            {t(
                              "form.discountRuleDescription",
                              "Choose a percentage or fixed amount deducted from the order subtotal."
                            )}
                          </p>
                        </header>
                        <FormField
                          control={form.control}
                          name="type"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t("form.type", "Coupon Type")}
                              </FormLabel>
                              <FormControl>
                                <RadioGroup
                                  className="grid grid-cols-2 gap-2"
                                  onValueChange={(value) => {
                                    const nextType = Number(value) as 1 | 2;
                                    setFieldValue(form, "type", nextType);
                                    setFieldValue(
                                      form,
                                      "discount",
                                      nextType === 1 ? 10 : 100
                                    );
                                  }}
                                  value={String(field.value)}
                                >
                                  <FieldLabel htmlFor={percentageTypeId}>
                                    <Field orientation="horizontal">
                                      <RadioGroupItem
                                        id={percentageTypeId}
                                        value="1"
                                      />
                                      <FieldContent>
                                        <FieldTitle>
                                          {t(
                                            "form.percentageDiscount",
                                            "Percentage"
                                          )}
                                        </FieldTitle>
                                      </FieldContent>
                                    </Field>
                                  </FieldLabel>
                                  <FieldLabel htmlFor={amountTypeId}>
                                    <Field orientation="horizontal">
                                      <RadioGroupItem
                                        id={amountTypeId}
                                        value="2"
                                      />
                                      <FieldContent>
                                        <FieldTitle>
                                          {t(
                                            "form.amountDiscount",
                                            "Fixed amount"
                                          )}
                                        </FieldTitle>
                                      </FieldContent>
                                    </Field>
                                  </FieldLabel>
                                </RadioGroup>
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                        <FormField
                          control={form.control}
                          name="discount"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {type === 1
                                  ? t(
                                      "form.percentageDiscount",
                                      "Percentage Discount"
                                    )
                                  : t("form.amountDiscount", "Amount Discount")}
                              </FormLabel>
                              <FormControl>
                                <EnhancedInput
                                  formatInput={
                                    type === 2
                                      ? (value) =>
                                          unitConversion(
                                            "centsToDollars",
                                            value
                                          )
                                      : undefined
                                  }
                                  formatOutput={
                                    type === 2
                                      ? (value) =>
                                          unitConversion(
                                            "dollarsToCents",
                                            value
                                          )
                                      : undefined
                                  }
                                  max={type === 1 ? 100 : undefined}
                                  min={1}
                                  onValueChange={field.onChange}
                                  suffix={type === 1 ? "%" : undefined}
                                  type="number"
                                  value={field.value}
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </section>

                      <section className="coupon-form-section">
                        <FormField
                          control={form.control}
                          name="subscribe"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>
                                {t(
                                  "form.specifiedServer",
                                  "Applicable subscriptions"
                                )}
                              </FormLabel>
                              <FormControl>
                                <Combobox<number, true>
                                  multiple
                                  onChange={(value) =>
                                    setFieldValue(
                                      form,
                                      "subscribe",
                                      value || []
                                    )
                                  }
                                  options={subscribes?.map((item) => ({
                                    value: item.id!,
                                    label: item.name!,
                                  }))}
                                  placeholder={t(
                                    "form.selectServer",
                                    "All subscription products"
                                  )}
                                  value={field.value}
                                />
                              </FormControl>
                              <FormDescription>
                                {t(
                                  "form.subscribeDescription",
                                  "Leave empty to allow every subscription product."
                                )}
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </section>
                    </TabsContent>

                    <TabsContent className="space-y-5 pt-5" value="limits">
                      <section className="coupon-form-section">
                        <header>
                          <h3 className="font-semibold text-sm">
                            {t("form.validity", "Validity period")}
                          </h3>
                          <p className="text-muted-foreground text-xs leading-5">
                            {t(
                              "form.validityDescription",
                              "Coupons are accepted from the start time until, but not including, the expiry time."
                            )}
                          </p>
                        </header>
                        <div className="grid gap-4 md:grid-cols-2">
                          <FormField
                            control={form.control}
                            name="start_time"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  {t("form.startTime", "Start Time")}
                                </FormLabel>
                                <FormControl>
                                  <DatePicker
                                    onChange={(value) =>
                                      setFieldValue(
                                        form,
                                        "start_time",
                                        value || 0
                                      )
                                    }
                                    value={field.value}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                          <FormField
                            control={form.control}
                            name="expire_time"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  {t("form.expireTime", "Expire Time")}
                                </FormLabel>
                                <FormControl>
                                  <DatePicker
                                    onChange={(value) =>
                                      setFieldValue(
                                        form,
                                        "expire_time",
                                        value || 0
                                      )
                                    }
                                    value={field.value}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </section>

                      <section className="coupon-form-section">
                        <header>
                          <h3 className="font-semibold text-sm">
                            {t("form.usageLimits", "Usage limits")}
                          </h3>
                          <p className="text-muted-foreground text-xs leading-5">
                            {t(
                              "form.usageLimitsDescription",
                              "Use zero for unlimited total or per-user redemptions."
                            )}
                          </p>
                        </header>
                        <div className="grid gap-4 md:grid-cols-2">
                          <FormField
                            control={form.control}
                            name="count"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  {t("form.count", "Total usage limit")}
                                </FormLabel>
                                <FormControl>
                                  <EnhancedInput
                                    min={0}
                                    onValueChange={field.onChange}
                                    step={1}
                                    type="number"
                                    value={field.value}
                                  />
                                </FormControl>
                                <FormDescription>
                                  {t("form.usedCount", "Already used")}:{" "}
                                  {usedCount || 0}
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                          <FormField
                            control={form.control}
                            name="user_limit"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>
                                  {t("form.userLimit", "Per-user usage limit")}
                                </FormLabel>
                                <FormControl>
                                  <EnhancedInput
                                    min={0}
                                    onValueChange={field.onChange}
                                    step={1}
                                    type="number"
                                    value={field.value}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </section>
                    </TabsContent>
                  </Tabs>
                </form>
              </Form>
            )}
          </ScrollArea>
          <SheetFooter className="coupon-form-footer flex-row justify-between gap-2">
            <span className="hidden text-muted-foreground text-xs sm:block">
              {form.formState.isDirty
                ? t("form.unsavedChanges", "Unsaved changes")
                : t("form.noChanges", "No unsaved changes")}
            </span>
            <div className="flex gap-2">
              <Button
                disabled={loading || opening}
                onClick={() => handleOpenChange(false)}
                type="button"
                variant="outline"
              >
                {t("form.cancel", "Cancel")}
              </Button>
              <Button
                disabled={loading || opening || !form.formState.isDirty}
                form={formId}
                type="submit"
              >
                {loading ? (
                  <Icon className="animate-spin" icon="mdi:loading" />
                ) : null}
                {t("form.confirm", "Save coupon")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <DiscardChangesDialog
        onDiscard={() => {
          form.reset(couponToFormValues(initialValues));
          setDiscardOpen(false);
          setOpen(false);
        }}
        onOpenChange={setDiscardOpen}
        open={discardOpen}
      />
    </>
  );
}
