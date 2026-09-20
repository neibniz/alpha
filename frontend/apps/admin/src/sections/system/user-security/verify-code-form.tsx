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
import { MailCheck, ChevronRight, LoaderCircle } from "lucide-react";
import {
  getVerifyCodeConfig,
  updateVerifyCodeConfig,
} from "@workspace/ui/services/admin/system";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

const verifyCodeSchema = z.object({
  verify_code_expire_time: z.number().int().min(60).optional(),
  verify_code_interval: z.number().int().min(30).optional(),
  verify_code_limit: z.number().int().min(1).optional(),
});

type VerifyCodeFormData = z.infer<typeof verifyCodeSchema>;

export default function VerifyCodeConfig() {
  const { t } = useTranslation("system");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const { data, refetch } = useQuery({
    queryKey: ["getVerifyCodeConfig"],
    queryFn: async () => {
      const { data } = await getVerifyCodeConfig();
      return data.data;
    },
    enabled: open,
  });

  const form = useForm<VerifyCodeFormData>({
    resolver: zodResolver(verifyCodeSchema),
    defaultValues: {
      verify_code_expire_time: 300,
      verify_code_interval: 60,
      verify_code_limit: 10,
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

  async function onSubmit(values: VerifyCodeFormData) {
    setLoading(true);
    try {
      await updateVerifyCodeConfig(values as API.VerifyCodeConfig);
      toast.success(t("verifyCode.saveSuccess", "Save Successful"));
      refetch();
      setOpen(false);
    } catch (_error) {
      toast.error(t("verifyCode.saveFailed", "Save Failed"));
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
              <MailCheck className="size-5 text-muted-foreground" />
            </div>
            <div className="flex-1">
              <p className="font-medium">
                {t("verifyCode.title", "Verification Code Settings")}
              </p>
              <p className="text-muted-foreground text-sm">
                {t(
                  "verifyCode.description",
                  "Configure email verification code sending rules and limits"
                )}
              </p>
            </div>
          </div>
          <ChevronRight className="size-4 text-muted-foreground" />
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[600px] max-w-full md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>
            {t("verifyCode.title", "Verification Code Settings")}
          </SheetTitle>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-24px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <AdminForm
              className="space-y-2 pt-4"
              id="verify-code-form"
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <FormField
                control={form.control}
                name="verify_code_expire_time"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("verifyCode.expireTime", "Verification Code Validity")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={60}
                        onValueBlur={(value) => field.onChange(Number(value))}
                        placeholder={t(
                          "verifyCode.inputPlaceholder",
                          "Please enter"
                        )}
                        step={1}
                        suffix={t("verifyCode.seconds", "seconds")}
                        type="number"
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "verifyCode.expireTimeDescription",
                        "Validity period of verification codes (seconds)"
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="verify_code_interval"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("verifyCode.interval", "Sending Interval")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={30}
                        onValueBlur={(value) => field.onChange(Number(value))}
                        placeholder={t(
                          "verifyCode.inputPlaceholder",
                          "Please enter"
                        )}
                        step={1}
                        suffix={t("verifyCode.seconds", "seconds")}
                        type="number"
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "verifyCode.intervalDescription",
                        "Minimum interval between two verification code sends (seconds)"
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="verify_code_limit"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("verifyCode.dailyLimit", "Daily Sending Limit")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={1}
                        onValueBlur={(value) => field.onChange(Number(value))}
                        placeholder={t(
                          "verifyCode.inputPlaceholder",
                          "Please enter"
                        )}
                        step={1}
                        suffix={t("verifyCode.times", "time(s)")}
                        type="number"
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "verifyCode.dailyLimitDescription",
                        "Maximum number of verification codes each user can send per day"
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
          <Button disabled={loading} form="verify-code-form" type="submit">
            {loading && <LoaderCircle className="animate-spin" />}
            {t("common.save", "Save Settings")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
