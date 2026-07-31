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
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@workspace/ui/components/item";
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
import {
  getAuthMethodConfig,
  updateAuthMethodConfig,
} from "@workspace/ui/services/admin/authMethod";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";
import { createGithubSchema, type GithubFormData } from "./github-schema";

const githubConfigSchema = z.object({
  client_id: z.string().optional(),
  client_secret: z.string().optional(),
  client_secret_configured: z.boolean().optional(),
  redirect_url: z.string().optional(),
});

export default function GithubForm() {
  const { t } = useTranslation("auth-control");
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const githubSchema = useMemo(
    () =>
      createGithubSchema({
        clientIdRequired: t("github.clientIdRequired", "Client ID is required"),
        clientSecretRequired: t(
          "github.clientSecretRequired",
          "Client Secret is required"
        ),
        callbackUrlRequired: t(
          "github.callbackUrlRequired",
          "Callback URL is required"
        ),
        callbackUrlInvalid: t(
          "github.callbackUrlInvalid",
          "Use the user frontend origin root, without a path, query, or fragment"
        ),
      }),
    [t]
  );

  const { data, refetch } = useQuery({
    queryKey: ["getAuthMethodConfig", "github"],
    queryFn: async () => {
      const { data } = await getAuthMethodConfig({
        method: "github",
      });

      return data.data;
    },
    enabled: open,
  });

  const form = useForm<GithubFormData>({
    resolver: zodResolver(githubSchema),
    defaultValues: {
      enabled: false,
      client_id: "",
      client_secret: "",
      client_secret_configured: false,
      redirect_url: "",
    },
  });

  useEffect(() => {
    if (data) {
      const config = githubConfigSchema.catch({}).parse(data.config);
      form.reset({
        enabled: data.enabled,
        client_id: config.client_id || "",
        client_secret: "",
        client_secret_configured:
          config.client_secret_configured || Boolean(config.client_secret),
        redirect_url: config.redirect_url || "",
      });
    }
  }, [data, form]);

  async function onSubmit(values: GithubFormData) {
    setLoading(true);
    try {
      await updateAuthMethodConfig({
        id: data?.id ?? 0,
        method: "github",
        enabled: values.enabled,
        config: {
          client_id: values.client_id ?? "",
          client_secret: values.client_secret ?? "",
          redirect_url: values.redirect_url ?? "",
        },
      });
      toast.success(t("common.saveSuccess", "Saved successfully"));
      refetch();
      setOpen(false);
    } catch (_error) {
      toast.error(t("common.saveFailed", "Save failed"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      <SheetTrigger asChild>
        <Item asChild className="w-full cursor-pointer border-0 p-0">
          <button type="button">
            <ItemMedia variant="icon">
              <Icon icon="mdi:github" />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>{t("github.title", "GitHub Sign-In")}</ItemTitle>
              <ItemDescription>
                {t(
                  "github.description",
                  "Authenticate users with GitHub accounts"
                )}
              </ItemDescription>
            </ItemContent>
            <ItemActions>
              <Icon icon="mdi:chevron-right" />
            </ItemActions>
          </button>
        </Item>
      </SheetTrigger>
      <SheetContent className="w-[500px] max-w-full md:max-w-screen-md">
        <SheetHeader>
          <SheetTitle>{t("github.title", "GitHub Sign-In")}</SheetTitle>
          <SheetDescription>
            {t("github.description", "Authenticate users with GitHub accounts")}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-24px-env(safe-area-inset-top))] px-6">
          <Form {...form}>
            <form
              className="space-y-2 pt-4"
              id="github-form"
              onSubmit={form.handleSubmit(onSubmit)}
            >
              <FormField
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("github.enable", "Enable")}</FormLabel>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        className="!mt-0 float-end"
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "github.enableDescription",
                        "When enabled, users can sign in with their GitHub account"
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="client_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("github.clientId", "Client ID")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={field.onChange}
                        placeholder="e.g., Iv1.1234567890abcdef"
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "github.clientIdDescription",
                        "GitHub OAuth App Client ID, available from GitHub Developer Settings"
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="client_secret"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("github.clientSecret", "Client Secret")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={field.onChange}
                        placeholder={t(
                          "github.clientSecretPlaceholder",
                          "Enter the Client Secret"
                        )}
                        type="password"
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {form.watch("client_secret_configured")
                        ? t(
                            "github.clientSecretConfiguredDescription",
                            "Leave blank to keep the configured Client Secret"
                          )
                        : t(
                            "github.clientSecretDescription",
                            "GitHub OAuth App Client Secret, available from GitHub Developer Settings"
                          )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="redirect_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t("github.callbackUrl", "Authorization callback URL")}
                    </FormLabel>
                    <FormControl>
                      <EnhancedInput
                        onValueChange={field.onChange}
                        placeholder="https://user.example.com/"
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        "github.callbackUrlDescription",
                        "Use the user frontend origin root and enter the same URL in the GitHub OAuth App"
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
          <Button disabled={loading} form="github-form" type="submit">
            {loading && (
              <Icon className="mr-2 animate-spin" icon="mdi:loading" />
            )}
            {t("common.save", "Save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
