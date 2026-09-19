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
import { FileText, ChevronRight, LoaderCircle } from "lucide-react";
import {
  getTosConfig,
  updateTosConfig,
} from "@workspace/ui/services/admin/system";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

const tosSchema = z.object({
  tos_content: z.string().optional(),
});

type TosFormData = z.infer<typeof tosSchema>;

export default function TosConfig() {
  const { t } = useTranslation("system");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const { data, refetch } = useQuery({
    queryKey: ["getTosConfig"],
    queryFn: async () => {
      const { data } = await getTosConfig();
      return data.data;
    },
    enabled: open,
  });

  const form = useForm<TosFormData>({
    resolver: zodResolver(tosSchema),
    defaultValues: {
      tos_content: "",
    },
  });

  useEffect(() => {
    if (data) {
      form.reset({
        tos_content: data.tos_content || "",
      });
    }
  }, [data, form]);

  async function onSubmit(values: TosFormData) {
    setLoading(true);
    try {
      await updateTosConfig(values as API.TosConfig);
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
              <FileText className="size-5 text-muted-foreground" />
            </div>
            <div className="flex-1">
              <p className="font-medium">
                {t("tos.title", "Terms of Service")}
              </p>
              <p className="text-muted-foreground text-sm">
                {t(
                  "tos.description",
                  "Edit and manage terms of service content"
                )}
              </p>
            </div>
          </div>
          <ChevronRight className="size-4 text-muted-foreground" />
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[600px] max-w-full md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>{t("tos.title", "Terms of Service")}</SheetTitle>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-24px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <form
              className="space-y-2 pt-4"
              id="tos-form"
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <FormField
                control={form.control}
                name="tos_content"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("tos.title", "Terms of Service")}</FormLabel>
                    <FormControl>
                      <LazyMarkdownEditor
                        onChange={field.onChange}
                        value={field.value || ""}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "tos.description",
                        "Edit and manage terms of service content"
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
          <Button disabled={loading} form="tos-form" type="submit">
            {loading && <LoaderCircle className="animate-spin" />}
            {t("common.save", "Save Settings")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
