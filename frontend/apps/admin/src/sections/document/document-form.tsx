import { AdminActionButton } from "@/components/admin-action-button";
import { LoaderCircle, Pencil, Plus } from "lucide-react";
import { useFormCloseGuard } from "@/components/use-form-close-guard";
import { AdminForm } from "@/components/admin-form";
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
  SheetDescription,
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
    if (form.formState.isDirty) return;
    form?.reset({
      title: "",
      tags: [],
      content: "",
      ...initialValues,
    });
  }, [form, initialValues]);

  const closeGuard = useFormCloseGuard({
    open,
    dirty: form.formState.isDirty,
    busy: Boolean(loading || form.formState.isSubmitting),
    onOpenChange: (next) => {
      if (!next) form.reset();
      setOpen(next);
    },
  });

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
    <Sheet onOpenChange={closeGuard.requestOpenChange} open={open}>
      {closeGuard.dialog}
      {initialValues ? (
        <AdminActionButton
          disabled={opening}
          onClick={handleOpen}
          label={trigger}
          icon={opening ? LoaderCircle : Pencil}
        />
      ) : (
        <Button disabled={opening} onClick={handleOpen}>
          {opening ? (
            <LoaderCircle className="size-4 animate-spin" />
          ) : (
            <Plus className="size-4" />
          )}
          {trigger}
        </Button>
      )}
      <SheetContent className="w-[800px] max-w-full md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription>
            {t(
              "form.dialogDescription",
              "Edit content and review its publication settings."
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="h-[calc(100vh-48px-36px-36px-env(safe-area-inset-top))]">
          <Form {...form}>
            <AdminForm
              className="px-6 py-6"
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
            </AdminForm>
          </Form>
        </ScrollArea>
        <SheetFooter className="flex-row justify-end gap-2 pt-3">
          <Button
            disabled={loading}
            onClick={() => {
              closeGuard.requestOpenChange(false);
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
