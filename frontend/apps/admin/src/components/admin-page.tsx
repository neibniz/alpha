import { Item, ItemGroup, ItemSeparator } from "@workspace/ui/components/item";
import { cn } from "@workspace/ui/lib/utils";
import { Fragment, type ReactNode } from "react";
import type React from "react";

export function AdminPage({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("flex min-w-0 flex-col gap-5", className)}
      data-slot="admin-page"
      {...props}
    />
  );
}

export function AdminPageHeader({
  actions,
  description,
  eyebrow,
  title,
}: {
  actions?: React.ReactNode;
  description?: React.ReactNode;
  eyebrow?: React.ReactNode;
  title: React.ReactNode;
}) {
  return (
    <header className="admin-page-header flex min-w-0 flex-col gap-4 md:flex-row md:items-center md:justify-between">
      <div className="min-w-0">
        {eyebrow ? <span className="sr-only">{eyebrow}</span> : null}
        <h1 className="text-balance font-semibold text-2xl leading-tight">
          {title}
        </h1>
        {description ? (
          <p className="mt-1.5 max-w-3xl text-muted-foreground text-sm leading-relaxed">
            {description}
          </p>
        ) : null}
      </div>
      {actions ? (
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {actions}
        </div>
      ) : null}
    </header>
  );
}

export function AdminSectionHeader({
  actions,
  description,
  title,
}: {
  actions?: React.ReactNode;
  description?: React.ReactNode;
  title: React.ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <h2 className="font-semibold text-base">{title}</h2>
        {description ? (
          <p className="mt-1 text-muted-foreground text-sm">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="shrink-0">{actions}</div> : null}
    </div>
  );
}

export function AdminSettingsGroup({
  children,
  description,
  title,
}: {
  children: React.ReactNode;
  description?: React.ReactNode;
  title: React.ReactNode;
}) {
  return (
    <section className="admin-settings-section grid gap-5 lg:grid-cols-[minmax(12rem,0.28fr)_minmax(0,0.72fr)]">
      <div>
        <h2 className="font-semibold text-sm">{title}</h2>
        {description ? (
          <p className="mt-1.5 max-w-sm text-muted-foreground text-xs leading-5">
            {description}
          </p>
        ) : null}
      </div>
      <ItemGroup className="min-w-0 border-y">
        {Array.isArray(children) ? (
          children.map((child, index) => (
            <Fragment key={index}>
              {index > 0 ? <ItemSeparator /> : null}
              <Item className="rounded-none border-0" size="default">
                {child as ReactNode}
              </Item>
            </Fragment>
          ))
        ) : (
          <Item className="rounded-none border-0" size="default">
            {children}
          </Item>
        )}
      </ItemGroup>
    </section>
  );
}
