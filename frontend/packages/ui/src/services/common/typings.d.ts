declare namespace API {
  type JsonValue = string | number | boolean | null | JsonObject | JsonValue[];

  type JsonObject = { [key: string]: JsonValue };

  type Ads = {
    id: number;
    title: string;
    type: string;
    content?: string;
    description: string;
    target_url: string;
    start_time: number;
    end_time: number;
    status: number;
    created_at: number;
    updated_at: number;
  };

  type AdsListItem = {
    id: number;
    title: string;
    type: string;
    description: string;
    target_url: string;
    start_time: number;
    end_time: number;
    status: number;
    created_at: number;
    updated_at: number;
  };

  type AlipayNotifyResponse = {
    return_code: string;
  };

  type Announcement = {
    id: number;
    title: string;
    content?: string;
    show: boolean;
    pinned: boolean;
    popup: boolean;
    created_at: number;
    updated_at: number;
  };

  type AnyTLS = {
    port: number;
    security_config: SecurityConfig;
  };

  type AppUserSubcbribe = {
    id: number;
    name: string;
    upload: number;
    traffic: number;
    download: number;
    device_limit: number;
    start_time: string;
    expire_time: string;
    list: AppUserSubscbribeNode[];
  };

  type AppUserSubscbribeNode = {
    id: number;
    name: string;
    uuid: string;
    protocol: string;
    relay_mode: string;
    relay_node: string;
    server_addr: string;
    speed_limit: number;
    tags: string[];
    traffic: number;
    traffic_ratio: number;
    upload: number;
    config: string;
    country: string;
    city: string;
    latitude: string;
    longitude: string;
    created_at: number;
    download: number;
  };

  type AuthConfig = {
    email: EmailAuthticateConfig;
    register: PubilcRegisterConfig;
  };

  type AuthMethodConfig = {
    id: number;
    method: string;
    config?: JsonValue;
    enabled: boolean;
  };

  type AuthMethodSummary = {
    id: number;
    method: string;
    enabled: boolean;
  };

  type BalanceLog = {
    type: number;
    user_id: number;
    amount: number;
    order_no?: string;
    balance: number;
    timestamp: number;
  };

  type CheckoutOrderRequest = {
    orderNo: string;
    returnUrl?: string;
  };

  type CheckoutOrderResponse = {
    type: string;
    checkout_url?: string;
    stripe?: StripePayment;
  };

  type CheckUserParams = {
    email: string;
  };

  type CheckUserRequest = {
    email: string;
  };

  type CheckUserResponse = {
    exist: boolean;
  };

  type CheckVerificationCodeRequest = {
    method: "email";
    account: string;
    code: string;
    type: number;
  };

  type CheckVerificationCodeRespone = {
    status: boolean;
  };

  type CloseOrderRequest = {
    orderNo: string;
  };

  type CommissionLog = {
    type: number;
    user_id: number;
    amount: number;
    order_no: string;
    timestamp: number;
  };

  type Coupon = {
    id: number;
    name: string;
    code: string;
    count: number;
    type: number;
    discount: number;
    start_time: number;
    expire_time: number;
    user_limit: number;
    subscribe: number[];
    used_count: number;
    enable: boolean;
    created_at: number;
    updated_at: number;
  };

  type Currency = {
    currency_unit: string;
    currency_symbol: string;
  };

  type CurrencyConfig = {
    access_key: string;
    currency_unit: string;
    currency_symbol: string;
  };

  type Document = {
    id: number;
    title: string;
    content?: string;
    tags: string[];
    show: boolean;
    created_at: number;
    updated_at: number;
  };

  type PublicDocumentListItem = {
    id: number;
    title: string;
    tags: string[];
    updated_at: number;
  };

  type DownloadLink = {
    ios?: string;
    android?: string;
    windows?: string;
    mac?: string;
    linux?: string;
    harmony?: string;
  };

  type EmailAuthticateConfig = {
    enable: boolean;
    enable_verify: boolean;
    enable_domain_suffix: boolean;
    domain_suffix_list: string;
  };

  type EPayNotifyRequest = {
    pid: number;
    trade_no: string;
    out_trade_no: string;
    type: string;
    name: string;
    money: string;
    trade_status: string;
    param: string;
    sign: string;
    sign_type: string;
  };

  type Follow = {
    id: number;
    ticket_id: number;
    from: string;
    type: number;
    content: string;
    images: string[];
    created_at: number;
  };

  type GetAdsParams = {
    device: string;
    position: string;
  };

  type GetAdsRequest = {
    device: string;
    position: string;
  };

  type GetAdsResponse = {
    list: AdsListItem[];
  };

  type GetAvailablePaymentMethodsResponse = {
    list: PaymentMethod[];
  };

  type GetGlobalConfigResponse = {
    site: SiteConfig;
    verify: VeifyConfig;
    auth: PublicAuthConfig;
    invite: InviteConfig;
    currency: Currency;
    subscribe: PublicSubscribeConfig;
    verify_code: PubilcVerifyCodeConfig;
    oauth_methods: string[];
    web_ad: boolean;
  };

  type GetStatResponse = {
    user: number;
    node: number;
    country: number;
    protocol: string[];
  };

  type GetSubscribeClientResponse = {
    total: number;
    list: SubscribeClient[];
  };

  type GetTosResponse = {
    tos_content: string;
  };

  type GetUserSubscribeTrafficLogsRequest = {
    page: number;
    size: number;
    user_id: number;
    subscribe_id: number;
    start_time: number;
    end_time: number;
  };

  type GetUserSubscribeTrafficLogsResponse = {
    list: TrafficLog[];
    total: number;
  };

  type GoogleLoginCallbackRequest = {
    code: string;
    state: string;
  };

  type HeartbeatResponse = {
    status: boolean;
    message?: string;
    timestamp?: number;
  };

  type Hysteria2 = {
    port: number;
    hop_ports: string;
    hop_interval: number;
    obfs_password: string;
    security_config: SecurityConfig;
  };

  type InviteConfig = {
    forced_invite: boolean;
    referral_percentage: number;
    only_first_purchase: boolean;
  };

  type LoginResponse = {
    token: string;
  };

  type MessageLog = {
    id: number;
    type: number;
    platform: string;
    to: string;
    subject: string;
    content: JsonValue;
    status: number;
    created_at: number;
  };

  type NodeConfig = {
    node_pull_interval: number;
    node_push_interval: number;
    traffic_report_threshold: number;
    ip_strategy: string;
  };

  type NodeRelay = {
    host: string;
    port: number;
    prefix: string;
  };

  type OAthLoginRequest = {
    /** google, facebook, apple, telegram, github etc. */
    method: string;
    redirect: string;
  };

  type OAuthCallbackPayload = {
    code?: string;
    state?: string;
    tgAuthResult?: string;
  };

  type OAuthLoginGetTokenRequest = {
    /** google, facebook, apple, telegram, github etc. */
    method: string;
    callback: OAuthCallbackPayload;
    invite?: string;
  };

  type OAuthLoginResponse = {
    redirect: string;
  };

  type Order = {
    id: number;
    user_id: number;
    order_no: string;
    source: string;
    type: number;
    quantity: number;
    price: number;
    amount: number;
    gift_amount: number;
    commission_deduction: number;
    discount: number;
    coupon: string;
    coupon_discount: number;
    commission?: number;
    payment: PaymentMethod;
    fee_amount: number;
    trade_no: string;
    status: number;
    subscribe_id: number;
    product_name: string;
    membership_plan?: MembershipPlan;
    created_at: number;
    updated_at: number;
  };

  type PublicOrder = {
    id: number;
    order_no: string;
    type: number;
    quantity: number;
    price: number;
    amount: number;
    gift_amount: number;
    commission_deduction: number;
    discount: number;
    coupon_discount: number;
    fee_amount: number;
    status: number;
    subscribe_id: number;
    product_name: string;
    membership_plan?: MembershipPlan;
    created_at: number;
    updated_at: number;
  };

  type OrderSubscribeProduct = {
    id: number;
    name: string;
    unit_price: number;
    unit_time: string;
    traffic: number;
    speed_limit: number;
    device_limit: number;
    show_original_price: boolean;
  };

  type OrderDetail = {
    id: number;
    user_id: number;
    order_no: string;
    source: string;
    type: number;
    quantity: number;
    price: number;
    amount: number;
    gift_amount: number;
    commission_deduction: number;
    discount: number;
    coupon: string;
    coupon_discount: number;
    commission?: number;
    payment: PaymentMethod;
    method: string;
    fee_amount: number;
    trade_no: string;
    status: number;
    subscribe_id: number;
    subscribe: OrderSubscribeProduct;
    membership_plan?: MembershipPlan;
    product_name: string;
    created_at: number;
    updated_at: number;
  };

  type MembershipPlan = {
    id: number;
    name: string;
    description: string;
    unit_price: number;
    duration_unit: string;
    duration_value: number;
    enabled: boolean;
    policy: MembershipPolicy;
    created_at?: number;
    updated_at?: number;
  };

  type MembershipPolicy = {
    monthly_consumption_limit_enabled: boolean;
    monthly_consumption_limit: number;
    monthly_order_limit_enabled: boolean;
    monthly_order_limit: number;
    subscription_expiry_limit_enabled: boolean;
    subscription_expiry_grace_days: number;
  };

  type PaymentConfig = {
    id: number;
    name: string;
    platform: string;
    description: string;
    icon?: string;
    domain?: string;
    config?: JsonValue;
    fee_mode: number;
    fee_percent?: number;
    fee_amount?: number;
    enable: boolean;
  };

  type PaymentMethod = {
    id: number;
    name: string;
    platform: string;
    description: string;
    icon: string;
    fee_mode: number;
    fee_percent: number;
    fee_amount: number;
  };

  type PaymentMethodListItem = {
    id: number;
    name: string;
    platform: string;
    description: string;
    icon: string;
    fee_mode: number;
    fee_percent: number;
    fee_amount: number;
    enable: boolean;
    notify_url: string;
  };

  type PlatformInfo = {
    platform: string;
    platform_url: string;
    platform_field_description: Record<string, string>;
  };

  type PlatformResponse = {
    list: PlatformInfo[];
  };

  type PreOrderResponse = {
    price: number;
    amount: number;
    discount: number;
    gift_amount: number;
    commission_deduction: number;
    coupon: string;
    coupon_discount: number;
    fee_amount: number;
  };

  type PreRenewalOrderResponse = {
    quantity: number;
    price: number;
    amount: number;
    discount: number;
    gift_amount: number;
    commission_deduction: number;
    coupon: string;
    coupon_discount: number;
    fee_amount: number;
    product: PublicSubscribe;
  };

  type PrivacyPolicyConfig = {
    privacy_policy: string;
  };

  type PublicAuthConfig = {
    email: PublicEmailAuthenticateConfig;
    register: PubilcRegisterConfig;
  };

  type PublicEmailAuthenticateConfig = {
    enable: boolean;
    enable_verify: boolean;
    enable_domain_suffix: boolean;
    domain_suffix_list: string[];
  };

  type PublicSubscribeConfig = {
    single_model: boolean;
    subscribe_path: string;
    subscribe_domain: string;
    pan_domain: boolean;
    user_agent_limit: boolean;
    user_agent_list: string[];
  };

  type Protocol = {
    type: string;
    port: number;
    enable: boolean;
    security?: string;
    sni?: string;
    allow_insecure?: boolean;
    fingerprint?: string;
    reality_server_addr?: string;
    reality_server_port?: number;
    reality_private_key?: string;
    reality_public_key?: string;
    reality_short_id?: string;
    transport?: string;
    host?: string;
    path?: string;
    service_name?: string;
    cipher?: string;
    server_key?: string;
    flow?: string;
    hop_ports?: string;
    hop_interval?: number;
    obfs_password?: string;
    disable_sni?: boolean;
    reduce_rtt?: boolean;
    udp_relay_mode?: string;
    congestion_controller?: string;
    /** mux, eg: off/low/medium/high */
    multiplex?: string;
    /** padding scheme */
    padding_scheme?: string;
    /** upload speed limit */
    up_mbps?: number;
    /** download speed limit */
    down_mbps?: number;
    /** obfs, 'none', 'http', 'tls' */
    obfs?: string;
    /** obfs host */
    obfs_host?: string;
    /** obfs path */
    obfs_path?: string;
    /** xhttp mode */
    xhttp_mode?: string;
    /** xhttp extra path */
    xhttp_extra?: string;
    /** encryption，'none', 'mlkem768x25519plus' */
    encryption?: string;
    /** encryption mode，'native', 'xorpub', 'random' */
    encryption_mode?: string;
    /** encryption rtt，'0rtt', '1rtt' */
    encryption_rtt?: string;
    /** encryption ticket */
    encryption_ticket?: string;
    /** encryption server padding */
    encryption_server_padding?: string;
    /** encryption private key */
    encryption_private_key?: string;
    /** encryption client padding */
    encryption_client_padding?: string;
    /** encryption password */
    encryption_password?: string;
    /** Traffic ratio, default is 1 */
    ratio?: number;
    /** Certificate mode, `none`｜`http`｜`dns`｜`self` */
    cert_mode?: string;
    /** DNS provider for certificate */
    cert_dns_provider?: string;
    /** Environment for DNS provider */
    cert_dns_env?: string;
  };

  type PubilcRegisterConfig = {
    stop_register: boolean;
    enable_ip_register_limit: boolean;
    ip_register_limit: number;
    ip_register_limit_duration: number;
  };

  type PubilcVerifyCodeConfig = {
    verify_code_interval: number;
  };

  type PurchaseOrderRequest = {
    subscribe_id: number;
    quantity: number;
    payment?: number;
    coupon?: string;
    request_id?: string;
  };

  type PurchaseOrderResponse = {
    order_no: string;
  };

  type QueryAnnouncementRequest = {
    page: number;
    size: number;
    pinned?: boolean;
    popup?: boolean;
  };

  type QueryAnnouncementResponse = {
    total: number;
    announcements: Announcement[];
  };

  type QueryDocumentDetailRequest = {
    id: number;
  };

  type QueryDocumentListResponse = {
    total: number;
    list: PublicDocumentListItem[];
  };

  type QueryOrderDetailRequest = {
    order_no: string;
  };

  type QueryOrderListRequest = {
    page: number;
    size: number;
    status?: number;
    search?: string;
  };

  type QueryOrderListResponse = {
    total: number;
    list: PublicOrder[];
  };

  type QuerySubscribeGroupListResponse = {
    list: SubscribeGroup[];
    total: number;
  };

  type QuerySubscribeListResponse = {
    list: PublicSubscribe[];
    total: number;
  };

  type QueryUserAffiliateCountResponse = {
    registers: number;
    total_commission: number;
  };

  type QueryUserAffiliateListRequest = {
    page: number;
    size: number;
  };

  type QueryUserAffiliateListResponse = {
    list: UserAffiliate[];
    total: number;
  };

  type RechargeOrderRequest = {
    amount: number;
    payment: number;
    request_id?: string;
  };

  type RechargeOrderResponse = {
    order_no: string;
  };

  type RegisterConfig = {
    stop_register: boolean;
    enable_trial: boolean;
    trial_subscribe: number;
    trial_time: number;
    trial_time_unit: string;
    enable_ip_register_limit: boolean;
    ip_register_limit: number;
    ip_register_limit_duration: number;
  };

  type RenewalOrderRequest = {
    user_subscribe_id: number;
    quantity: number;
    payment: number;
    coupon?: string;
    request_id?: string;
  };

  type RenewalOrderResponse = {
    order_no: string;
  };

  type ResetPasswordRequest = {
    identifier: string;
    email: string;
    password: string;
    code?: string;
    cf_token?: string;
  };

  type ResetSubscribeTrafficLog = {
    id: number;
    type: number;
    user_subscribe_id: number;
    order_no?: string;
    timestamp: number;
  };

  type ResetTrafficOrderRequest = {
    user_subscribe_id: number;
    payment: number;
    request_id?: string;
  };

  type PreResetTrafficOrderRequest = {
    user_subscribe_id: number;
    payment: number;
  };

  type PreResetTrafficOrderResponse = {
    price: number;
    amount: number;
    gift_amount: number;
    commission_deduction: number;
    fee_amount: number;
    product: PublicSubscribe;
  };

  type ResetTrafficOrderResponse = {
    order_no: string;
  };

  type ResetUserSubscribeTokenRequest = {
    user_subscribe_id: number;
  };

  type Response = {
    /** 状态码 */
    code?: number;
    /** 消息 */
    msg?: string;
    /** 数据 */
    data?: unknown;
  };

  type SecurityConfig = {
    sni: string;
    allow_insecure: boolean;
    fingerprint: string;
    pinned_peer_cert_sha256: string;
    verify_peer_cert_by_name: string;
    reality_server_addr: string;
    reality_server_port: number;
    reality_private_key: string;
    reality_public_key: string;
    reality_short_id: string;
  };

  type SendCodeRequest = {
    email: string;
    type: number;
  };

  type SendCodeResponse = {
    code?: string;
    status: boolean;
  };

  type ServerGroup = {
    id: number;
    name: string;
    description: string;
    created_at: number;
    updated_at: number;
  };

  type ServerRuleGroup = {
    id: number;
    icon: string;
    name: string;
    type: string;
    tags: string[];
    rules: string;
    enable: boolean;
    default: boolean;
    created_at: number;
    updated_at: number;
  };

  type Shadowsocks = {
    method: string;
    port: number;
    server_key: string;
  };

  type SiteConfig = {
    host: string;
    site_name: string;
    site_desc: string;
    site_logo: string;
    keywords: string;
    custom_html: string;
    custom_data: string;
  };

  type SiteCustomDataContacts = {
    email: string;
    telephone: string;
    address: string;
  };

  type SortItem = {
    id: number;
    sort: number;
  };

  type StripePayment = {
    method: string;
    client_secret: string;
    publishable_key: string;
  };

  type Subscribe = {
    id: number;
    name: string;
    language: string;
    description: string;
    unit_price: number;
    unit_time: string;
    discount: SubscribeDiscount[];
    replacement: number;
    inventory: number;
    traffic: number;
    speed_limit: number;
    device_limit: number;
    quota: number;
    nodes: number[];
    node_tags: string[];
    show: boolean;
    user_visible: boolean;
    sell: boolean;
    sort: number;
    deduction_ratio: number;
    allow_deduction: boolean;
    allow_renewal: boolean;
    allow_reset_traffic: boolean;
    reset_cycle: number;
    renewal_reset: boolean;
    show_original_price: boolean;
    display_template: string;
    badge: string;
    min_quantity: number;
    max_quantity: number;
    sale_starts_at: number;
    sale_ends_at: number;
    created_at: number;
    updated_at: number;
  };

  type PublicSubscribe = {
    id: number;
    name: string;
    description: string;
    unit_price: number;
    unit_time: string;
    discount: SubscribeDiscount[];
    replacement: number;
    traffic: number;
    speed_limit: number;
    device_limit: number;
    user_visible: boolean;
    sell: boolean;
    show_original_price: boolean;
    display_template: string;
    badge: string;
    min_quantity: number;
    max_quantity: number;
    sale_starts_at: number;
    sale_ends_at: number;
    available: boolean;
    availability: string;
  };

  type SubscribeClient = {
    id: number;
    name: string;
    description?: string;
    icon?: string;
    scheme?: string;
    enabled: boolean;
    is_default: boolean;
    download_link: DownloadLink;
  };

  type SubscribeConfig = {
    single_model: boolean;
    subscribe_path: string;
    subscribe_domain: string;
    pan_domain: boolean;
    user_agent_limit: boolean;
    user_agent_list: string;
  };

  type SubscribeDiscount = {
    quantity: number;
    discount: number;
  };

  type SubscribeGroup = {
    id: number;
    name: string;
    description: string;
    created_at: number;
    updated_at: number;
  };

  type SubscribeType = {
    subscribe_types: string[];
  };

  type TelegramConfig = {
    telegram_bot_token: string;
    telegram_group_url: string;
    telegram_notify: boolean;
    telegram_web_hook_domain: string;
  };

  type Ticket = {
    id: number;
    title: string;
    description?: string;
    images: string[];
    user_id: number;
    follow: Follow[];
    status: number;
    created_at: number;
    updated_at: number;
  };

  type TicketListItem = {
    id: number;
    title: string;
    user_id: number;
    status: number;
    updated_at: number;
  };

  type UserTicketListItem = {
    id: number;
    title: string;
    description?: string;
    status: number;
    updated_at: number;
  };

  type TimePeriod = {
    start_time: string;
    end_time: string;
    multiplier: number;
  };

  type TosConfig = {
    tos_content: string;
  };

  type TrafficLog = {
    id: number;
    server_id: number;
    user_id: number;
    subscribe_id: number;
    download: number;
    upload: number;
    timestamp: number;
  };

  type TransportConfig = {
    path: string;
    host: string;
    service_name: string;
  };

  type Trojan = {
    port: number;
    transport: string;
    transport_config: TransportConfig;
    security: string;
    security_config: SecurityConfig;
  };

  type Tuic = {
    port: number;
    disable_sni: boolean;
    reduce_rtt: boolean;
    udp_relay_mode: string;
    congestion_controller: string;
    security_config: SecurityConfig;
  };

  type User = {
    id: number;
    avatar: string;
    balance: number;
    commission: number;
    referral_percentage: number;
    only_first_purchase: boolean;
    gift_amount: number;
    member_expired_at: number;
    is_member: boolean;
    telegram: number;
    refer_code: string;
    referer_id: number;
    enable: boolean;
    is_admin: boolean;
    enable_balance_notify: boolean;
    enable_login_notify: boolean;
    enable_subscribe_notify: boolean;
    enable_trade_notify: boolean;
    auth_methods: UserAuthMethod[];
    rules: string[];
    created_at: number;
    updated_at: number;
    deleted_at?: number;
  };

  type UserAffiliate = {
    avatar: string;
    identifier: string;
    registered_at: number;
    enable: boolean;
  };

  type UserAuthMethod = {
    auth_type: string;
    auth_identifier: string;
    verified: boolean;
  };

  type UserDevice = {
    id: number;
    ip: string;
    country_short?: string;
    region?: string;
    location?: string;
    identifier: string;
    user_agent: string;
    online: boolean;
    enabled: boolean;
    created_at: number;
    updated_at: number;
  };

  type UserLoginLog = {
    id: number;
    user_id: number;
    login_ip: string;
    country_short?: string;
    region?: string;
    location?: string;
    user_agent: string;
    success: boolean;
    timestamp: number;
  };

  type UserLoginRequest = {
    identifier: string;
    email: string;
    password: string;
    cf_token?: string;
  };

  type UserRegisterRequest = {
    identifier: string;
    email: string;
    password: string;
    invite?: string;
    code?: string;
    cf_token?: string;
  };

  type UserSubscribe = {
    id: number;
    user_id: number;
    order_id: number;
    subscribe_id: number;
    subscribe: Subscribe;
    product_name: string;
    start_time: number;
    expire_time: number;
    finished_at: number;
    reset_time: number;
    traffic: number;
    download: number;
    upload: number;
    token: string;
    status: number;
    short: string;
    created_at: number;
    updated_at: number;
  };

  type UserSubscribeLog = {
    id: number;
    user_id: number;
    user_subscribe_id: number;
    token: string;
    ip: string;
    country_short?: string;
    region?: string;
    location?: string;
    user_agent: string;
    timestamp: number;
  };

  type VeifyConfig = {
    turnstile_site_key: string;
    enable_login_verify: boolean;
    enable_register_verify: boolean;
    enable_reset_password_verify: boolean;
  };

  type VerifyCodeConfig = {
    verify_code_expire_time: number;
    verify_code_limit: number;
    verify_code_interval: number;
  };

  type VerifyConfig = {
    turnstile_site_key: string;
    turnstile_secret: string;
    enable_login_verify: boolean;
    enable_register_verify: boolean;
    enable_reset_password_verify: boolean;
  };

  type Vless = {
    port: number;
    flow: string;
    transport: string;
    transport_config: TransportConfig;
    security: string;
    security_config: SecurityConfig;
  };

  type Vmess = {
    port: number;
    transport: string;
    transport_config: TransportConfig;
    security: string;
    security_config: SecurityConfig;
  };
}
