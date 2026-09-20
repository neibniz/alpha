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
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { CircleDollarSign, ChevronRight, LoaderCircle } from "lucide-react";
import {
  getCurrencyConfig,
  updateCurrencyConfig,
} from "@workspace/ui/services/admin/system";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

// Constants
const EXCHANGE_RATE_HOST_URL = "https://exchangerate.host";

const currencySchema = z.object({
  access_key: z.string().optional(),
  currency_unit: z.string().min(1),
  currency_symbol: z.string().min(1),
});

type CurrencyFormData = z.infer<typeof currencySchema>;

export default function CurrencyConfig() {
  const { t } = useTranslation("system");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const { data, refetch } = useQuery({
    queryKey: ["getCurrencyConfig"],
    queryFn: async () => {
      const { data } = await getCurrencyConfig();
      return data.data;
    },
    enabled: open, // Only request data when the modal is open
  });

  const form = useForm<CurrencyFormData>({
    resolver: zodResolver(currencySchema),
    defaultValues: {
      access_key: "",
      currency_unit: "USD",
      currency_symbol: "$",
    },
  });

  useEffect(() => {
    if (data && !form.formState.isDirty) {
      form.reset(data);
    }
  }, [data, form]);

  const closeGuard = useFormCloseGuard({
    open,
    dirty: form.formState.isDirty,
    busy: loading || form.formState.isSubmitting,
    onOpenChange: (next) => {
      if (!next) form.reset();
      setOpen(next);
    },
  });

  async function onSubmit(values: CurrencyFormData) {
    setLoading(true);
    try {
      await updateCurrencyConfig(values as API.CurrencyConfig);
      toast.success(t("common.saveSuccess", "Save Successful"));
      refetch();
      setOpen(false);
    } catch (_error) {
      toast.error(t("common.saveFailed", "Save Failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Sheet onOpenChange={closeGuard.requestOpenChange} open={open}>
      {closeGuard.dialog}
      <SheetTrigger asChild>
        <Button
          className="admin-setting-trigger h-auto w-full justify-between whitespace-normal text-left"
          type="button"
          variant="ghost"
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <CircleDollarSign className="size-5 text-muted-foreground" />
            </div>
            <div className="flex-1">
              <p className="font-medium">
                {t("currency.title", "Currency Configuration")}
              </p>
              <p className="text-muted-foreground text-sm">
                {t(
                  "currency.description",
                  "Configure currency units, symbols, and exchange rate API settings"
                )}
              </p>
            </div>
          </div>
          <ChevronRight className="size-4 text-muted-foreground" />
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[600px] max-w-full gap-0 md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>
            {t("currency.title", "Currency Configuration")}
          </SheetTitle>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-24px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <AdminForm
              className="space-y-2 pt-4"
              id="currency-form"
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <FormField
                control={form.control}
                name="access_key"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("currency.accessKey", "API Key")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={field.onChange}
                        placeholder={t(
                          "currency.accessKeyPlaceholder",
                          "Enter API key"
                        )}
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "currency.accessKeyDescription",
                        "Free exchange rate API key provided by {{url}}",
                        {
                          url: EXCHANGE_RATE_HOST_URL,
                        }
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="currency_unit"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("currency.currencyUnit", "Currency Unit")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={field.onChange}
                        placeholder={t(
                          "currency.currencyUnitPlaceholder",
                          "USD"
                        )}
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "currency.currencyUnitDescription",
                        "Used for display purposes only; changing this will affect all currency units in the system"
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="currency_symbol"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("currency.currencySymbol", "Currency Symbol")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={field.onChange}
                        placeholder={t(
                          "currency.currencySymbolPlaceholder",
                          "$"
                        )}
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "currency.currencySymbolDescription",
                        "Used for display purposes only; changing this will affect all currency units in the system"
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </AdminForm>
          </Form>
        </ScrollArea>
        <SheetFooter className="flex-row justify-end gap-2 pt-3">
          <Button
            disabled={loading}
            onClick={() => closeGuard.requestOpenChange(false)}
            variant="outline"
          >
            {t("common.cancel", "Cancel")}
          </Button>
          <Button disabled={loading} form="currency-form" type="submit">
            {loading && <LoaderCircle className="animate-spin" />}
            {t("common.save", "Save Settings")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
