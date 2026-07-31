"use client";

import {
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@workspace/ui/components/field";
import {
  RadioGroup,
  RadioGroupItem,
} from "@workspace/ui/components/radio-group";
import { CreditCard, WalletCards } from "lucide-react";
import type { PaymentMethod } from "@/lib/models";

export function PaymentMethodSelector({
  idPrefix,
  methods,
  onChange,
  value,
}: Readonly<{
  idPrefix: string;
  methods: PaymentMethod[];
  onChange: (value: number) => void;
  value: number;
}>) {
  return (
    <FieldSet>
      <FieldLegend>支付方式</FieldLegend>
      <RadioGroup
        className="grid gap-2 sm:grid-cols-2"
        onValueChange={(next) => onChange(Number(next))}
        value={String(value)}
      >
        {methods.map((method) => {
          const Icon =
            method.platform.toLowerCase() === "balance"
              ? WalletCards
              : CreditCard;
          const id = `${idPrefix}-${method.id}`;
          return (
            <FieldLabel
              className="flex cursor-pointer items-center gap-3 rounded-lg border p-3"
              htmlFor={id}
              key={method.id}
            >
              <RadioGroupItem id={id} value={String(method.id)} />
              <Icon
                aria-hidden="true"
                className="size-4 text-muted-foreground"
              />
              <span className="min-w-0">
                <strong className="block font-medium text-sm">
                  {method.name}
                </strong>
                <small className="block truncate font-normal text-muted-foreground text-xs">
                  {method.description || method.platform}
                </small>
              </span>
            </FieldLabel>
          );
        })}
      </RadioGroup>
    </FieldSet>
  );
}
