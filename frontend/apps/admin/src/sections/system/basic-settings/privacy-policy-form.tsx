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
import { LazyMarkdownEditor } from "@/components/lazy-markdown-editor";
import { ShieldCheck, ChevronRight, LoaderCircle } from "lucide-react";
import {
  getPrivacyPolicyConfig,
  updatePrivacyPolicyConfig,
} from "@workspace/ui/services/admin/system";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

const privacyPolicySchema = z.object({
  privacy_policy: z.string().optional(),
});

type PrivacyPolicyFormData = z.infer<typeof privacyPolicySchema>;

export default function PrivacyPolicyConfig() {
  const { t } = useTranslation("system");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const { data, refetch } = useQuery({
    queryKey: ["getPrivacyPolicyConfig"],
    queryFn: async () => {
      const { data } = await getPrivacyPolicyConfig();
      return data.data;
    },
    enabled: open,
  });

  const form = useForm<PrivacyPolicyFormData>({
    resolver: zodResolver(privacyPolicySchema),
    defaultValues: {
      privacy_policy: "",
    },
  });

  useEffect(() => {
    if (data) {
      form.reset({
        privacy_policy: data.privacy_policy || "",
      });
    }
  }, [data, form]);

  async function onSubmit(values: PrivacyPolicyFormData) {
    setLoading(true);
    try {
      await updatePrivacyPolicyConfig(values as API.PrivacyPolicyConfig);
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
    <Sheet onOpenChange={setOpen} open={open}>
      <SheetTrigger asChild>
        <Button
          className="admin-setting-trigger h-auto w-full justify-between whitespace-normal text-left"
          type="button"
          variant="ghost"
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <ShieldCheck className="size-5 text-muted-foreground" />
            </div>
            <div className="flex-1">
              <p className="font-medium">
                {t("privacyPolicy.title", "Privacy Policy")}
              </p>
              <p className="text-muted-foreground text-sm">
                {t(
                  "privacyPolicy.description",
                  "Edit and manage privacy policy content"
                )}
              </p>
            </div>
          </div>
          <ChevronRight className="size-4 text-muted-foreground" />
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[600px] max-w-full md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>{t("privacyPolicy.title", "Privacy Policy")}</SheetTitle>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-24px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <form
              className="space-y-2 pt-4"
              id="privacy-policy-form"
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <FormField
                control={form.control}
                name="privacy_policy"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("privacyPolicy.title", "Privacy Policy")}
                    </FormLabel>
                    <FormControl>
                      <LazyMarkdownEditor
                        onChange={field.onChange}
                        value={field.value || ""}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "privacyPolicy.description",
                        "Edit and manage privacy policy content"
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
            disabled={loading}
            onClick={() => setOpen(false)}
            variant="outline"
          >
            {t("common.cancel", "Cancel")}
          </Button>
          <Button disabled={loading} form="privacy-policy-form" type="submit">
            {loading && <LoaderCircle className="animate-spin" />}
            {t("common.save", "Save Settings")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
