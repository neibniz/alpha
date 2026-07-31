"use client";

import { useQuery } from "@tanstack/react-query";
import { Badge } from "@workspace/ui/components/badge";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@workspace/ui/components/card";
import { Input } from "@workspace/ui/components/input";
import { Label } from "@workspace/ui/components/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Separator } from "@workspace/ui/components/separator";
import { Skeleton } from "@workspace/ui/components/skeleton";
import { Switch } from "@workspace/ui/components/switch";
import { Textarea } from "@workspace/ui/components/textarea";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import {
  queryMembershipPlan,
  updateMembershipPlan,
} from "@workspace/ui/services/admin/membership";
import { unitConversion } from "@workspace/ui/utils/unit-conversions";
import { CircleAlert, RefreshCw, Save } from "lucide-react";
import { useEffect, useState, useTransition } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { AdminPage, AdminPageHeader } from "@/components/admin-page";
import { Display } from "@/components/display";
import { formatDate } from "@/utils/common";
import { DiscardChangesDialog } from "../../components/discard-changes-dialog";
import {
  buildMembershipPlanPayload,
  membershipPlanFormKey,
  membershipPlanToForm,
  type MembershipPlanErrors,
  validateMembershipPlan,
} from "./membership-plan-data";

type DurationUnit = "Year" | "Month" | "Day" | "Hour" | "Minute";

const durationUnits: DurationUnit[] = [
  "Year",
  "Month",
  "Day",
  "Hour",
  "Minute",
];

export default function Membership() {
  const { t } = useTranslation("membership");
  const [form, setForm] = useState<API.UpdateMembershipPlanRequest>(() =>
    membershipPlanToForm()
  );
  const [baseline, setBaseline] = useState<API.UpdateMembershipPlanRequest>(
    () => membershipPlanToForm()
  );
  const [errors, setErrors] = useState<MembershipPlanErrors>({});
  const [discardOpen, setDiscardOpen] = useState(false);
  const [loading, startTransition] = useTransition();

  const { data, isError, isFetching, isLoading, refetch } = useQuery({
    queryKey: ["queryMembershipPlan"],
    queryFn: async () => {
      const { data } = await queryMembershipPlan();
      return data.data;
    },
  });

  const dirty = membershipPlanFormKey(form) !== membershipPlanFormKey(baseline);
  const errorText = (error?: string) => (error ? t(error, error) : "");

  useEffect(() => {
    if (!data || dirty) return;
    const next = membershipPlanToForm(data);
    setForm(next);
    setBaseline(next);
    setErrors({});
  }, [data, dirty]);

  const updateForm = <K extends keyof API.UpdateMembershipPlanRequest>(
    key: K,
    value: API.UpdateMembershipPlanRequest[K]
  ) => {
    setErrors((current) => ({ ...current, [key]: undefined }));
    setForm((current) => ({
      ...current,
      [key]: value,
    }));
  };

  const updatePolicy = <K extends keyof API.MembershipPolicy>(
    key: K,
    value: API.MembershipPolicy[K]
  ) => {
    const errorKey =
      key === "monthly_consumption_limit_enabled"
        ? "monthly_consumption_limit"
        : key === "monthly_order_limit_enabled"
          ? "monthly_order_limit"
          : key === "subscription_expiry_limit_enabled"
            ? "subscription_expiry_grace_days"
            : key;
    setErrors((current) => ({ ...current, [errorKey]: undefined }));
    setForm((current) => ({
      ...current,
      policy: {
        ...current.policy,
        [key]: value,
      },
    }));
  };

  const handleSubmit = () => {
    const nextErrors = validateMembershipPlan(form);
    setErrors(nextErrors);
    const firstError = Object.values(nextErrors).find(Boolean);
    if (firstError) {
      toast.error(errorText(firstError));
      return;
    }
    startTransition(async () => {
      try {
        const payload = buildMembershipPlanPayload(form);
        const response = await updateMembershipPlan(payload);
        const saved = membershipPlanToForm(response.data.data || payload);
        setForm(saved);
        setBaseline(saved);
        setErrors({});
        toast.success(t("updateSuccess", "Updated successfully"));
        refetch().catch(() => undefined);
      } catch {
        toast.error(t("updateFailed", "Update failed"));
      }
    });
  };

  const durationLabel = (unit?: string) =>
    t(`duration.${unit || "Year"}`, unit || "Year");

  const disabled = loading || !form.id || !dirty;

  const refreshPlan = async () => {
    const result = await refetch();
    if (result.data) {
      const next = membershipPlanToForm(result.data);
      setForm(next);
      setBaseline(next);
      setErrors({});
    }
  };

  if (isLoading && !data) {
    return (
      <div className="grid gap-4">
        <Skeleton className="h-16 rounded-md" />
        <Skeleton className="h-[520px] rounded-md" />
      </div>
    );
  }

  return (
    <>
      <AdminPage className="gap-6">
        <AdminPageHeader
          actions={
            <div className="flex gap-2">
              {dirty ? (
                <Badge className="self-center" variant="outline">
                  {t("unsavedChanges", "Unsaved changes")}
                </Badge>
              ) : null}
              <Button
                disabled={isFetching}
                onClick={() => {
                  if (dirty) {
                    setDiscardOpen(true);
                    return;
                  }
                  refreshPlan().catch(() =>
                    toast.error(
                      t("loadFailed", "Unable to load membership plan")
                    )
                  );
                }}
                type="button"
                variant="outline"
              >
                <RefreshCw className="size-4" />
                {t("refresh", "Refresh")}
              </Button>
              <Button disabled={disabled} onClick={handleSubmit} type="button">
                <Save className="size-4" />
                {t("save", "Save")}
              </Button>
            </div>
          }
          description={t(
            "description",
            "Control whether membership is required, then configure the membership card and its policies."
          )}
          eyebrow={t("workspace", "Commerce")}
          title={t("title", "Membership Management")}
        />

        {isError ? (
          <Alert variant="destructive">
            <CircleAlert />
            <AlertTitle>
              {t("loadFailed", "Unable to load membership plan")}
            </AlertTitle>
            <AlertDescription>
              {t(
                "loadFailedDescription",
                "The current membership settings could not be loaded."
              )}
            </AlertDescription>
            <Button
              className="ml-auto"
              onClick={() => {
                refreshPlan().catch(() =>
                  toast.error(t("loadFailed", "Unable to load membership plan"))
                );
              }}
              size="sm"
              variant="outline"
            >
              {t("retry", "Retry")}
            </Button>
          </Alert>
        ) : null}

        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Card>
            <CardHeader>
              <CardTitle>{t("cardSettings", "Card Settings")}</CardTitle>
              <CardDescription>
                {t(
                  "cardSettingsDescription",
                  "This switch controls the entire membership requirement across subscription purchases and renewals."
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className="grid gap-6">
              <div className="border-y py-4">
                <div className="flex items-center justify-between gap-4">
                  <div className="flex flex-col gap-1">
                    <Label>{t("enabled", "Enable membership")}</Label>
                    <p className="text-muted-foreground text-sm">
                      {t(
                        "enabledDescription",
                        "When disabled, users can purchase and renew subscriptions without membership checks, and all membership policies are paused."
                      )}
                    </p>
                  </div>
                  <Switch
                    aria-label={t("enabled", "Enabled")}
                    checked={form.enabled}
                    onCheckedChange={(checked) =>
                      updateForm("enabled", checked)
                    }
                  />
                </div>
              </div>

              {!form.enabled ? (
                <Alert>
                  <CircleAlert />
                  <AlertTitle>
                    {t("disabledTitle", "Membership checks are disabled")}
                  </AlertTitle>
                  <AlertDescription>
                    {t(
                      "disabledDescription",
                      "After saving, membership expiry is no longer checked for subscription purchases or renewals. Existing membership expiry dates and orders are retained."
                    )}
                  </AlertDescription>
                </Alert>
              ) : null}

              <div className="grid gap-4 md:grid-cols-2">
                <div className="grid gap-2">
                  <Label htmlFor="membership-name">{t("name", "Name")}</Label>
                  <Input
                    aria-invalid={Boolean(errors.name)}
                    id="membership-name"
                    onChange={(event) => updateForm("name", event.target.value)}
                    placeholder={t("namePlaceholder", "Annual Membership")}
                    value={form.name}
                  />
                  {errors.name ? (
                    <p className="text-destructive text-xs">
                      {errorText(errors.name)}
                    </p>
                  ) : null}
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="membership-price">
                    {t("price", "Price")}
                  </Label>
                  <EnhancedInput<number>
                    aria-invalid={Boolean(errors.unit_price)}
                    formatInput={(value) =>
                      unitConversion("centsToDollars", value)
                    }
                    formatOutput={(value) =>
                      unitConversion("dollarsToCents", value)
                    }
                    id="membership-price"
                    min={0}
                    onValueChange={(value) => updateForm("unit_price", value)}
                    type="number"
                    value={form.unit_price}
                  />
                  {errors.unit_price ? (
                    <p className="text-destructive text-xs">
                      {errorText(errors.unit_price)}
                    </p>
                  ) : null}
                  <p className="text-muted-foreground text-xs">
                    {t(
                      "priceDescription",
                      "This is the price for one full membership card period, not a monthly unit price."
                    )}
                  </p>
                </div>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="grid gap-2">
                  <Label htmlFor="membership-duration-value">
                    {t("durationValue", "Duration")}
                  </Label>
                  <Input
                    aria-invalid={Boolean(errors.duration_value)}
                    id="membership-duration-value"
                    min={1}
                    onChange={(event) =>
                      updateForm(
                        "duration_value",
                        Number(event.target.value) || 1
                      )
                    }
                    type="number"
                    value={form.duration_value}
                  />
                  {errors.duration_value ? (
                    <p className="text-destructive text-xs">
                      {errorText(errors.duration_value)}
                    </p>
                  ) : null}
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="membership-duration-unit">
                    {t("durationUnit", "Duration Unit")}
                  </Label>
                  <Select
                    onValueChange={(value) =>
                      updateForm("duration_unit", value as DurationUnit)
                    }
                    value={form.duration_unit || "Year"}
                  >
                    <SelectTrigger
                      className="w-full"
                      id="membership-duration-unit"
                    >
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {durationUnits.map((unit) => (
                          <SelectItem key={unit} value={unit}>
                            {durationLabel(unit)}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <section className="border-t pt-5">
                <div className="flex flex-col gap-1">
                  <h3 className="font-semibold text-base">
                    {t("policy.title", "Abuse Prevention Policies")}
                  </h3>
                  <p className="text-muted-foreground text-sm">
                    {t(
                      "policy.description",
                      "Choose which monthly limits apply to active members when they purchase, renew, or reset subscription traffic."
                    )}
                  </p>
                </div>
                <div className="mt-4 grid gap-5 md:grid-cols-2">
                  <div className="flex flex-col gap-3 border-b pb-5">
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <Label htmlFor="membership-monthly-consumption-limit">
                          {t(
                            "policy.monthlyConsumption",
                            "Monthly Consumption Limit"
                          )}
                        </Label>
                        <p className="text-muted-foreground text-xs">
                          {t(
                            "policy.monthlyConsumptionHint",
                            "Counts pending, paid, and completed subscription orders."
                          )}
                        </p>
                      </div>
                      <Switch
                        aria-label={t(
                          "policy.monthlyConsumption",
                          "Monthly Consumption Limit"
                        )}
                        checked={form.policy?.monthly_consumption_limit_enabled}
                        onCheckedChange={(checked) =>
                          updatePolicy(
                            "monthly_consumption_limit_enabled",
                            checked
                          )
                        }
                      />
                    </div>
                    <EnhancedInput<number>
                      aria-invalid={Boolean(errors.monthly_consumption_limit)}
                      disabled={!form.policy?.monthly_consumption_limit_enabled}
                      formatInput={(value) =>
                        unitConversion("centsToDollars", value)
                      }
                      formatOutput={(value) =>
                        unitConversion("dollarsToCents", value)
                      }
                      id="membership-monthly-consumption-limit"
                      min={0}
                      onValueChange={(value) =>
                        updatePolicy("monthly_consumption_limit", value)
                      }
                      type="number"
                      value={form.policy?.monthly_consumption_limit || 0}
                    />
                    {errors.monthly_consumption_limit ? (
                      <p className="text-destructive text-xs">
                        {errorText(errors.monthly_consumption_limit)}
                      </p>
                    ) : null}
                  </div>

                  <div className="flex flex-col gap-3 border-b pb-5">
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <Label htmlFor="membership-monthly-order-limit">
                          {t("policy.monthlyOrders", "Monthly Order Limit")}
                        </Label>
                        <p className="text-muted-foreground text-xs">
                          {t(
                            "policy.monthlyOrdersHint",
                            "Limits the number of subscription orders a member can create each month."
                          )}
                        </p>
                      </div>
                      <Switch
                        aria-label={t(
                          "policy.monthlyOrders",
                          "Monthly Order Limit"
                        )}
                        checked={form.policy?.monthly_order_limit_enabled}
                        onCheckedChange={(checked) =>
                          updatePolicy("monthly_order_limit_enabled", checked)
                        }
                      />
                    </div>
                    <Input
                      aria-invalid={Boolean(errors.monthly_order_limit)}
                      disabled={!form.policy?.monthly_order_limit_enabled}
                      id="membership-monthly-order-limit"
                      min={0}
                      onChange={(event) =>
                        updatePolicy(
                          "monthly_order_limit",
                          Number(event.target.value) || 0
                        )
                      }
                      type="number"
                      value={form.policy?.monthly_order_limit || 0}
                    />
                    {errors.monthly_order_limit ? (
                      <p className="text-destructive text-xs">
                        {errorText(errors.monthly_order_limit)}
                      </p>
                    ) : null}
                  </div>

                  <div className="flex flex-col gap-3 border-b pb-5 md:col-span-2">
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <Label>
                          {t(
                            "policy.subscriptionExpiry",
                            "Subscription Expiry Guard"
                          )}
                        </Label>
                        <p className="text-muted-foreground text-xs">
                          {t(
                            "policy.subscriptionExpiryHint",
                            "Block subscription purchases or renewals when the resulting expiry is beyond the membership expiry plus the allowed days."
                          )}
                        </p>
                      </div>
                      <Switch
                        aria-label={t(
                          "policy.subscriptionExpiry",
                          "Subscription Expiry Guard"
                        )}
                        checked={form.policy?.subscription_expiry_limit_enabled}
                        onCheckedChange={(checked) =>
                          updatePolicy(
                            "subscription_expiry_limit_enabled",
                            checked
                          )
                        }
                      />
                    </div>
                    <div className="grid gap-2 md:max-w-xs">
                      <Label htmlFor="membership-subscription-expiry-grace">
                        {t(
                          "policy.subscriptionExpiryGraceDays",
                          "Allowed extra days"
                        )}
                      </Label>
                      <Input
                        aria-invalid={Boolean(
                          errors.subscription_expiry_grace_days
                        )}
                        disabled={
                          !form.policy?.subscription_expiry_limit_enabled
                        }
                        id="membership-subscription-expiry-grace"
                        min={0}
                        onChange={(event) =>
                          updatePolicy(
                            "subscription_expiry_grace_days",
                            Number(event.target.value) || 0
                          )
                        }
                        type="number"
                        value={form.policy?.subscription_expiry_grace_days || 0}
                      />
                      {errors.subscription_expiry_grace_days ? (
                        <p className="text-destructive text-xs">
                          {errorText(errors.subscription_expiry_grace_days)}
                        </p>
                      ) : null}
                    </div>
                  </div>
                </div>
              </section>

              <div className="grid gap-2">
                <Label htmlFor="membership-description">
                  {t("cardDescription", "Description")}
                </Label>
                <Textarea
                  aria-invalid={Boolean(errors.description)}
                  className="min-h-28"
                  id="membership-description"
                  onChange={(event) =>
                    updateForm("description", event.target.value)
                  }
                  placeholder={t(
                    "descriptionPlaceholder",
                    "Required before purchasing subscription products."
                  )}
                  value={form.description}
                />
                {errors.description ? (
                  <p className="text-destructive text-xs">
                    {errorText(errors.description)}
                  </p>
                ) : null}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("preview", "Preview")}</CardTitle>
              <CardDescription>
                {t(
                  "previewDescription",
                  "The membership card configuration retained for the user experience."
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-5">
              <div className="flex flex-col gap-3">
                <div className="flex items-center justify-between gap-3">
                  <h2 className="font-medium text-lg">
                    {form.name || t("emptyName", "Membership Card")}
                  </h2>
                  <Badge variant={form.enabled ? "default" : "outline"}>
                    {form.enabled
                      ? t("status.enabled", "Enabled")
                      : t("status.disabled", "Checks disabled")}
                  </Badge>
                </div>
                <p className="min-h-10 text-muted-foreground text-sm">
                  {form.description ||
                    t("emptyDescription", "No description yet.")}
                </p>
              </div>

              <Separator />

              <div className="grid gap-3 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">
                    {t("previewDuration", "Duration")}
                  </span>
                  <span className="font-medium">
                    {form.duration_value || 1}{" "}
                    {durationLabel(form.duration_unit)}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">
                    {t("previewPrice", "Price")}
                  </span>
                  <span className="font-semibold text-xl">
                    <Display type="currency" value={form.unit_price} />
                  </span>
                </div>
                <div className="flex items-start justify-between gap-4">
                  <span className="text-muted-foreground">
                    {t("previewPolicies", "Policies")}
                  </span>
                  <span className="text-right text-xs leading-5">
                    {form.policy?.monthly_consumption_limit_enabled ? (
                      <span className="block">
                        {t("policy.monthlyConsumption", "Monthly Consumption")}:{" "}
                        <Display
                          type="currency"
                          value={form.policy.monthly_consumption_limit}
                        />
                      </span>
                    ) : null}
                    {form.policy?.monthly_order_limit_enabled ? (
                      <span className="block">
                        {t("policy.monthlyOrders", "Monthly Orders")}:{" "}
                        {form.policy.monthly_order_limit}
                      </span>
                    ) : null}
                    {form.policy?.subscription_expiry_limit_enabled ? (
                      <span className="block">
                        {t("policy.subscriptionExpiry", "Subscription Expiry")}:
                        +{form.policy.subscription_expiry_grace_days || 0}{" "}
                        {t("days", "days")}
                      </span>
                    ) : null}
                    {form.policy?.monthly_consumption_limit_enabled ||
                    form.policy?.monthly_order_limit_enabled ||
                    form.policy?.subscription_expiry_limit_enabled
                      ? null
                      : t("noPolicies", "No policy enabled")}
                  </span>
                </div>
                {data?.id ? (
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">
                      {t("updatedAt", "Updated At")}
                    </span>
                    <span>
                      {data.updated_at ? formatDate(data.updated_at) : "--"}
                    </span>
                  </div>
                ) : null}
              </div>
            </CardContent>
          </Card>
        </div>
      </AdminPage>
      <DiscardChangesDialog
        onDiscard={() => {
          setForm(baseline);
          setErrors({});
          setDiscardOpen(false);
          refreshPlan().catch(() =>
            toast.error(t("loadFailed", "Unable to load membership plan"))
          );
        }}
        onOpenChange={setDiscardOpen}
        open={discardOpen}
      />
    </>
  );
}
