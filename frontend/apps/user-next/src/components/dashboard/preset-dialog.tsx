"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Badge } from "@workspace/ui/components/badge";
import { Button } from "@workspace/ui/components/button";
import { Checkbox } from "@workspace/ui/components/checkbox";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@workspace/ui/components/sheet";
import {
  Field,
  FieldDescription,
  FieldLabel,
} from "@workspace/ui/components/field";
import { Input } from "@workspace/ui/components/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@workspace/ui/components/select";
import { Spinner } from "@workspace/ui/components/spinner";
import { Switch } from "@workspace/ui/components/switch";
import {
  Check,
  CircleCheck,
  Copy,
  Plus,
  RadioTower,
  Star,
  Trash2,
} from "lucide-react";
import { QRCodeCanvas } from "qrcode.react";
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import {
  buildSubscriptionUrls,
  createAccessPreset,
  deleteAccessPreset,
  updateAccessPreset,
} from "@/lib/client-api";
import { copyText } from "@/lib/copy-text";
import type {
  AccessNode,
  AccessPreset,
  SiteConfig,
  Subscription,
} from "@/lib/models";
import { presetQrColor } from "@/lib/preset-utils";

type EditorDraft = {
  displayNames: Record<string, string>;
  enabled: boolean;
  isDefault: boolean;
  name: string;
  selected: Set<string>;
};

const NEW_PRESET = "new" as const;

function accessKey(nodeId: number, inboundAlias: string) {
  return `${nodeId}:${inboundAlias}`;
}

function makeDraft(nodes: AccessNode[], preset?: AccessPreset): EditorDraft {
  if (!preset) {
    return {
      displayNames: {},
      enabled: true,
      isDefault: false,
      name: "我的接入预设",
      selected: new Set(nodes.map((node) => node.key)),
    };
  }
  const selected = new Set<string>();
  const displayNames: Record<string, string> = {};
  for (const item of preset.items) {
    const key = accessKey(item.nodeId, item.inboundAlias);
    selected.add(key);
    displayNames[key] = item.displayName;
  }
  return {
    displayNames,
    enabled: preset.enabled,
    isDefault: preset.isDefault,
    name: preset.name,
    selected,
  };
}

export function PresetDialog({
  initialPresetId,
  nodes,
  onOpenChange,
  onPresetSelected,
  open,
  presets,
  config,
  subscription,
}: Readonly<{
  config?: SiteConfig;
  initialPresetId?: number;
  nodes: AccessNode[];
  onOpenChange: (open: boolean) => void;
  onPresetSelected: (id?: number) => void;
  open: boolean;
  presets: AccessPreset[];
  subscription: Subscription;
}>) {
  const queryClient = useQueryClient();
  const [activeId, setActiveId] = useState<number | typeof NEW_PRESET>(
    NEW_PRESET
  );
  const [draft, setDraft] = useState<EditorDraft>(() =>
    makeDraft(nodes, undefined)
  );
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteArmed, setDeleteArmed] = useState(false);
  const [selectedUrlIndex, setSelectedUrlIndex] = useState(0);
  const initializedOpen = useRef(false);

  const activePreset =
    activeId === NEW_PRESET
      ? undefined
      : presets.find((preset) => preset.id === activeId);
  const presetLinks = useMemo(
    () =>
      activePreset
        ? buildSubscriptionUrls(subscription, config, activePreset.key)
        : [],
    [activePreset, config, subscription]
  );
  const presetLink = presetLinks[selectedUrlIndex] || presetLinks[0] || "";
  const qrColor = activePreset
    ? presetQrColor(presets, activePreset.id)
    : "#111827";

  const protocolGroups = useMemo(() => {
    const groups = new Map<string, AccessNode[]>();
    for (const node of nodes) {
      const protocol = node.protocol || node.profileName || "其他协议";
      const group = groups.get(protocol);
      if (group) group.push(node);
      else groups.set(protocol, [node]);
    }
    return [...groups.entries()].map(([protocol, items]) => ({
      items,
      protocol,
    }));
  }, [nodes]);

  useEffect(() => {
    if (!open) {
      initializedOpen.current = false;
      return;
    }
    if (initializedOpen.current) return;
    initializedOpen.current = true;
    const initial = presets.find(
      (preset) => preset.id === initialPresetId && preset.enabled
    );
    if (initial) {
      setActiveId(initial.id);
      setDraft(makeDraft(nodes, initial));
    } else {
      setActiveId(NEW_PRESET);
      setDraft(makeDraft(nodes, undefined));
    }
    setDeleteArmed(false);
  }, [initialPresetId, nodes, open, presets]);

  useEffect(() => {
    setSelectedUrlIndex(0);
  }, [activeId]);

  useEffect(() => {
    if (selectedUrlIndex < presetLinks.length) return;
    setSelectedUrlIndex(0);
  }, [presetLinks.length, selectedUrlIndex]);

  const choosePreset = (preset?: AccessPreset) => {
    setActiveId(preset?.id ?? NEW_PRESET);
    setDraft(makeDraft(nodes, preset));
    setDeleteArmed(false);
  };

  const toggleItem = (key: string, checked: boolean) => {
    setDraft((current) => {
      const selected = new Set(current.selected);
      if (checked) selected.add(key);
      else selected.delete(key);
      return { ...current, selected };
    });
  };

  const toggleProtocol = (items: AccessNode[], checked: boolean) => {
    setDraft((current) => {
      const selected = new Set(current.selected);
      for (const node of items) {
        if (checked) selected.add(node.key);
        else selected.delete(node.key);
      }
      return { ...current, selected };
    });
  };

  const save = async () => {
    const isCreating = activeId === NEW_PRESET;
    const name = draft.name.trim();
    if (!name) {
      toast.error("请输入预设名称");
      return;
    }
    const items = nodes
      .filter((node) => draft.selected.has(node.key))
      .map((node) => ({
        display_name: draft.displayNames[node.key]?.trim() || undefined,
        inbound_alias: node.inboundAlias,
        node_id: node.id,
      }));
    if (!items.length) {
      toast.error("至少选择一个接入条目");
      return;
    }

    setSaving(true);
    try {
      const saved = isCreating
        ? await createAccessPreset({
            is_default: draft.isDefault,
            items,
            name,
            user_subscribe_id: subscription.id,
          })
        : await updateAccessPreset({
            enabled: draft.enabled,
            id: activeId,
            is_default: draft.isDefault ? true : undefined,
            items,
            name,
          });
      await queryClient.invalidateQueries({
        queryKey: ["alpha-subscription-scopes"],
      });
      onPresetSelected(saved.enabled ? saved.id : undefined);
      toast.success(isCreating ? "预设已创建" : "预设已更新");
      if (isCreating) {
        setActiveId(saved.id);
        setDraft(makeDraft(nodes, saved));
        setDeleteArmed(false);
        return;
      }
      onOpenChange(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (activeId === NEW_PRESET) return;
    if (!deleteArmed) {
      setDeleteArmed(true);
      return;
    }
    setDeleting(true);
    try {
      await deleteAccessPreset(activeId);
      await queryClient.invalidateQueries({
        queryKey: ["alpha-subscription-scopes"],
      });
      const next = presets.find(
        (preset) => preset.id !== activeId && preset.enabled
      );
      onPresetSelected(next?.id);
      choosePreset(next);
      toast.success("预设已删除");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "删除失败");
    } finally {
      setDeleting(false);
      setDeleteArmed(false);
    }
  };

  const copyPresetLink = async () => {
    try {
      await copyText(presetLink);
      toast.success("预设订阅链接已复制");
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : "预设订阅链接复制失败"
      );
    }
  };

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent
        className="w-full gap-0 overflow-hidden p-0 sm:max-w-[860px]"
        side="right"
      >
        <SheetHeader className="border-b px-6 py-5">
          <SheetTitle>接入预设</SheetTitle>
          <SheetDescription>
            一个预设可组合多个协议与节点，并为每个接入条目设置独立名称。
          </SheetDescription>
        </SheetHeader>

        <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)] gap-4 overflow-hidden p-6 md:grid-cols-[220px_minmax(0,1fr)]">
          <aside className="min-h-0 rounded-lg border bg-muted/20 p-2">
            <div className="flex items-center justify-between px-2 py-2 text-sm">
              <span className="font-medium">已保存</span>
              <Badge variant="outline">{presets.length} / 10</Badge>
            </div>
            <div className="flex max-h-36 gap-1 overflow-x-auto md:max-h-[58vh] md:flex-col md:overflow-y-auto">
              <Button
                className="h-auto min-w-48 justify-start gap-3 px-3 py-3 text-left md:min-w-0"
                disabled={presets.length >= 10}
                onClick={() => choosePreset(undefined)}
                type="button"
                variant={activeId === NEW_PRESET ? "secondary" : "ghost"}
              >
                <Plus aria-hidden="true" className="shrink-0" />
                <span className="min-w-0">
                  <strong className="block font-medium text-sm">
                    新建预设
                  </strong>
                  <small className="block truncate font-normal text-muted-foreground text-xs">
                    从全部接入条目开始
                  </small>
                </span>
              </Button>
              {presets.map((preset) => (
                <Button
                  className="h-auto min-w-48 justify-start gap-3 px-3 py-3 text-left md:min-w-0"
                  key={preset.id}
                  onClick={() => choosePreset(preset)}
                  type="button"
                  variant={activeId === preset.id ? "secondary" : "ghost"}
                >
                  {preset.isDefault ? (
                    <Star
                      aria-hidden="true"
                      className="shrink-0 fill-current text-amber-600"
                    />
                  ) : (
                    <CircleCheck aria-hidden="true" className="shrink-0" />
                  )}
                  <span className="min-w-0">
                    <strong className="block truncate font-medium text-sm">
                      {preset.name}
                    </strong>
                    <small className="block truncate font-normal text-muted-foreground text-xs">
                      {preset.items.length} 个条目 ·{" "}
                      {preset.isDefault
                        ? "默认"
                        : preset.enabled
                          ? "启用"
                          : "停用"}
                    </small>
                  </span>
                </Button>
              ))}
            </div>
          </aside>

          <div className="min-h-0 min-w-0 overflow-y-auto pr-1">
            <div className="grid gap-4 sm:grid-cols-2">
              <Field className="sm:col-span-2">
                <FieldLabel htmlFor="preset-name">预设名称</FieldLabel>
                <Input
                  id="preset-name"
                  maxLength={100}
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                  value={draft.name}
                />
              </Field>
              <Field
                className="rounded-lg border px-3 py-2"
                orientation="horizontal"
              >
                <div className="min-w-0 flex-1">
                  <FieldLabel htmlFor="preset-default">默认预设</FieldLabel>
                  <FieldDescription className="truncate text-xs">
                    复制主订阅链接时使用
                  </FieldDescription>
                </div>
                <Switch
                  checked={draft.isDefault}
                  id="preset-default"
                  onCheckedChange={(checked) => {
                    if (!checked && activePreset?.isDefault) {
                      toast.info("请将其他预设设为默认");
                      return;
                    }
                    setDraft((current) => ({
                      ...current,
                      enabled: checked ? true : current.enabled,
                      isDefault: checked,
                    }));
                  }}
                />
              </Field>
              <Field
                className="rounded-lg border px-3 py-2"
                orientation="horizontal"
              >
                <div className="min-w-0 flex-1">
                  <FieldLabel htmlFor="preset-enabled">启用预设</FieldLabel>
                  <FieldDescription className="truncate text-xs">
                    停用后原链接不再输出节点
                  </FieldDescription>
                </div>
                <Switch
                  checked={draft.enabled}
                  disabled={draft.isDefault}
                  id="preset-enabled"
                  onCheckedChange={(checked) =>
                    setDraft((current) => ({
                      ...current,
                      enabled: checked,
                    }))
                  }
                />
              </Field>
            </div>

            <div className="my-4 flex flex-wrap items-center gap-2 text-muted-foreground text-xs">
              <Badge variant="secondary">
                {draft.selected.size} 个接入条目
              </Badge>
              <Badge variant="outline">
                {
                  protocolGroups.filter((group) =>
                    group.items.some((node) => draft.selected.has(node.key))
                  ).length
                }{" "}
                种协议
              </Badge>
              <Button
                className="ml-auto h-auto px-1 py-0"
                onClick={() =>
                  setDraft((current) => ({
                    ...current,
                    selected: new Set(nodes.map((node) => node.key)),
                  }))
                }
                type="button"
                variant="link"
              >
                选择全部
              </Button>
            </div>

            {activePreset && presetLink ? (
              <section className="mb-4 grid gap-4 rounded-lg border p-4 sm:grid-cols-[minmax(0,1fr)_132px] sm:items-center">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <strong className="font-medium text-sm">预设订阅</strong>
                    <Badge
                      variant={activePreset.enabled ? "secondary" : "outline"}
                    >
                      {activePreset.enabled ? "可用" : "已停用"}
                    </Badge>
                    {activePreset.isDefault ? (
                      <Badge className="gap-1" variant="outline">
                        <Star className="fill-current" />
                        默认预设
                      </Badge>
                    ) : null}
                  </div>
                  <p className="mt-1 text-muted-foreground text-xs leading-5">
                    此链接和二维码仅输出当前预设选择的接入条目。
                  </p>
                  <div className="mt-3 grid gap-3">
                    {presetLinks.length > 1 ? (
                      <Field>
                        <FieldLabel htmlFor="preset-subscription-domain">
                          订阅域名
                        </FieldLabel>
                        <Select
                          onValueChange={(value) =>
                            setSelectedUrlIndex(Number(value))
                          }
                          value={String(selectedUrlIndex)}
                        >
                          <SelectTrigger
                            className="w-full"
                            id="preset-subscription-domain"
                          >
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectGroup>
                              {presetLinks.map((item, index) => (
                                <SelectItem key={item} value={String(index)}>
                                  {new URL(item).host}
                                </SelectItem>
                              ))}
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      </Field>
                    ) : null}
                    <Field>
                      <FieldLabel htmlFor="preset-subscription-link">
                        专属订阅链接
                      </FieldLabel>
                      <InputGroup>
                        <InputGroupInput
                          className="font-mono text-xs"
                          id="preset-subscription-link"
                          onFocus={(event) => event.currentTarget.select()}
                          readOnly
                          value={presetLink}
                        />
                        <InputGroupAddon align="inline-end">
                          <InputGroupButton
                            aria-label="复制预设订阅链接"
                            onClick={copyPresetLink}
                            size="icon-sm"
                            title="复制预设订阅链接"
                          >
                            <Copy />
                          </InputGroupButton>
                        </InputGroupAddon>
                      </InputGroup>
                    </Field>
                  </div>
                </div>
                <div className="justify-self-center text-center">
                  <div
                    aria-label={`${activePreset.name} 订阅二维码`}
                    className="rounded-lg border bg-white p-2"
                    role="img"
                  >
                    <QRCodeCanvas
                      bgColor="#ffffff"
                      fgColor={qrColor}
                      size={112}
                      value={presetLink}
                    />
                  </div>
                  <span className="mt-2 block text-muted-foreground text-xs">
                    扫码导入
                  </span>
                </div>
              </section>
            ) : null}

            <div className="max-h-[44vh] overflow-y-auto rounded-lg border">
              {protocolGroups.map((group) => {
                const selectedCount = group.items.filter((node) =>
                  draft.selected.has(node.key)
                ).length;
                const checked = selectedCount === group.items.length;
                const groupState =
                  selectedCount > 0 && !checked ? "indeterminate" : checked;
                return (
                  <section
                    className="border-b last:border-b-0"
                    key={group.protocol}
                  >
                    <header className="flex items-center gap-3 bg-muted/30 px-4 py-3">
                      <Checkbox
                        aria-label={`选择 ${group.protocol} 的全部接入条目`}
                        checked={groupState}
                        onCheckedChange={(value) =>
                          toggleProtocol(group.items, value === true)
                        }
                      />
                      <RadioTower
                        aria-hidden="true"
                        className="size-4 text-muted-foreground"
                      />
                      <strong className="font-medium text-sm">
                        {group.protocol}
                      </strong>
                      <span className="ml-auto text-muted-foreground text-xs tabular-nums">
                        {selectedCount} / {group.items.length}
                      </span>
                    </header>
                    <div className="divide-y">
                      {group.items.map((node) => {
                        const selected = draft.selected.has(node.key);
                        return (
                          <div
                            className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-3 px-4 py-3 sm:grid-cols-[auto_minmax(140px,0.8fr)_minmax(180px,1fr)]"
                            key={node.key}
                          >
                            <Checkbox
                              aria-label={`选择 ${node.name} ${node.profileName}`}
                              checked={selected}
                              onCheckedChange={(checkedValue) =>
                                toggleItem(node.key, checkedValue === true)
                              }
                            />
                            <div className="min-w-0">
                              <strong className="block truncate font-medium text-sm">
                                {node.name}
                              </strong>
                              <span className="block truncate text-muted-foreground text-xs">
                                {node.profileName || node.inboundAlias} ·{" "}
                                {node.country} {node.city}
                              </span>
                            </div>
                            <Input
                              className="col-span-2 sm:col-span-1"
                              aria-label={`${node.name} ${node.profileName || node.inboundAlias} 的显示名称`}
                              disabled={!selected}
                              maxLength={100}
                              onChange={(event) => {
                                const value = event.target.value;
                                setDraft((current) => ({
                                  ...current,
                                  displayNames: {
                                    ...current.displayNames,
                                    [node.key]: value,
                                  },
                                }));
                              }}
                              placeholder="使用默认节点名称"
                              value={draft.displayNames[node.key] || ""}
                            />
                          </div>
                        );
                      })}
                    </div>
                  </section>
                );
              })}
            </div>
          </div>
        </div>

        <SheetFooter className="flex-row items-center gap-2 border-t px-6 py-4 sm:justify-between">
          <div className="mr-auto">
            {activeId !== NEW_PRESET ? (
              <Button
                disabled={deleting || saving}
                onClick={remove}
                variant="destructive"
              >
                {deleting ? <Spinner /> : <Trash2 />}
                {deleteArmed ? "再次点击确认删除" : "删除预设"}
              </Button>
            ) : null}
          </div>
          <div className="flex w-full gap-2 sm:w-auto">
            <Button onClick={() => onOpenChange(false)} variant="outline">
              取消
            </Button>
            <Button
              className="flex-1"
              disabled={saving || deleting}
              onClick={save}
            >
              {saving ? <Spinner /> : <Check />}
              {activeId === NEW_PRESET ? "创建预设" : "保存修改"}
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
