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
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";

const formSchema = z.object({
  title: z.string(),
  type: z.enum(["image", "video"]),
  content: z.string(),
  description: z.string(),
  target_url: z.string().url(),
  start_time: z.number().int().nonnegative(),
  end_time: z.number().int().nonnegative(),
});

type AdsFormValues = z.infer<typeof formSchema>;
type AdsInitialValues = Partial<Omit<AdsFormValues, "type">> & {
  type?: string;
};

function normalizeAdsType(value?: string): AdsFormValues["type"] {
  return value === "video" ? "video" : "image";
}

function buildAdsFormValues(values?: AdsInitialValues): AdsFormValues {
  return {
    title: values?.title ?? "",
    type: normalizeAdsType(values?.type),
    content: values?.content ?? "",
    description: values?.description ?? "",
    target_url: values?.target_url ?? "",
    start_time: values?.start_time ?? 0,
    end_time: values?.end_time ?? 0,
  };
}

interface AdsFormProps<T> {
  onSubmit: (data: T) => Promise<boolean> | boolean;
  initialValues?: Partial<T> & AdsInitialValues;
  loadInitialValues?: () => Promise<
    (Partial<T> & AdsInitialValues) | undefined
  >;
  loading?: boolean;
  trigger: string;
  title: string;
}

export default function AdsForm<T extends object>({
  onSubmit,
  initialValues,
  loadInitialValues,
  loading,
  trigger,
  title,
}: AdsFormProps<T>) {
  const { t } = useTranslation("ads");
  const [open, setOpen] = useState(false);
  const [opening, setOpening] = useState(false);

  const form = useForm<AdsFormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: buildAdsFormValues(initialValues),
  });

  const type = form.watch("type");
  const startTime = form.watch("start_time");

  const renderContentField = () => (
    <FormField
      control={form.control}
      name="content"
      render={({ field }) => (
        <FormItem>
          <FormLabel>{t("form.content", "Content")}</FormLabel>
          <FormControl>
            <EnhancedInput
              onValueChange={(value) => {
                form.setValue("content", value);
              }}
              placeholder={
                type === "image"
                  ? "https://example.com/image.jpg"
                  : "https://example.com/video.mp4"
              }
              value={field.value}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  );

  const closeGuard = useFormCloseGuard({
    open,
    dirty: form.formState.isDirty,
    busy: Boolean(loading || form.formState.isSubmitting),
    onOpenChange: (next) => {
      if (!next) form.reset();
      setOpen(next);
    },
  });

  async function handleSubmit(data: AdsFormValues) {
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
      form.reset(buildAdsFormValues(values));
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
      <SheetContent className="w-[500px] max-w-full md:max-w-screen-md">
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
                      <EnhancedInput
                        onValueChange={(value) => {
                          form.setValue(field.name, value, {
                            shouldDirty: true,
                          });
                        }}
                        placeholder={t("form.enterTitle", "Enter title")}
                        value={field.value}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.type", "Type")}</FormLabel>
                    <FormControl>
                      <RadioGroup
                        className="flex gap-4"
                        defaultValue={field.value}
                        onValueChange={(value) => {
                          form.setValue(field.name, normalizeAdsType(value), {
                            shouldDirty: true,
                          });
                        }}
                      >
                        <FormItem className="flex items-center space-x-3 space-y-0">
                          <FormControl>
                            <RadioGroupItem value="image" />
                          </FormControl>
                          <FormLabel className="font-normal">
                            {t("form.typeImage", "Image")}
                          </FormLabel>
                        </FormItem>
                        <FormItem className="flex items-center space-x-3 space-y-0">
                          <FormControl>
                            <RadioGroupItem value="video" />
                          </FormControl>
                          <FormLabel className="font-normal">
                            {t("form.typeVideo", "Video")}
                          </FormLabel>
                        </FormItem>
                      </RadioGroup>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {renderContentField()}

              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("form.description", "Description")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={(value) => {
                          form.setValue(field.name, value, {
                            shouldDirty: true,
                          });
                        }}
                        placeholder={t(
                          "form.enterDescription",
                          "Enter description"
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
                name="target_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.targetUrl", "Target URL")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={(value) => {
                          form.setValue(field.name, value, {
                            shouldDirty: true,
                          });
                        }}
                        placeholder={t(
                          "form.enterTargetUrl",
                          "Enter target URL"
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
                name="start_time"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.startTime", "Start Time")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        min={Number(new Date().toISOString().slice(0, 16))}
                        onValueChange={(value) => {
                          const timestamp = value
                            ? new Date(value).getTime()
                            : 0;
                          form.setValue(field.name, timestamp, {
                            shouldDirty: true,
                          });
                          const endTime = form.getValues("end_time");
                          if (endTime && timestamp > endTime) {
                            form.setValue("end_time", 0);
                          }
                        }}
                        placeholder={t(
                          "form.enterStartTime",
                          "Select start time"
                        )}
                        step="1"
                        type="datetime-local"
                        value={
                          field.value
                            ? new Date(field.value).toISOString().slice(0, 16)
                            : ""
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="end_time"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("form.endTime", "End Time")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        disabled={!startTime}
                        min={Number(
                          startTime
                            ? new Date(startTime).toISOString().slice(0, 16)
                            : new Date().toISOString().slice(0, 16)
                        )}
                        onValueChange={(value) => {
                          const timestamp = value
                            ? new Date(value).getTime()
                            : 0;
                          if (!startTime || timestamp < startTime) return;
                          form.setValue(field.name, timestamp, {
                            shouldDirty: true,
                          });
                        }}
                        placeholder={t("form.enterEndTime", "Select end time")}
                        step="1"
                        type="datetime-local"
                        value={
                          field.value
                            ? new Date(field.value).toISOString().slice(0, 16)
                            : ""
                        }
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
            )}
            {t("form.confirm", "Confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
