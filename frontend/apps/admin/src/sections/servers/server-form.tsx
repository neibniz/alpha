"use client";

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
import { EnhancedInput } from "@workspace/ui/composed/enhanced-input";
import { Icon } from "@workspace/ui/composed/icon";
import { type ReactNode, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

const serverFormSchema = z.object({
  name: z.string().trim().min(1),
  address: z.string().trim().min(1),
  country: z.string().optional(),
  city: z.string().optional(),
});

type ServerFormValues = z.infer<typeof serverFormSchema>;

export default function ServerForm(props: {
  trigger?: ReactNode;
  title: string;
  loading?: boolean;
  initialValues?: Partial<API.ServerListItem>;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onSubmit: (values: ServerFormValues) => Promise<boolean> | boolean;
}) {
  const {
    trigger,
    title,
    loading,
    initialValues,
    open: controlledOpen,
    onOpenChange,
    onSubmit,
  } = props;
  const { t } = useTranslation("servers");
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const open = controlledOpen ?? uncontrolledOpen;
  const setOpen = (nextOpen: boolean) => {
    onOpenChange?.(nextOpen);
    if (controlledOpen === undefined) {
      setUncontrolledOpen(nextOpen);
    }
  };

  const form = useForm<ServerFormValues>({
    resolver: zodResolver(serverFormSchema),
    defaultValues: {
      name: "",
      address: "",
      country: "",
      city: "",
    },
  });

  useEffect(() => {
    if (!initialValues) return;
    form.reset({
      name: (initialValues.name as string) || "",
      address: (initialValues.address as string) || "",
      country: (initialValues.country as string) || "",
      city: (initialValues.city as string) || "",
    });
  }, [form, initialValues]);

  async function handleSubmit(values: ServerFormValues) {
    const ok = await onSubmit(values);
    if (ok) {
      form.reset();
      setOpen(false);
    }
  }

  return (
    <Sheet onOpenChange={setOpen} open={open}>
      {trigger ? (
        <SheetTrigger asChild>
          <Button
            onClick={() => {
              if (!initialValues) {
                form.reset({
                  name: "",
                  address: "",
                  country: "",
                  city: "",
                });
              }
              setOpen(true);
            }}
          >
            {trigger}
          </Button>
        </SheetTrigger>
      ) : null}
      <SheetContent className="w-[560px] max-w-full gap-0">
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
          <SheetDescription>
            {t(
              "formDescription",
              "Maintain the server name, address, and location details."
            )}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="h-[calc(100dvh-48px-36px-36px-env(safe-area-inset-top))]">
          <Form {...form}>
            <form className="grid grid-cols-1 gap-4 px-6 pt-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("name", "Name")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        {...field}
                        onValueChange={(v) => field.onChange(v)}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="address"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("address", "Address")}</FormLabel>
                    <FormControl>
                      <EnhancedInput
                        {...field}
                        onValueChange={(v) => field.onChange(v)}
                        placeholder={t("address_placeholder", "Server address")}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <FormField
                  control={form.control}
                  name="country"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("country", "Country")}</FormLabel>
                      <FormControl>
                        <EnhancedInput
                          {...field}
                          onValueChange={(v) => field.onChange(v)}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="city"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("city", "City")}</FormLabel>
                      <FormControl>
                        <EnhancedInput
                          {...field}
                          onValueChange={(v) => field.onChange(v)}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </form>
          </Form>
        </ScrollArea>
        <SheetFooter className="flex-row justify-end gap-2 pt-3">
          <Button
            disabled={loading}
            onClick={() => setOpen(false)}
            variant="outline"
          >
            {t("cancel", "Cancel")}
          </Button>
          <Button
            disabled={loading}
            onClick={form.handleSubmit(handleSubmit, (errors) => {
              const key = Object.keys(errors)[0] as keyof typeof errors;
              if (key) toast.error(String(errors[key]?.message));
              return false;
            })}
          >
            {loading && (
              <Icon className="mr-2 animate-spin" icon="mdi:loading" />
            )}
            {t("confirm", "Confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
