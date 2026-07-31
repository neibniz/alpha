import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@workspace/ui/components/button";
import {
  Form,
  FormControl,
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
import { Switch } from "@workspace/ui/components/switch";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import { unitConversion } from "@workspace/ui/utils/unit-conversions";
import { type ReactNode, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { useGlobalStore } from "@/stores/global";

type UserFormValues = {
  email: string;
  password: string;
  referer_user?: string;
  refer_code?: string;
  referral_percentage?: number;
  only_first_purchase?: boolean;
  is_admin?: boolean;
  balance?: number;
  gift_amount?: number;
  commission?: number;
};

interface UserFormProps<T> {
  onSubmit: (data: T) => Promise<boolean> | boolean;
  initialValues?: Partial<T> & Partial<UserFormValues>;
  loading?: boolean;
  trigger: ReactNode;
  title: string;
  update?: boolean;
}

export default function UserForm<T extends object>({
  onSubmit,
  initialValues,
  loading,
  trigger,
  title,
}: Readonly<UserFormProps<T>>) {
  const { t } = useTranslation("user");
  const { common } = useGlobalStore();
  const { currency } = common;

  const [open, setOpen] = useState(false);
  const formSchema = z.object({
    email: z.email(t("invalidEmailFormat", "Invalid email format")),
    password: z
      .string()
      .min(
        12,
        t("passwordMinLength", "Password must be at least 12 characters")
      )
      .max(
        72,
        t("passwordMaxLength", "Password must be at most 72 characters")
      ),
    referer_user: z.string().optional(),
    refer_code: z.string().optional(),
    referral_percentage: z.number().optional(),
    only_first_purchase: z.boolean().optional(),
    is_admin: z.boolean().optional(),
    balance: z.number().optional(),
    gift_amount: z.number().optional(),
    commission: z.number().optional(),
  });
  const form = useForm<UserFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      email: "",
      password: "",
      referer_user: "",
      refer_code: "",
      referral_percentage: 0,
      only_first_purchase: true,
      balance: 0,
      gift_amount: 0,
      commission: 0,
      is_admin: false,
      ...initialValues,
    },
  });

  useEffect(() => {
    form?.reset(initialValues);
  }, [form, initialValues]);

  async function handleSubmit(data: UserFormValues) {
    const bool = await onSubmit(data as T);

    if (bool) setOpen(false);
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <SheetTrigger asChild>
        <Button
          onClick={() => {
            form.reset();
            setOpen(true);
          }}
        >
          {trigger}
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[640px] max-w-full gap-0 md:max-w-screen-md">
        <SheetHeader className="border-b px-5 py-4">
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription>
            {t(
              "createDescription",
              "Create account credentials, referral policy, opening balances, and permissions."
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-153px-env(safe-area-inset-top))]">
          <Form {...form}>
            <form
              className="grid gap-4 px-5 py-5 sm:grid-cols-2"
              onSubmit={form.handleSubmit(handleSubmit)}
            >
              <div className="sm:col-span-2">
                <h3 className="font-medium text-sm">
                  {t("accountAccessSection", "Account access")}
                </h3>
                <p className="mt-1 text-muted-foreground text-xs">
                  {t(
                    "accountAccessDescription",
                    "Email is the primary sign-in identity; GitHub can be connected separately."
                  )}
                </p>
              </div>
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("userEmail", "Email")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        placeholder={t("userEmailPlaceholder", "Enter email")}
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
                name="password"
                render={({ field }) => (
                  <FormItem className="sm:col-span-2">
                    <FormLabel>{t("password", "Password")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        autoComplete="new-password"
                        placeholder={t("passwordPlaceholder", "Enter password")}
                        type="password"
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
              <div className="border-t pt-4 sm:col-span-2">
                <h3 className="font-medium text-sm">
                  {t("referralSection", "Referral policy")}
                </h3>
                <p className="mt-1 text-muted-foreground text-xs">
                  {t(
                    "referralSectionDescription",
                    "Optionally link a referrer and define this user's commission policy."
                  )}
                </p>
              </div>
              <FormField
                control={form.control}
                name="referer_user"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("refererAccount", "Referrer email")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        placeholder={t(
                          "refererAccountPlaceholder",
                          "Enter referrer email"
                        )}
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
                name="refer_code"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("inviteCode", "Invite Code")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        placeholder={t(
                          "inviteCodePlaceholder",
                          "Enter invite code"
                        )}
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
                name="referral_percentage"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("referralPercentage", "Referral Percentage")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        max={100}
                        min={0}
                        placeholder={t(
                          "referralPercentagePlaceholder",
                          "Enter percentage"
                        )}
                        type="number"
                        {...field}
                        onValueChange={(value) => {
                          form.setValue(field.name, Number(value));
                        }}
                        suffix="%"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="only_first_purchase"
                render={({ field }) => (
                  <FormItem className="flex min-h-16 items-center justify-between gap-3 rounded-md border px-3 py-2">
                    <div>
                      <FormLabel>
                        {t("onlyFirstPurchase", "First purchase only")}
                      </FormLabel>
                      <p className="mt-1 text-muted-foreground text-xs">
                        {t(
                          "onlyFirstPurchaseDescription",
                          "Apply commission only to the first paid order."
                        )}
                      </p>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
              <div className="border-t pt-4 sm:col-span-2">
                <h3 className="font-medium text-sm">
                  {t("openingBalancesSection", "Opening balances")}
                </h3>
                <p className="mt-1 text-muted-foreground text-xs">
                  {t(
                    "openingBalancesDescription",
                    "Set optional starting balances in the configured site currency."
                  )}
                </p>
              </div>
              <FormField
                control={form.control}
                name="balance"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("balance", "Balance")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        placeholder={t("balancePlaceholder", "Enter balance")}
                        prefix={currency?.currency_symbol ?? "$"}
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
                name="gift_amount"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("giftAmount", "Gift Amount")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        placeholder={t(
                          "giftAmountPlaceholder",
                          "Enter gift amount"
                        )}
                        prefix={currency?.currency_symbol ?? "$"}
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
                name="commission"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("commission", "Commission")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        placeholder={t(
                          "commissionPlaceholder",
                          "Enter commission"
                        )}
                        prefix={currency?.currency_symbol ?? "$"}
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
                name="is_admin"
                render={({ field }) => (
                  <FormItem className="flex min-h-16 items-center justify-between gap-3 rounded-md border px-3 py-2 sm:col-span-2">
                    <div>
                      <FormLabel>{t("manager", "Administrator")}</FormLabel>
                      <p className="mt-1 text-muted-foreground text-xs">
                        {t(
                          "administratorDescription",
                          "Grant access to the administrative console."
                        )}
                      </p>
                    </div>
                    <FormControl>
                      <div>
                        <Switch
                          checked={!!field.value}
                          onCheckedChange={field.onChange}
                        />
                      </div>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </form>
          </Form>
        </ScrollArea>
        <SheetFooter className="flex-row justify-end gap-2 border-t px-5 py-3">
          <Button
            disabled={loading}
            onClick={() => {
              setOpen(false);
            }}
            variant="outline"
          >
            {t("cancel", "Cancel")}
          </Button>
          <Button disabled={loading} onClick={form.handleSubmit(handleSubmit)}>
            {loading && (
              <Icon className="mr-2 animate-spin" icon="mdi:loading" />
            )}{" "}
            {t("confirm", "Confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
