import { useFormCloseGuard } from "@/components/use-form-close-guard";
import { AdminForm } from "@/components/admin-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@workspace/ui/components/button";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Skeleton } from "@workspace/ui/components/skeleton";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@workspace/ui/components/tabs";
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
  SheetDescription,
  SheetTitle,
  SheetTrigger,
} from "@workspace/ui/components/sheet";
import { Textarea } from "@workspace/ui/components/textarea";
import { JSONEditor } from "@workspace/ui/composed/editor/json";
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import {
  Globe,
  ChevronRight,
  LoaderCircle,
  RefreshCw,
  AlertCircle,
  Palette,
  Code2,
} from "lucide-react";
import { UploadImage } from "@workspace/ui/composed/upload-image";
import {
  getSiteConfig,
  updateSiteConfig,
} from "@workspace/ui/services/admin/system";
import { useEffect, useId, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

const siteSchema = z.object({
  site_logo: z.string().optional(),
  site_name: z.string().min(1),
  site_desc: z.string().optional(),
  keywords: z.string().optional(),
  custom_html: z.string().optional(),
  host: z.string().optional(),
  custom_data: z.string().optional(),
});

type SiteFormData = z.infer<typeof siteSchema>;

export default function SiteConfig() {
  const { t } = useTranslation("system");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tab, setTab] = useState("identity");
  const formId = useId();

  const { data, refetch, isFetching, isError } = useQuery({
    queryKey: ["getSiteConfig"],
    queryFn: async () => {
      const { data } = await getSiteConfig();
      return siteSchema.extend({ site_name: z.string() }).parse(data.data);
    },
    enabled: open,
  });

  const form = useForm<SiteFormData>({
    resolver: zodResolver(siteSchema),
    defaultValues: {
      site_logo: "",
      site_name: "",
      site_desc: "",
      keywords: "",
      custom_html: "",
      host: "",
      custom_data: "",
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
      else setTab("identity");
      setOpen(next);
    },
  });

  async function onSubmit(values: SiteFormData) {
    setLoading(true);
    try {
      await updateSiteConfig(values as API.SiteConfig);
      toast.success(t("common.saveSuccess", "Save Successful"));
      form.reset(values);
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
              <Globe className="size-5 text-muted-foreground" />
            </div>
            <div className="flex-1">
              <p className="font-medium">
                {t("site.title", "Site Configuration")}
              </p>
              <p className="text-muted-foreground text-sm">
                {t(
                  "site.description",
                  "Configure basic site information, logo, domain and other settings"
                )}
              </p>
            </div>
          </div>
          <ChevronRight className="size-4 text-muted-foreground" />
        </Button>
      </SheetTrigger>
      <SheetContent className="flex w-[680px] max-w-full flex-col gap-0 p-0 sm:max-w-2xl">
        <SheetHeader className="border-b px-6 py-5 pr-12">
          <SheetTitle>{t("site.title", "Site Configuration")}</SheetTitle>
          <SheetDescription>
            {t(
              "site.editorDescription",
              "Site identity, public metadata and custom extensions."
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="min-h-0 flex-1 px-6">
          {!data && isFetching ? (
            <div aria-busy="true" className="flex flex-col gap-5 py-6">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
          ) : isError ? (
            <Alert className="my-6" variant="destructive">
              <AlertCircle />
              <AlertTitle>
                {t("site.loadFailed", "Unable to load site settings")}
              </AlertTitle>
              <AlertDescription>
                <Button
                  disabled={isFetching}
                  onClick={() => refetch()}
                  size="sm"
                  variant="outline"
                >
                  <RefreshCw />
                  {t("site.retry", "Retry")}
                </Button>
              </AlertDescription>
            </Alert>
          ) : data ? (
            <Form {...form}>
              <AdminForm
                className="py-6"
                id={formId}
                onSubmit={form.handleSubmit(onSubmit, (errors) =>
                  setTab(
                    ["site_logo", "site_name", "site_desc", "keywords"].some(
                      (key) => key in errors
                    )
                      ? "identity"
                      : "extensions"
                  )
                )}
              >
                <fieldset
                  className="min-w-0"
                  disabled={
                    loading || form.formState.isSubmitting || isFetching
                  }
                >
                  <Tabs value={tab} onValueChange={setTab}>
                    <TabsList className="grid w-full grid-cols-2">
                      <TabsTrigger value="identity">
                        <Palette />
                        {t("site.identityTab", "Identity")}
                      </TabsTrigger>
                      <TabsTrigger value="extensions">
                        <Code2 />
                        {t("site.extensionsTab", "Domain & extensions")}
                      </TabsTrigger>
                    </TabsList>
                    <TabsContent
                      className="flex flex-col gap-6 pt-5"
                      value="identity"
                    >
                      <FormField
                        control={form.control}
                        name="site_logo"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>{t("site.logo", "Site Logo")}</FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "site.logoPlaceholder",
                                  "Enter the URL of the logo, without ending with '/'"
                                )}
                                suffix={
                                  <UploadImage
                                    className="h-9 rounded-none border-none bg-muted px-2"
                                    onChange={(value) => {
                                      field.onChange(value);
                                    }}
                                  />
                                }
                                value={field.value}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.logoDescription",
                                "Used for displaying the logo in designated locations"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="site_name"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("site.siteName", "Site Name")}
                            </FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "site.siteNamePlaceholder",
                                  "Enter site name"
                                )}
                                value={field.value}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.siteNameDescription",
                                "Used for displaying the site name in designated locations"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="site_desc"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("site.siteDesc", "Site Description")}
                            </FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "site.siteDescPlaceholder",
                                  "Enter site description"
                                )}
                                value={field.value}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.siteDescDescription",
                                "Used for displaying the site description in designated locations"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="keywords"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("site.keywords", "Keywords")}
                            </FormLabel>
                            <FormControl>
                              <EnhancedInput
                                onValueChange={field.onChange}
                                placeholder={t(
                                  "site.keywordsPlaceholder",
                                  "keyword1, keyword2, keyword3"
                                )}
                                value={field.value}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.keywordsDescription",
                                "Used for SEO purposes"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </TabsContent>
                    <TabsContent
                      className="flex flex-col gap-6 pt-5"
                      value="extensions"
                    >
                      <FormField
                        control={form.control}
                        name="custom_html"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("site.customHtml", "Custom HTML")}
                            </FormLabel>
                            <FormControl>
                              <Textarea
                                className="min-h-40 font-mono text-xs"
                                placeholder={t(
                                  "site.customHtmlDescription",
                                  "Custom HTML code to be injected into the bottom of the site's body tag"
                                )}
                                {...field}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.customHtmlDescription",
                                "Custom HTML code to be injected into the bottom of the site's body tag"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="host"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("site.siteDomain", "Site Domain")}
                            </FormLabel>
                            <FormControl>
                              <Textarea
                                className="min-h-24"
                                placeholder={`${t("site.siteDomainPlaceholder", "Please enter the domain address. For multiple domains, please enter one per line.")}\nexample.com\nwww.example.com`}
                                {...field}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.siteDomainDescription",
                                "Domain address of the current website, e.g., used in emails"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <FormField
                        control={form.control}
                        name="custom_data"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>
                              {t("site.customData", "Custom Data")}
                            </FormLabel>
                            <FormControl>
                              <JSONEditor
                                onChange={field.onChange}
                                schema={{
                                  type: "object",
                                  additionalProperties: true,
                                  properties: {
                                    website: {
                                      type: "string",
                                      title: t(
                                        "site.customDataFields.website",
                                        "Website"
                                      ),
                                    },
                                    contacts: {
                                      type: "object",
                                      title: t(
                                        "site.customDataFields.contacts",
                                        "Contacts"
                                      ),
                                      additionalProperties: true,
                                      properties: {
                                        email: {
                                          type: "string",
                                          title: t(
                                            "site.customDataFields.email",
                                            "Email"
                                          ),
                                        },
                                        telephone: {
                                          type: "string",
                                          title: t(
                                            "site.customDataFields.telephone",
                                            "Telephone"
                                          ),
                                        },
                                        address: {
                                          type: "string",
                                          title: t(
                                            "site.customDataFields.address",
                                            "Address"
                                          ),
                                        },
                                      },
                                    },
                                    community: {
                                      type: "object",
                                      title: t(
                                        "site.customDataFields.community",
                                        "Community"
                                      ),
                                      additionalProperties: true,
                                      properties: {
                                        telegram: {
                                          type: "string",
                                          title: "Telegram",
                                        },
                                        twitter: {
                                          type: "string",
                                          title: "Twitter",
                                        },
                                        discord: {
                                          type: "string",
                                          title: "Discord",
                                        },
                                        instagram: {
                                          type: "string",
                                          title: "Instagram",
                                        },
                                        linkedin: {
                                          type: "string",
                                          title: "Linkedin",
                                        },
                                        facebook: {
                                          type: "string",
                                          title: "Facebook",
                                        },
                                        github: {
                                          type: "string",
                                          title: "Github",
                                        },
                                      },
                                    },
                                  },
                                }}
                                value={field.value}
                              />
                            </FormControl>
                            <FormDescription>
                              {t(
                                "site.customDataDescription",
                                "Custom data for website customization"
                              )}
                            </FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </TabsContent>
                  </Tabs>
                </fieldset>
              </AdminForm>
            </Form>
          ) : null}
        </ScrollArea>
        <SheetFooter className="flex-row items-center justify-between gap-3 border-t px-6 py-4">
          <span aria-live="polite" className="text-muted-foreground text-xs">
            {form.formState.isDirty
              ? t("site.unsaved", "Unsaved changes")
              : t("site.saved", "No unsaved changes")}
          </span>
          <div className="flex shrink-0 gap-2">
            <Button
              disabled={loading}
              onClick={() => closeGuard.requestOpenChange(false)}
              variant="outline"
            >
              {t("common.cancel", "Cancel")}
            </Button>
            <Button
              disabled={
                loading ||
                form.formState.isSubmitting ||
                isFetching ||
                isError ||
                !data ||
                !form.formState.isDirty
              }
              form={formId}
              type="submit"
            >
              {loading && <LoaderCircle className="animate-spin" />}
              {t("common.save", "Save Settings")}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
