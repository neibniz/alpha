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
import { Input } from "@workspace/ui/components/input";
import { ScrollArea } from "@workspace/ui/components/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import { LazyMarkdownEditor } from "@/components/lazy-markdown-editor";
import { Icon } from "@workspace/ui/composed/icon";
import { TagInput } from "@workspace/ui/composed/tag-input";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";

type DocumentFormValues = {
  title: string;
  tags: string[];
  content: string;
};

interface DocumentFormProps<T> {
  onSubmit: (data: T) => Promise<boolean> | boolean;
  initialValues?: Partial<T> & Partial<DocumentFormValues>;
  loadInitialValues?: () => Promise<
    (Partial<T> & Partial<DocumentFormValues>) | undefined
  >;
  loading?: boolean;
  trigger: string;
  title: string;
}

export default function DocumentForm<T extends object>({
  onSubmit,
  initialValues,
  loadInitialValues,
  loading,
  trigger,
  title,
}: DocumentFormProps<T>) {
  const { t } = useTranslation("document");
  const [open, setOpen] = useState(false);
  const [opening, setOpening] = useState(false);
  const formSchema = z.object({
    title: z
      .string()
      .trim()
      .min(1, t("form.titleRequired", "Document title is required")),
    tags: z.array(z.string()),
    content: z
      .string()
      .trim()
      .min(1, t("form.contentRequired", "Document content is required")),
  });
  const form = useForm<DocumentFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      title: "",
      tags: [],
      content: "",
      ...initialValues,
    },
  });

  useEffect(() => {
    form?.reset({
      title: "",
      tags: [],
      content: "",
      ...initialValues,
    });
  }, [form, initialValues]);

  async function handleSubmit(data: DocumentFormValues) {
    const bool = await onSubmit(data as T);
    if (bool) setOpen(false);
  }

  async function handleOpen() {
    setOpening(true);
    try {
      const values = loadInitialValues
        ? await loadInitialValues()
        : initialValues;
      if (loadInitialValues && !values) {
        return;
      }
      form.reset({
        title: "",
        tags: [],
        content: "",
        ...values,
      });
      setOpen(true);
    } finally {
      setOpening(false);
    }
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <Button disabled={opening} onClick={handleOpen}>
        {opening && <Icon className="mr-2 animate-spin" icon="mdi:loading" />}
        {trigger}
      </Button>
      <SheetContent className="w-[800px] max-w-full md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
        </SheetHeader>
        <ScrollArea className="h-[calc(100vh-48px-36px-36px-env(safe-area-inset-top))]">
          <Form {...form}>
            <form
              className="space-y-4 px-6 pt-4"
              onSubmit={form.handleSubmit(handleSubmit)}
            >
              <FormField
                control={form.control}
                name="title"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.title", "Title")}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t(
                          "form.titlePlaceholder",
                          "Enter document title"
                        )}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="tags"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.tags", "Tags")}</FormLabel>
                    <FormControl>
                      <TagInput
                        onChange={field.onChange}
                        placeholder={t("form.tagsPlaceholder", "Enter tags")}
                        value={field.value}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="content"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.content", "Content")}</FormLabel>
                    <FormControl>
                      <LazyMarkdownEditor
                        onChange={(value) => field.onChange(value || "")}
                        value={field.value}
                      />
                    </FormControl>
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
            onClick={() => {
              setOpen(false);
            }}
            variant="outline"
          >
            {t("form.cancel", "Cancel")}
          </Button>
          <Button disabled={loading} onClick={form.handleSubmit(handleSubmit)}>
            {loading && (
              <Icon className="mr-2 animate-spin" icon="mdi:loading" />
            )}{" "}
            {t("form.confirm", "Confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
