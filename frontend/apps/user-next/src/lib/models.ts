export type TrafficPeriod = "5m" | "15m" | "1h" | "24h" | "7d" | "30d";

export type PriceDiscount = {
  discount: number;
  quantity: number;
};

export type Subscription = {
  allowRenewal: boolean;
  allowResetTraffic: boolean;
  canRenew: boolean;
  canResetTraffic: boolean;
  createdAt: number;
  deviceLimit: number;
  discounts: PriceDiscount[];
  downloadBytes: number;
  expiresAt: number;
  id: number;
  name: string;
  planName: string;
  productId: number;
  resetAt: number;
  resetPrice: number;
  short: string;
  showOriginalPrice: boolean;
  speedLimitMbps: number;
  renewalReason: string;
  remainingTrafficBytes: number;
  resetTrafficReason: string;
  status:
    | "active"
    | "deducted"
    | "exhausted"
    | "expired"
    | "pending"
    | "stopped"
    | "unknown";
  statusReason: string;
  token: string;
  trafficBytes: number;
  unitPrice: number;
  unitTime: string;
  uploadBytes: number;
};

export type TrafficPoint = {
  downloadMbps: number;
  observed: boolean;
  timestamp: number;
  uploadMbps: number;
};

export type TrafficSeries = {
  intervalSeconds: number;
  points: TrafficPoint[];
};

export type AccessNode = {
  city: string;
  country: string;
  id: number;
  inboundAlias: string;
  itemKey: string;
  key: string;
  name: string;
  profileKey: string;
  profileName: string;
  protocol: string;
  serverName: string;
  tags: string[];
};

export type AccessProfile = {
  key: string;
  name: string;
  nodeCount: number;
  protocol: string;
  security: string;
  sort: number;
  transport: string;
};

export type AccessPresetItem = {
  displayName: string;
  inboundAlias: string;
  nodeId: number;
};

export type AccessPreset = {
  enabled: boolean;
  id: number;
  isDefault: boolean;
  items: AccessPresetItem[];
  key: string;
  mode: string;
  name: string;
  profileKey: string;
};

export type SubscriptionScope = {
  id: number;
  nodes: AccessNode[];
  presets: AccessPreset[];
  profiles: AccessProfile[];
};

export type Product = {
  available: boolean;
  badge: string;
  description: string;
  deviceLimit: number;
  discounts: PriceDiscount[];
  id: number;
  maxQuantity: number;
  minQuantity: number;
  name: string;
  speedLimitMbps: number;
  template: "essential" | "popular" | "studio";
  trafficBytes: number;
  unitPrice: number;
  unitTime: string;
};

export type OrderSummary = {
  amount: number;
  createdAt: number;
  id: string;
  plan: string;
  status: "paid" | "pending" | "refunded";
};

export type SiteConfig = {
  currencySymbol: string;
  oauthMethods: string[];
  panDomain: boolean;
  subscribeDomain: string;
  subscribePath: string;
};

export type PaymentMethod = {
  description: string;
  feeAmount: number;
  feeMode: number;
  feePercent: number;
  icon: string;
  id: number;
  name: string;
  platform: string;
};

export type OrderProduct = {
  deviceLimit: number;
  discounts: PriceDiscount[];
  id: number;
  maxQuantity: number;
  minQuantity: number;
  name: string;
  resetPrice: number;
  showOriginalPrice: boolean;
  speedLimitMbps: number;
  trafficBytes: number;
  unitPrice: number;
  unitTime: string;
};

export type RenewalQuote = {
  amount: number;
  commissionDeduction: number;
  coupon: string;
  couponDiscount: number;
  discount: number;
  feeAmount: number;
  giftAmount: number;
  price: number;
  product: OrderProduct;
  quantity: number;
};

export type ResetTrafficQuote = {
  amount: number;
  commissionDeduction: number;
  feeAmount: number;
  giftAmount: number;
  price: number;
  product: OrderProduct;
};

export type StripeCheckout = {
  clientSecret: string;
  method: string;
  publishableKey: string;
};

export type CheckoutResult = {
  checkoutUrl?: string;
  stripe?: StripeCheckout;
  type: "balance" | "free" | "processing" | "qr" | "stripe" | "url";
};

export type PurchaseQuote = {
  amount: number;
  commissionDeduction: number;
  coupon: string;
  couponDiscount: number;
  discount: number;
  feeAmount: number;
  giftAmount: number;
  price: number;
};

export type MembershipPolicy = {
  monthlyConsumptionLimit: number;
  monthlyConsumptionLimitEnabled: boolean;
  monthlyOrderLimit: number;
  monthlyOrderLimitEnabled: boolean;
  subscriptionExpiryGraceDays: number;
  subscriptionExpiryLimitEnabled: boolean;
};

export type MembershipPlan = {
  description: string;
  durationUnit: string;
  durationValue: number;
  enabled: boolean;
  id: number;
  name: string;
  policy: MembershipPolicy;
  unitPrice: number;
};

export type MembershipCard = {
  enabled: boolean;
  expiredAt: number;
  isMember: boolean;
  plan: MembershipPlan;
  usage: {
    monthlyConsumption: number;
    monthlyOrderCount: number;
  };
};

export type MembershipQuote = {
  amount: number;
  commissionDeduction: number;
  feeAmount: number;
  giftAmount: number;
  price: number;
};

export type UserAuthMethod = {
  identifier: string;
  type: string;
  verified: boolean;
};

export type CurrentUser = {
  authMethods: UserAuthMethod[];
  balance: number;
  commission: number;
  createdAt: number;
  giftAmount: number;
  id: number;
  isMember: boolean;
  memberExpiredAt: number;
  referCode: string;
};

export type UserOrder = {
  amount: number;
  createdAt: number;
  feeAmount: number;
  id: number;
  orderNo: string;
  productName: string;
  quantity: number;
  status: number;
  type: number;
  updatedAt: number;
};

export type WalletLog = {
  amount: number;
  asset: "balance" | "commission" | "gift";
  balance?: number;
  id: number;
  orderNo: string;
  timestamp: number;
  type: number;
};

export type UserTicket = {
  description: string;
  id: number;
  status: number;
  title: string;
  updatedAt: number;
};

export type UserTicketMessage = {
  content: string;
  createdAt: number;
  from: "System" | "User";
  id: number;
  images: string[];
};

export type UserTicketDetail = UserTicket & {
  createdAt: number;
  images: string[];
  messages: UserTicketMessage[];
};

export type Announcement = {
  content: string;
  createdAt: number;
  id: number;
  title: string;
};

export type UserDocument = {
  content?: string;
  description: string;
  id: number;
  title: string;
};
