import {
  demoProducts,
  demoScopes,
  demoSubscriptions,
  makeDemoTraffic,
} from "@/lib/demo-data";
import { bitsPerSecondToMbps, normalizeTimestamp } from "@/lib/format";
import type {
  AccessPreset,
  Announcement,
  CheckoutResult,
  CurrentUser,
  MembershipCard,
  MembershipQuote,
  OrderProduct,
  PaymentMethod,
  Product,
  PurchaseQuote,
  RenewalQuote,
  ResetTrafficQuote,
  SiteConfig,
  Subscription,
  SubscriptionScope,
  TrafficPeriod,
  TrafficSeries,
  UserDocument,
  UserOrder,
  UserTicket,
  UserTicketDetail,
  UserAuthMethod,
  WalletLog,
} from "@/lib/models";
import { isClientDemoMode } from "@/lib/runtime";
export { createOrderRequestId } from "@/lib/request-id";
export {
  buildSubscriptionUrl,
  buildSubscriptionUrls,
} from "@/lib/subscription-url";

type ApiEnvelope<T> = {
  code?: number;
  data?: T;
  message?: string;
  msg?: string;
};

type ListPayload<T> = T[] | { list?: T[] };

export type ApiRequester = <T>(
  path: string,
  init?: RequestInit
) => Promise<T | undefined>;

function listFrom<T>(payload: ListPayload<T> | undefined) {
  if (Array.isArray(payload)) return payload;
  return payload?.list || [];
}

const AUTH_ERROR_CODES = new Set([40_002, 40_003, 40_004]);

export class ApiError extends Error {
  code?: number;
  status: number;

  constructor(message: string, status: number, code?: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

export function isMembershipRequiredError(error: unknown) {
  return error instanceof ApiError && error.code === 60_008;
}

async function apiRequest<T>(path: string, init?: RequestInit) {
  const response = await fetch(`/api/backend/${path.replace(/^\/+/, "")}`, {
    cache: "no-store",
    credentials: "same-origin",
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.headers || {}),
    },
  });
  const raw = await response.text();
  let payload: ApiEnvelope<T> = {};
  try {
    payload = raw ? (JSON.parse(raw) as ApiEnvelope<T>) : {};
  } catch {
    throw new ApiError("服务响应格式无效", response.status);
  }
  if (
    !response.ok ||
    (payload.code !== undefined && payload.code !== 0 && payload.code !== 200)
  ) {
    if (
      typeof window !== "undefined" &&
      (response.status === 401 ||
        (payload.code !== undefined && AUTH_ERROR_CODES.has(payload.code)))
    ) {
      const next = `${window.location.pathname}${window.location.search}`;
      window.location.assign(
        `/login?reason=session-expired&next=${encodeURIComponent(next)}`
      );
    }
    throw new ApiError(
      payload.message || payload.msg || "请求失败",
      response.status,
      payload.code
    );
  }
  return payload.data;
}

function cloneSubscription(subscription: Subscription): Subscription {
  return { ...subscription, discounts: [...subscription.discounts] };
}

function cloneScope(scope: SubscriptionScope): SubscriptionScope {
  return {
    ...scope,
    nodes: scope.nodes.map((node) => ({ ...node, tags: [...node.tags] })),
    presets: scope.presets.map((preset) => ({
      ...preset,
      items: preset.items.map((item) => ({ ...item })),
    })),
    profiles: scope.profiles.map((profile) => ({ ...profile })),
  };
}

const demoSubscriptionState = demoSubscriptions.map(cloneSubscription);
const demoScopeState = demoScopes.map(cloneScope);
let demoPresetSequence =
  Math.max(
    0,
    ...demoScopeState.flatMap((scope) =>
      scope.presets.map((preset) => preset.id)
    )
  ) + 1;
let demoOrderSequence = 1;
type DemoOrderType =
  | "membership"
  | "purchase"
  | "recharge"
  | "renewal"
  | "reset";

let demoMembershipExpiredAt = Date.now() + 120 * 24 * 60 * 60 * 1000;
const demoOrders = new Map<
  string,
  {
    createdAt: number;
    payment: number;
    quantity: number;
    status: number;
    subscriptionId: number;
    type: DemoOrderType;
  }
>();

function demoOrderType(type: DemoOrderType) {
  if (type === "renewal") return 2;
  if (type === "reset") return 3;
  if (type === "recharge") return 4;
  if (type === "membership") return 5;
  return 1;
}

function demoOrderName(type: DemoOrderType) {
  if (type === "reset") return "流量重置";
  if (type === "recharge") return "余额充值";
  if (type === "renewal") return "订阅续费";
  if (type === "membership") return "年度会员";
  return "订阅套餐";
}

type RawSubscription = {
  can_renew: boolean;
  can_reset_traffic: boolean;
  created_at?: number;
  download: number;
  effective_status: number;
  expire_time: number;
  id: number;
  product_name: string;
  remaining_traffic: number;
  renewal_reason: string;
  reset_time: number;
  reset_traffic_reason: string;
  short: string;
  status: number;
  status_reason: string;
  subscribe: {
    allow_renewal: boolean;
    allow_reset_traffic: boolean;
    device_limit: number;
    discount: Array<{ discount: number; quantity: number }>;
    id: number;
    name: string;
    replacement: number;
    show_original_price: boolean;
    speed_limit: number;
    unit_price: number;
    unit_time: string;
  };
  token: string;
  traffic: number;
  upload: number;
};

export function subscriptionStatusFromCode(
  status: number
): Subscription["status"] {
  if (status === 0) return "pending";
  if (status === 1) return "active";
  if (status === 2) return "exhausted";
  if (status === 3) return "expired";
  if (status === 4) return "deducted";
  if (status === 5) return "stopped";
  return "unknown";
}

export function getSubscriptions(): Promise<Subscription[]> {
  return getSubscriptionsWith(apiRequest);
}

export async function getSubscriptionsWith(
  request: ApiRequester
): Promise<Subscription[]> {
  if (isClientDemoMode()) return demoSubscriptionState.map(cloneSubscription);
  const data = await request<ListPayload<RawSubscription>>(
    "v1/public/user/subscribe"
  );
  return listFrom(data).map((item) => ({
    allowRenewal: item.subscribe.allow_renewal,
    allowResetTraffic: item.subscribe.allow_reset_traffic,
    canRenew: item.can_renew,
    canResetTraffic: item.can_reset_traffic,
    createdAt: item.created_at ? normalizeTimestamp(item.created_at) : 0,
    deviceLimit: item.subscribe.device_limit,
    discounts: item.subscribe.discount || [],
    downloadBytes: item.download,
    expiresAt: item.expire_time ? normalizeTimestamp(item.expire_time) : 0,
    id: item.id,
    name: item.product_name || item.subscribe.name,
    planName: item.subscribe.name,
    productId: item.subscribe.id,
    resetAt: item.reset_time ? normalizeTimestamp(item.reset_time) : 0,
    resetPrice: item.subscribe.replacement,
    renewalReason: item.renewal_reason || "",
    remainingTrafficBytes: item.remaining_traffic,
    resetTrafficReason: item.reset_traffic_reason || "",
    short: item.short,
    showOriginalPrice: item.subscribe.show_original_price,
    speedLimitMbps: bitsPerSecondToMbps(item.subscribe.speed_limit),
    status: subscriptionStatusFromCode(item.effective_status ?? item.status),
    statusReason: item.status_reason || "unknown",
    token: item.token,
    trafficBytes: item.traffic,
    unitPrice: item.subscribe.unit_price,
    unitTime: item.subscribe.unit_time,
    uploadBytes: item.upload,
  }));
}

type RawPreset = {
  enabled: boolean;
  id: number;
  is_default: boolean;
  items: Array<{
    display_name?: string;
    inbound_alias: string;
    node_id: number;
  }>;
  mode: string;
  name: string;
  preset_key: string;
  profile_key: string;
};

type RawScope = {
  id: number;
  nodes: Array<{
    city: string;
    country: string;
    id: number;
    inbound_alias: string;
    item_key: string;
    name: string;
    profile_key: string;
    profile_name: string;
    protocol: string;
    server_name: string;
    tags: string[];
  }>;
  presets: RawPreset[];
  profiles: Array<{
    key: string;
    name: string;
    node_count: number;
    protocol: string;
    security: string;
    sort: number;
    transport: string;
  }>;
};

function mapPreset(preset: RawPreset): AccessPreset {
  return {
    enabled: preset.enabled,
    id: preset.id,
    isDefault: preset.is_default,
    items: (preset.items || []).map((item) => ({
      displayName: item.display_name || "",
      inboundAlias: item.inbound_alias,
      nodeId: item.node_id,
    })),
    key: preset.preset_key,
    mode: preset.mode,
    name: preset.name,
    profileKey: preset.profile_key,
  };
}

export function getSubscriptionScopes(): Promise<SubscriptionScope[]> {
  return getSubscriptionScopesWith(apiRequest);
}

export async function getSubscriptionScopesWith(
  request: ApiRequester
): Promise<SubscriptionScope[]> {
  if (isClientDemoMode()) return demoScopeState.map(cloneScope);
  const data = await request<ListPayload<RawScope>>(
    "v1/public/subscribe/node/list"
  );
  return listFrom(data).map((scope) => ({
    id: scope.id,
    nodes: (scope.nodes || []).map((node) => ({
      city: node.city,
      country: node.country,
      id: node.id,
      inboundAlias: node.inbound_alias,
      itemKey: node.item_key,
      key: `${node.id}:${node.inbound_alias}`,
      name: node.name,
      profileKey: node.profile_key,
      profileName: node.profile_name,
      protocol: node.protocol,
      serverName: node.server_name,
      tags: node.tags || [],
    })),
    presets: (scope.presets || []).map(mapPreset),
    profiles: (scope.profiles || []).map((profile) => ({
      key: profile.key,
      name: profile.name,
      nodeCount: profile.node_count,
      protocol: profile.protocol,
      security: profile.security,
      sort: profile.sort,
      transport: profile.transport,
    })),
  }));
}

type RawTraffic = {
  interval_seconds: number;
  list: Array<{ download: number; timestamp: number; upload: number }>;
};

export async function getTrafficSeries(
  id: number,
  period: TrafficPeriod
): Promise<TrafficSeries> {
  if (isClientDemoMode()) return makeDemoTraffic(id, period);
  const params = new URLSearchParams({ period, user_subscribe_id: String(id) });
  const data = await apiRequest<RawTraffic>(
    `v1/public/user/subscribe_traffic?${params}`
  );
  const intervalSeconds = Math.max(1, data?.interval_seconds || 1);
  return {
    intervalSeconds,
    points: (data?.list || []).map((point) => ({
      downloadMbps: (point.download * 8) / intervalSeconds / 1_000_000,
      timestamp: normalizeTimestamp(point.timestamp),
      uploadMbps: (point.upload * 8) / intervalSeconds / 1_000_000,
    })),
  };
}

type RawProduct = {
  available: boolean;
  badge: string;
  description: string;
  device_limit: number;
  discount: Array<{ discount: number; quantity: number }>;
  display_template: string;
  id: number;
  max_quantity: number;
  min_quantity: number;
  name: string;
  sell: boolean;
  show: boolean;
  speed_limit: number;
  traffic: number;
  unit_price: number;
  unit_time: string;
  user_visible: boolean;
};

export async function getProducts(
  options: { authenticated?: boolean } = {},
  request: ApiRequester = apiRequest
): Promise<Product[]> {
  if (isClientDemoMode()) return demoProducts;
  const authenticated = options.authenticated !== false;
  const data = await request<ListPayload<RawProduct>>(
    authenticated
      ? "v1/public/subscribe/list?language=zh-CN"
      : "v1/public/portal/subscribe?language=zh-CN"
  );
  return listFrom(data)
    .filter((product) => product.sell)
    .map((product, index) => ({
      available: product.available,
      badge: product.badge,
      description: product.description,
      deviceLimit: product.device_limit,
      discounts: product.discount,
      id: product.id,
      maxQuantity: product.max_quantity || 0,
      minQuantity: product.min_quantity || 1,
      name: product.name,
      speedLimitMbps: bitsPerSecondToMbps(product.speed_limit),
      template:
        product.display_template === "studio"
          ? "studio"
          : product.display_template === "popular" || index === 1
            ? "popular"
            : "essential",
      trafficBytes: product.traffic,
      unitPrice: product.unit_price,
      unitTime: product.unit_time,
    }));
}

export async function getPurchaseQuote(input: {
  coupon: string;
  payment: number;
  quantity: number;
  subscribe_id: number;
}): Promise<PurchaseQuote> {
  if (isClientDemoMode()) {
    const product = demoProducts.find((item) => item.id === input.subscribe_id);
    if (!product) throw new Error("商品不存在");
    const rate =
      product.discounts.find((item) => item.quantity === input.quantity)
        ?.discount || 100;
    const price = product.unitPrice * input.quantity;
    const amount = Math.round((price * rate) / 100);
    return {
      amount,
      commissionDeduction: 0,
      coupon: input.coupon,
      couponDiscount: 0,
      discount: price - amount,
      feeAmount: 0,
      giftAmount: 0,
      price,
    };
  }
  const data = await apiRequest<{
    amount: number;
    commission_deduction: number;
    coupon: string;
    coupon_discount: number;
    discount: number;
    fee_amount: number;
    gift_amount: number;
    price: number;
  }>("v1/public/order/pre", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  if (!data) throw new Error("商品报价不可用");
  return {
    amount: data.amount,
    commissionDeduction: data.commission_deduction,
    coupon: data.coupon,
    couponDiscount: data.coupon_discount,
    discount: data.discount,
    feeAmount: data.fee_amount,
    giftAmount: data.gift_amount,
    price: data.price,
  };
}

export async function getMembershipQuote(input: {
  payment: number;
}): Promise<MembershipQuote> {
  if (isClientDemoMode()) {
    const plan = demoMembershipCard().plan;
    const user = demoCurrentUser();
    const commissionDeduction = Math.min(plan.unitPrice, user.commission);
    const afterCommission = plan.unitPrice - commissionDeduction;
    const giftAmount = Math.min(afterCommission, user.giftAmount);
    const providerAmount = afterCommission - giftAmount;
    const feeAmount =
      input.payment === 8 ? Math.round(providerAmount * 0.02) : 0;
    return {
      amount: providerAmount + feeAmount,
      commissionDeduction,
      feeAmount,
      giftAmount,
      price: plan.unitPrice,
    };
  }
  const data = await apiRequest<{
    amount: number;
    commission_deduction: number;
    fee_amount: number;
    gift_amount: number;
    price: number;
  }>("v1/public/order/membership/pre", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  if (!data) throw new Error("会员报价不可用");
  return {
    amount: data.amount,
    commissionDeduction: data.commission_deduction,
    feeAmount: data.fee_amount,
    giftAmount: data.gift_amount,
    price: data.price,
  };
}

export async function createMembershipOrder(input: {
  payment: number;
  request_id: string;
}) {
  if (isClientDemoMode()) {
    return createDemoOrder({
      payment: input.payment,
      quantity: 1,
      subscriptionId: 0,
      type: "membership",
    });
  }
  const data = await apiRequest<{ order_no: string }>(
    "v1/public/order/membership",
    {
      body: JSON.stringify(input),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    }
  );
  if (!data?.order_no) throw new Error("会员订单创建失败");
  return data.order_no;
}

export async function createPurchaseOrder(input: {
  coupon: string;
  payment: number;
  quantity: number;
  request_id: string;
  subscribe_id: number;
}) {
  if (isClientDemoMode()) {
    return createDemoOrder({
      payment: input.payment,
      quantity: input.quantity,
      subscriptionId: 0,
      type: "purchase",
    });
  }
  const data = await apiRequest<{ order_no: string }>(
    "v1/public/order/purchase",
    {
      body: JSON.stringify(input),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    }
  );
  if (!data?.order_no) throw new Error("购买订单创建失败");
  return data.order_no;
}

type RawOrder = {
  amount: number;
  created_at: number;
  fee_amount: number;
  id: number;
  order_no: string;
  product_name: string;
  quantity: number;
  status: number;
  type: number;
  updated_at: number;
};

function mapOrder(item: RawOrder): UserOrder {
  return {
    amount: item.amount,
    createdAt: normalizeTimestamp(item.created_at),
    feeAmount: item.fee_amount,
    id: item.id,
    orderNo: item.order_no,
    productName: item.product_name,
    quantity: item.quantity,
    status: item.status,
    type: item.type,
    updatedAt: normalizeTimestamp(item.updated_at),
  };
}

export async function getOrders(
  input: {
    page?: number;
    search?: string;
    size?: number;
    status?: number;
  } = {},
  request: ApiRequester = apiRequest
): Promise<{ list: UserOrder[]; total: number }> {
  if (isClientDemoMode()) {
    const now = Date.now();
    const search = input.search?.trim().toLowerCase();
    const filtered = [...demoOrders.entries()]
      .map(([orderNo, order], index) => ({
        amount: 6800,
        createdAt: order.createdAt,
        feeAmount: 0,
        id: index + 1,
        orderNo,
        productName: demoOrderName(order.type),
        quantity: order.quantity,
        status: order.status,
        type: demoOrderType(order.type),
        updatedAt: now,
      }))
      .filter(
        (order) =>
          (!input.status || order.status === input.status) &&
          (!search ||
            order.orderNo.toLowerCase().includes(search) ||
            order.productName.toLowerCase().includes(search))
      )
      .sort((a, b) => b.createdAt - a.createdAt);
    const page = Math.max(1, input.page || 1);
    const size = Math.max(1, input.size || 20);
    return {
      list: filtered.slice((page - 1) * size, page * size),
      total: filtered.length,
    };
  }
  const params = new URLSearchParams({
    page: String(input.page || 1),
    size: String(input.size || 20),
  });
  if (input.search?.trim()) params.set("search", input.search.trim());
  if (input.status) params.set("status", String(input.status));
  const data = await request<{ list: RawOrder[]; total: number }>(
    `v1/public/order/list?${params}`
  );
  return {
    list: (data?.list || []).map(mapOrder),
    total: data?.total || 0,
  };
}

export async function getOrderDetail(orderNo: string): Promise<UserOrder> {
  if (isClientDemoMode()) {
    const order = demoOrders.get(orderNo);
    if (!order) throw new Error("订单不存在");
    if (
      order.status === 1 &&
      order.payment !== -1 &&
      Date.now() - order.createdAt >= 3000
    ) {
      completeDemoOrder(orderNo);
    }
    return {
      amount: 6800,
      createdAt: order.createdAt,
      feeAmount: 0,
      id: 1,
      orderNo,
      productName: demoOrderName(order.type),
      quantity: order.quantity,
      status: order.status,
      type: demoOrderType(order.type),
      updatedAt: Date.now(),
    };
  }
  const params = new URLSearchParams({ order_no: orderNo });
  const data = await apiRequest<RawOrder>(`v1/public/order/detail?${params}`);
  if (!data) throw new Error("订单不存在");
  return mapOrder(data);
}

export async function closeOrder(orderNo: string) {
  if (isClientDemoMode()) {
    const order = demoOrders.get(orderNo);
    if (!order) throw new Error("订单不存在");
    order.status = 3;
    return;
  }
  await apiRequest("v1/public/order/close", {
    body: JSON.stringify({ orderNo }),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
}

export async function createRechargeOrder(input: {
  amount: number;
  payment: number;
  request_id: string;
}) {
  if (isClientDemoMode()) {
    return createDemoOrder({
      payment: input.payment,
      quantity: 1,
      subscriptionId: 0,
      type: "recharge",
    });
  }
  const data = await apiRequest<{ order_no: string }>(
    "v1/public/order/recharge",
    {
      body: JSON.stringify(input),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    }
  );
  if (!data?.order_no) throw new Error("充值订单创建失败");
  return data.order_no;
}

type RawSiteConfig = {
  currency?: { currency_symbol?: string };
  oauth_methods?: string[];
  subscribe?: {
    pan_domain?: boolean;
    subscribe_domain?: string;
    subscribe_path?: string;
  };
};

export function getSiteConfig(): Promise<SiteConfig> {
  return getSiteConfigWith(apiRequest);
}

export async function getSiteConfigWith(
  request: ApiRequester
): Promise<SiteConfig> {
  if (isClientDemoMode()) {
    return {
      currencySymbol: "¥",
      oauthMethods: ["github"],
      panDomain: false,
      subscribeDomain: "subscribe.alpha.example",
      subscribePath: "/s",
    };
  }
  const data = await request<RawSiteConfig>("v1/common/site/config");
  return {
    currencySymbol: data?.currency?.currency_symbol || "¥",
    oauthMethods: data?.oauth_methods || [],
    panDomain: Boolean(data?.subscribe?.pan_domain),
    subscribeDomain: data?.subscribe?.subscribe_domain || "",
    subscribePath: data?.subscribe?.subscribe_path || "",
  };
}

type RawUserAuthMethod = {
  auth_identifier: string;
  auth_type: string;
  verified: boolean;
};

type RawCurrentUser = {
  auth_methods: RawUserAuthMethod[];
  balance: number;
  commission: number;
  created_at: number;
  gift_amount: number;
  id: number;
  is_member: boolean;
  member_expired_at: number;
  refer_code: string;
};

function mapAuthMethod(method: RawUserAuthMethod) {
  return {
    identifier: method.auth_identifier,
    type: method.auth_type,
    verified: method.verified,
  };
}

function demoCurrentUser(): CurrentUser {
  return {
    authMethods: [
      { identifier: "d***@outlook.com", type: "email", verified: true },
      { identifier: "dawn-alpha", type: "github", verified: true },
    ],
    balance: 12_800,
    commission: 0,
    createdAt: Date.UTC(2026, 5, 10),
    giftAmount: 2000,
    id: 1001,
    isMember: demoMembershipExpiredAt > Date.now(),
    memberExpiredAt: demoMembershipExpiredAt,
    referCode: "ALPHA-7K2M",
  };
}

export function getCurrentUser(): Promise<CurrentUser> {
  return getCurrentUserWith(apiRequest);
}

export async function getCurrentUserWith(
  request: ApiRequester
): Promise<CurrentUser> {
  if (isClientDemoMode()) return demoCurrentUser();
  const data = await request<RawCurrentUser>("v1/public/user/info");
  if (!data) throw new Error("账户资料不可用");
  return {
    authMethods: (data.auth_methods || []).map(mapAuthMethod),
    balance: data.balance,
    commission: data.commission,
    createdAt: normalizeTimestamp(data.created_at),
    giftAmount: data.gift_amount,
    id: data.id,
    isMember: Boolean(data.is_member),
    memberExpiredAt: data.member_expired_at
      ? normalizeTimestamp(data.member_expired_at)
      : 0,
    referCode: data.refer_code,
  };
}

type RawMembershipCard = {
  enabled: boolean;
  expired_at: number;
  is_member: boolean;
  plan: {
    description: string;
    duration_unit: string;
    duration_value: number;
    enabled: boolean;
    id: number;
    name: string;
    policy: {
      monthly_consumption_limit: number;
      monthly_consumption_limit_enabled: boolean;
      monthly_order_limit: number;
      monthly_order_limit_enabled: boolean;
      subscription_expiry_grace_days: number;
      subscription_expiry_limit_enabled: boolean;
    };
    unit_price: number;
  };
  usage: {
    monthly_consumption: number;
    monthly_order_count: number;
  };
};

function demoMembershipCard(): MembershipCard {
  return {
    enabled: true,
    expiredAt: demoMembershipExpiredAt,
    isMember: demoMembershipExpiredAt > Date.now(),
    plan: {
      description: "订阅购买资格与账户级权益管理。",
      durationUnit: "Year",
      durationValue: 1,
      enabled: true,
      id: 1,
      name: "年度会员",
      policy: {
        monthlyConsumptionLimit: 50_000,
        monthlyConsumptionLimitEnabled: true,
        monthlyOrderLimit: 20,
        monthlyOrderLimitEnabled: true,
        subscriptionExpiryGraceDays: 30,
        subscriptionExpiryLimitEnabled: true,
      },
      unitPrice: 8800,
    },
    usage: {
      monthlyConsumption: 6800,
      monthlyOrderCount: 2,
    },
  };
}

function mapMembershipCard(data: RawMembershipCard): MembershipCard {
  return {
    enabled: Boolean(data.enabled),
    expiredAt: data.expired_at ? normalizeTimestamp(data.expired_at) : 0,
    isMember: Boolean(data.is_member),
    plan: {
      description: data.plan?.description || "",
      durationUnit: data.plan?.duration_unit || "Year",
      durationValue: data.plan?.duration_value || 1,
      enabled: Boolean(data.plan?.enabled),
      id: data.plan?.id || 0,
      name: data.plan?.name || "会员",
      policy: {
        monthlyConsumptionLimit:
          data.plan?.policy?.monthly_consumption_limit || 0,
        monthlyConsumptionLimitEnabled: Boolean(
          data.plan?.policy?.monthly_consumption_limit_enabled
        ),
        monthlyOrderLimit: data.plan?.policy?.monthly_order_limit || 0,
        monthlyOrderLimitEnabled: Boolean(
          data.plan?.policy?.monthly_order_limit_enabled
        ),
        subscriptionExpiryGraceDays:
          data.plan?.policy?.subscription_expiry_grace_days || 0,
        subscriptionExpiryLimitEnabled: Boolean(
          data.plan?.policy?.subscription_expiry_limit_enabled
        ),
      },
      unitPrice: data.plan?.unit_price || 0,
    },
    usage: {
      monthlyConsumption: data.usage?.monthly_consumption || 0,
      monthlyOrderCount: data.usage?.monthly_order_count || 0,
    },
  };
}

export function getMembershipCard(): Promise<MembershipCard> {
  return getMembershipCardWith(apiRequest);
}

export async function getMembershipCardWith(
  request: ApiRequester
): Promise<MembershipCard> {
  if (isClientDemoMode()) return demoMembershipCard();
  const data = await request<RawMembershipCard>(
    "v1/public/order/membership/card"
  );
  if (!data) throw new Error("会员状态不可用");
  return mapMembershipCard(data);
}

export async function updatePassword(input: {
  current_password: string;
  password: string;
}) {
  if (isClientDemoMode()) return;
  await apiRequest("v1/public/user/password", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "PUT",
  });
}

export function getOAuthMethods(): Promise<UserAuthMethod[]> {
  return getOAuthMethodsWith(apiRequest);
}

export async function getOAuthMethodsWith(request: ApiRequester) {
  if (isClientDemoMode()) return demoCurrentUser().authMethods;
  const data = await request<{ methods: RawUserAuthMethod[] }>(
    "v1/public/user/oauth_methods"
  );
  return (data?.methods || []).map(mapAuthMethod);
}

export async function startOAuthBind(method: "github", redirect: string) {
  const data = await apiRequest<{ redirect: string }>(
    "v1/public/user/bind_oauth",
    {
      body: JSON.stringify({ method, redirect }),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    }
  );
  if (!data?.redirect) throw new Error("GitHub 绑定暂时不可用");
  return data.redirect;
}

export async function finishOAuthBind(input: { code: string; state: string }) {
  await apiRequest("v1/public/user/bind_oauth/callback", {
    body: JSON.stringify({ method: "github", callback: input }),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
}

export async function unbindOAuth(method: "github") {
  if (isClientDemoMode()) return;
  await apiRequest("v1/public/user/unbind_oauth", {
    body: JSON.stringify({ method }),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
}

type RawWalletLog = {
  amount: number;
  asset: WalletLog["asset"];
  balance?: number;
  id: number;
  order_no?: string;
  timestamp: number;
  type: number;
};

export function getWalletLogs(): Promise<WalletLog[]> {
  return getWalletLogsWith(apiRequest);
}

export async function getWalletLogsWith(
  request: ApiRequester
): Promise<WalletLog[]> {
  if (isClientDemoMode()) return [];
  const data = await request<{ list: RawWalletLog[] }>(
    "v1/public/user/wallet_log?page=1&size=30"
  );
  return (data?.list || []).map((item) => ({
    amount: item.amount,
    asset: item.asset,
    balance: item.balance,
    id: item.id,
    orderNo: item.order_no || "",
    timestamp: normalizeTimestamp(item.timestamp),
    type: item.type,
  }));
}

type RawTicket = {
  description?: string;
  id: number;
  status: number;
  title: string;
  updated_at: number;
};

type RawTicketDetail = RawTicket & {
  created_at: number;
  follow?: Array<{
    content?: string;
    created_at: number;
    from: "System" | "User";
    id: number;
    images?: string[];
  }>;
  images?: string[];
};

export function getTickets(): Promise<UserTicket[]> {
  return getTicketsWith(apiRequest);
}

export async function getTicketsWith(
  request: ApiRequester
): Promise<UserTicket[]> {
  if (isClientDemoMode()) {
    return [
      {
        description: "移动网络偶发断开",
        id: 1,
        status: 1,
        title: "东京节点连接不稳定",
        updatedAt: Date.now() - 3_600_000,
      },
    ];
  }
  const data = await request<{ list: RawTicket[] }>(
    "v1/public/ticket/list?page=1&size=30"
  );
  return (data?.list || []).map((item) => ({
    description: item.description || "",
    id: item.id,
    status: item.status,
    title: item.title,
    updatedAt: normalizeTimestamp(item.updated_at),
  }));
}

export async function createTicket(input: {
  description: string;
  images: string[];
  title: string;
}) {
  if (isClientDemoMode()) return;
  await apiRequest("v1/public/ticket/", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
}

export async function getTicketDetail(id: number): Promise<UserTicketDetail> {
  if (isClientDemoMode()) {
    return {
      createdAt: Date.now() - 3_600_000,
      description: "移动网络偶发断开",
      id,
      images: [],
      messages: [],
      status: 1,
      title: "东京节点连接不稳定",
      updatedAt: Date.now() - 3_600_000,
    };
  }
  const data = await apiRequest<RawTicketDetail>(
    `v1/public/ticket/detail?id=${encodeURIComponent(id)}`
  );
  if (!data) {
    throw new ApiError("工单详情为空", 502);
  }
  return {
    createdAt: normalizeTimestamp(data.created_at),
    description: data.description || "",
    id: data.id,
    images: data.images || [],
    messages: (data.follow || []).map((message) => ({
      content: message.content || "",
      createdAt: normalizeTimestamp(message.created_at),
      from: message.from,
      id: message.id,
      images: message.images || [],
    })),
    status: data.status,
    title: data.title,
    updatedAt: normalizeTimestamp(data.updated_at),
  };
}

export async function replyToTicket(input: {
  content: string;
  images: string[];
  ticketId: number;
}) {
  if (isClientDemoMode()) return;
  await apiRequest("v1/public/ticket/follow", {
    body: JSON.stringify({
      content: input.content,
      images: input.images,
      ticket_id: input.ticketId,
      type: 1,
    }),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
}

export function getAnnouncements(): Promise<Announcement[]> {
  return getAnnouncementsWith(apiRequest);
}

export async function getAnnouncementsWith(
  request: ApiRequester
): Promise<Announcement[]> {
  if (isClientDemoMode()) {
    return [
      {
        content:
          "近 5 分钟与近 15 分钟视图会按服务端实际采样间隔展示实时速率。",
        createdAt: Date.now() - 86_400_000,
        id: 1,
        title: "实时流量视图已更新",
      },
    ];
  }
  const data = await request<{
    announcements: Array<{
      content?: string;
      created_at: number;
      id: number;
      title: string;
    }>;
  }>("v1/public/announcement/list?page=1&size=10");
  return (data?.announcements || []).map((item) => ({
    content: item.content || "",
    createdAt: normalizeTimestamp(item.created_at),
    id: item.id,
    title: item.title,
  }));
}

export function getDocuments(): Promise<UserDocument[]> {
  return getDocumentsWith(apiRequest);
}

export async function getDocumentsWith(
  request: ApiRequester
): Promise<UserDocument[]> {
  if (isClientDemoMode()) {
    return [
      {
        description: "接入 · 预设",
        id: 1,
        title: "客户端接入指南",
      },
      {
        description: "购买 · 续费 · 重置",
        id: 2,
        title: "套餐与流量规则",
      },
    ];
  }
  const data = await request<{
    list: Array<{ id: number; tags?: string[]; title: string }>;
  }>("v1/public/document/list?page=1&size=30");
  return (data?.list || []).map((item) => ({
    description: item.tags?.join(" · ") || "使用文档",
    id: item.id,
    title: item.title,
  }));
}

export async function getDocumentDetail(id: number): Promise<UserDocument> {
  if (isClientDemoMode()) {
    const document = (await getDocuments()).find((item) => item.id === id);
    if (!document) throw new Error("文档不存在");
    return {
      ...document,
      content:
        id === 1
          ? "为每个订阅选择协议和节点，并保存适合不同设备的接入预设。"
          : "新购订单锁定当前价格；后续续费使用商品届时的有效价格。",
    };
  }
  const data = await apiRequest<{
    content?: string;
    id: number;
    tags?: string[];
    title: string;
  }>(`v1/public/document/detail?id=${encodeURIComponent(String(id))}`);
  if (!data) throw new Error("文档不存在");
  return {
    content: data.content || "",
    description: data.tags?.join(" · ") || "使用文档",
    id: data.id,
    title: data.title,
  };
}

export type AccessPresetInput = {
  is_default?: boolean;
  items: Array<{
    display_name?: string;
    inbound_alias: string;
    node_id: number;
  }>;
  name: string;
};

function demoPresetFromInput(
  id: number,
  key: string,
  input: AccessPresetInput,
  enabled = true,
  isDefault = input.is_default ?? false
): AccessPreset {
  return {
    enabled,
    id,
    isDefault,
    items: input.items.map((item) => ({
      displayName: item.display_name || "",
      inboundAlias: item.inbound_alias,
      nodeId: item.node_id,
    })),
    key,
    mode: "custom",
    name: input.name,
    profileKey: "",
  };
}

export async function createAccessPreset(
  input: AccessPresetInput & { user_subscribe_id: number }
) {
  if (isClientDemoMode()) {
    const scope = demoScopeState.find(
      (item) => item.id === input.user_subscribe_id
    );
    if (!scope) throw new Error("订阅接入信息不存在");
    if (scope.presets.length >= 10)
      throw new Error("每个订阅最多保存 10 个预设");
    const isDefault =
      Boolean(input.is_default) ||
      !scope.presets.some((preset) => preset.isDefault);
    if (isDefault) {
      for (const current of scope.presets) current.isDefault = false;
    }
    const preset = demoPresetFromInput(
      demoPresetSequence,
      `ps_demo_${demoPresetSequence}`,
      input,
      true,
      isDefault
    );
    demoPresetSequence += 1;
    scope.presets.push(preset);
    return cloneScope({ ...scope, presets: [preset] }).presets[0]!;
  }
  const data = await apiRequest<{ preset: RawPreset }>(
    "v1/public/user/subscribe_preset",
    {
      body: JSON.stringify(input),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    }
  );
  if (!data?.preset) throw new Error("保存成功，但未返回预设信息");
  return mapPreset(data.preset);
}

export async function updateAccessPreset(
  input: AccessPresetInput & { enabled?: boolean; id: number }
) {
  if (isClientDemoMode()) {
    for (const scope of demoScopeState) {
      const index = scope.presets.findIndex((preset) => preset.id === input.id);
      if (index < 0) continue;
      const current = scope.presets[index]!;
      const isDefault = input.is_default ?? current.isDefault;
      if (current.isDefault && input.enabled === false)
        throw new Error("默认预设不能停用");
      if (input.is_default) {
        for (const item of scope.presets) item.isDefault = false;
      }
      const preset = demoPresetFromInput(
        current.id,
        current.key,
        input,
        input.is_default ? true : (input.enabled ?? current.enabled),
        isDefault
      );
      scope.presets[index] = preset;
      return cloneScope({ ...scope, presets: [preset] }).presets[0]!;
    }
    throw new Error("预设不存在");
  }
  const data = await apiRequest<{ preset: RawPreset }>(
    "v1/public/user/subscribe_preset",
    {
      body: JSON.stringify(input),
      headers: { "Content-Type": "application/json" },
      method: "PUT",
    }
  );
  if (!data?.preset) throw new Error("保存成功，但未返回预设信息");
  return mapPreset(data.preset);
}

export async function deleteAccessPreset(id: number) {
  if (isClientDemoMode()) {
    for (const scope of demoScopeState) {
      const index = scope.presets.findIndex((preset) => preset.id === id);
      if (index < 0) continue;
      const [removed] = scope.presets.splice(index, 1);
      if (removed?.isDefault) {
        const replacement = scope.presets.find((preset) => preset.enabled);
        if (replacement) replacement.isDefault = true;
      }
      return;
    }
    throw new Error("预设不存在");
  }
  await apiRequest("v1/public/user/subscribe_preset", {
    body: JSON.stringify({ id }),
    headers: { "Content-Type": "application/json" },
    method: "DELETE",
  });
}

type RawPaymentMethod = {
  description: string;
  fee_amount: number;
  fee_mode: number;
  fee_percent: number;
  icon: string;
  id: number;
  name: string;
  platform: string;
};

export function getPaymentMethods(): Promise<PaymentMethod[]> {
  return getPaymentMethodsWith(apiRequest);
}

export async function getPaymentMethodsWith(
  request: ApiRequester
): Promise<PaymentMethod[]> {
  if (isClientDemoMode()) {
    return [
      {
        description: "余额充足时立即完成",
        feeAmount: 0,
        feeMode: 0,
        feePercent: 0,
        icon: "",
        id: -1,
        name: "账户余额",
        platform: "balance",
      },
      {
        description: "扫码完成在线支付",
        feeAmount: 0,
        feeMode: 1,
        feePercent: 2,
        icon: "",
        id: 8,
        name: "在线支付",
        platform: "alipay_f2f",
      },
    ];
  }
  const data = await request<ListPayload<RawPaymentMethod>>(
    "v1/public/payment/methods"
  );
  return listFrom(data).map((method) => ({
    description: method.description,
    feeAmount: method.fee_amount,
    feeMode: method.fee_mode,
    feePercent: method.fee_percent,
    icon: method.icon,
    id: method.id,
    name: method.name,
    platform: method.platform,
  }));
}

type RawOrderProduct = {
  device_limit: number;
  discount: Array<{ discount: number; quantity: number }>;
  id: number;
  max_quantity: number;
  min_quantity: number;
  name: string;
  replacement: number;
  show_original_price: boolean;
  speed_limit: number;
  traffic: number;
  unit_price: number;
  unit_time: string;
};

function mapOrderProduct(product: RawOrderProduct): OrderProduct {
  return {
    deviceLimit: product.device_limit,
    discounts: product.discount || [],
    id: product.id,
    maxQuantity: product.max_quantity || 0,
    minQuantity: product.min_quantity || 1,
    name: product.name,
    resetPrice: product.replacement,
    showOriginalPrice: product.show_original_price,
    speedLimitMbps: bitsPerSecondToMbps(product.speed_limit),
    trafficBytes: product.traffic,
    unitPrice: product.unit_price,
    unitTime: product.unit_time,
  };
}

function demoOrderProduct(subscription: Subscription): OrderProduct {
  return {
    deviceLimit: subscription.deviceLimit,
    discounts: subscription.discounts,
    id: subscription.productId,
    maxQuantity: 24,
    minQuantity: 1,
    name: subscription.planName,
    resetPrice: subscription.resetPrice,
    showOriginalPrice: subscription.showOriginalPrice,
    speedLimitMbps: subscription.speedLimitMbps,
    trafficBytes: subscription.trafficBytes,
    unitPrice: subscription.unitPrice,
    unitTime: subscription.unitTime,
  };
}

export async function getRenewalQuote(input: {
  coupon: string;
  payment: number;
  quantity: number;
  user_subscribe_id: number;
}): Promise<RenewalQuote> {
  if (isClientDemoMode()) {
    const subscription = demoSubscriptionState.find(
      (item) => item.id === input.user_subscribe_id
    );
    if (!subscription) throw new Error("订阅不存在");
    const quantity = input.quantity > 0 ? input.quantity : 1;
    const rate =
      subscription.discounts.find((item) => item.quantity === quantity)
        ?.discount || 100;
    const price = subscription.unitPrice * quantity;
    const discounted = Math.round((price * rate) / 100);
    const feeAmount = input.payment === 8 ? Math.round(discounted * 0.02) : 0;
    return {
      amount: discounted + feeAmount,
      commissionDeduction: 0,
      coupon: input.coupon,
      couponDiscount: 0,
      discount: price - discounted,
      feeAmount,
      giftAmount: 0,
      price,
      product: demoOrderProduct(subscription),
      quantity,
    };
  }
  const data = await apiRequest<{
    amount: number;
    commission_deduction: number;
    coupon: string;
    coupon_discount: number;
    discount: number;
    fee_amount: number;
    gift_amount: number;
    price: number;
    product: RawOrderProduct;
    quantity: number;
  }>("v1/public/order/renewal/pre", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  if (!data) throw new Error("续费报价不可用");
  return {
    amount: data.amount,
    commissionDeduction: data.commission_deduction,
    coupon: data.coupon,
    couponDiscount: data.coupon_discount,
    discount: data.discount,
    feeAmount: data.fee_amount,
    giftAmount: data.gift_amount,
    price: data.price,
    product: mapOrderProduct(data.product),
    quantity: data.quantity,
  };
}

export async function getResetTrafficQuote(input: {
  payment: number;
  user_subscribe_id: number;
}): Promise<ResetTrafficQuote> {
  if (isClientDemoMode()) {
    const subscription = demoSubscriptionState.find(
      (item) => item.id === input.user_subscribe_id
    );
    if (!subscription) throw new Error("订阅不存在");
    const feeAmount =
      input.payment === 8 ? Math.round(subscription.resetPrice * 0.02) : 0;
    return {
      amount: subscription.resetPrice + feeAmount,
      commissionDeduction: 0,
      feeAmount,
      giftAmount: 0,
      price: subscription.resetPrice,
      product: demoOrderProduct(subscription),
    };
  }
  const data = await apiRequest<{
    amount: number;
    commission_deduction: number;
    fee_amount: number;
    gift_amount: number;
    price: number;
    product: RawOrderProduct;
  }>("v1/public/order/reset/pre", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  if (!data) throw new Error("重置报价不可用");
  return {
    amount: data.amount,
    commissionDeduction: data.commission_deduction,
    feeAmount: data.fee_amount,
    giftAmount: data.gift_amount,
    price: data.price,
    product: mapOrderProduct(data.product),
  };
}

function createDemoOrder(input: {
  payment: number;
  quantity: number;
  subscriptionId: number;
  type: DemoOrderType;
}) {
  const orderNo = `ALP-DEMO-${String(demoOrderSequence).padStart(4, "0")}`;
  demoOrderSequence += 1;
  demoOrders.set(orderNo, {
    ...input,
    createdAt: Date.now(),
    status: 1,
  });
  return orderNo;
}

export async function createRenewalOrder(input: {
  coupon: string;
  payment: number;
  quantity: number;
  request_id: string;
  user_subscribe_id: number;
}) {
  if (isClientDemoMode()) {
    return createDemoOrder({
      payment: input.payment,
      quantity: input.quantity,
      subscriptionId: input.user_subscribe_id,
      type: "renewal",
    });
  }
  const data = await apiRequest<{ order_no: string }>(
    "v1/public/order/renewal",
    {
      body: JSON.stringify(input),
      headers: { "Content-Type": "application/json" },
      method: "POST",
    }
  );
  if (!data?.order_no) throw new Error("续费订单创建失败");
  return data.order_no;
}

export async function createResetTrafficOrder(input: {
  payment: number;
  request_id: string;
  user_subscribe_id: number;
}) {
  if (isClientDemoMode()) {
    return createDemoOrder({
      payment: input.payment,
      quantity: 1,
      subscriptionId: input.user_subscribe_id,
      type: "reset",
    });
  }
  const data = await apiRequest<{ order_no: string }>("v1/public/order/reset", {
    body: JSON.stringify(input),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  if (!data?.order_no) throw new Error("重置订单创建失败");
  return data.order_no;
}

function completeDemoOrder(orderNo: string) {
  const order = demoOrders.get(orderNo);
  if (!order) throw new Error("订单不存在");
  order.status = 5;
  if (order.type === "membership") {
    const base = Math.max(Date.now(), demoMembershipExpiredAt);
    const expiresAt = new Date(base);
    expiresAt.setFullYear(expiresAt.getFullYear() + 1);
    demoMembershipExpiredAt = expiresAt.getTime();
    return;
  }
  if (order.type === "purchase" || order.type === "recharge") return;
  const subscription = demoSubscriptionState.find(
    (item) => item.id === order.subscriptionId
  );
  if (!subscription) throw new Error("订阅不存在");
  if (order.type === "reset") {
    subscription.downloadBytes = 0;
    subscription.uploadBytes = 0;
  } else {
    const date = new Date(Math.max(Date.now(), subscription.expiresAt));
    if (subscription.unitTime.toLowerCase() === "year") {
      date.setFullYear(date.getFullYear() + order.quantity);
    } else if (subscription.unitTime.toLowerCase() === "day") {
      date.setDate(date.getDate() + order.quantity);
    } else {
      date.setMonth(date.getMonth() + order.quantity);
    }
    subscription.expiresAt = date.getTime();
  }
}

export async function checkoutOrder(orderNo: string): Promise<CheckoutResult> {
  if (isClientDemoMode()) {
    const order = demoOrders.get(orderNo);
    if (!order) throw new Error("订单不存在");
    if (order.payment === -1) {
      completeDemoOrder(orderNo);
      return { type: "balance" };
    }
    return {
      checkoutUrl: `https://pay.alpha.example/checkout/${orderNo}`,
      type: "qr",
    };
  }
  const data = await apiRequest<{
    checkout_url?: string;
    stripe?: {
      client_secret: string;
      method: string;
      publishable_key: string;
    };
    type: CheckoutResult["type"];
  }>("v1/public/portal/order/checkout", {
    body: JSON.stringify({
      orderNo,
      returnUrl: window.location.href,
    }),
    headers: { "Content-Type": "application/json" },
    method: "POST",
  });
  if (!data?.type) throw new Error("支付初始化失败");
  return {
    checkoutUrl: data.checkout_url,
    stripe: data.stripe
      ? {
          clientSecret: data.stripe.client_secret,
          method: data.stripe.method,
          publishableKey: data.stripe.publishable_key,
        }
      : undefined,
    type: data.type,
  };
}
