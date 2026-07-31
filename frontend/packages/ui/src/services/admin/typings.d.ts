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

  type AnnouncementListItem = {
    id: number;
    title: string;
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

  type BatchDeleteCouponRequest = {
    ids: number[];
  };

  type BatchDeleteDocumentRequest = {
    ids: number[];
  };

  type BatchDeleteSubscribeGroupRequest = {
    ids: number[];
  };

  type BatchDeleteSubscribeRequest = {
    ids: number[];
  };

  type BatchDeleteUserRequest = {
    ids: number[];
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

  type CouponListItem = {
    id: number;
    name: string;
    code: string;
    count: number;
    type: number;
    discount: number;
    start_time: number;
    expire_time: number;
    user_limit: number;
    used_count: number;
    enable: boolean;
  };

  type CreateAdsRequest = {
    title: string;
    type: string;
    content: string;
    description: string;
    target_url: string;
    start_time: number;
    end_time: number;
    status: number;
  };

  type CreateAnnouncementRequest = {
    title: string;
    content: string;
  };

  type CreateCouponRequest = {
    name: string;
    code?: string;
    count?: number;
    type: number;
    discount: number;
    start_time: number;
    expire_time: number;
    user_limit?: number;
    subscribe?: number[];
    used_count?: number;
    enable?: boolean;
  };

  type CreateDocumentRequest = {
    title: string;
    content: string;
    tags?: string[];
    show: boolean;
  };

  type CreateNodeRequest = {
    name: string;
    tags?: string[];
    port: number;
    address: string;
    server_id: number;
    inbound_alias: string;
    protocol?: string;
    enabled: boolean;
  };

  type CreateOrderRequest = {
    user_id: number;
    type: number;
    quantity?: number;
    price: number;
    amount: number;
    discount?: number;
    coupon?: string;
    coupon_discount?: number;
    commission: number;
    fee_amount: number;
    payment_id: number;
    trade_no?: string;
    status?: number;
    subscribe_id?: number;
  };

  type CreatePaymentMethodRequest = {
    name: string;
    platform: string;
    description: string;
    icon?: string;
    domain?: string;
    config: JsonValue;
    fee_mode: number;
    fee_percent?: number;
    fee_amount?: number;
    enable: boolean;
  };

  type CreateServerRequest = {
    name: string;
    country?: string;
    city?: string;
    address: string;
    sort?: number;
  };

  type CreateSubscribeApplicationRequest = {
    name: string;
    description?: string;
    icon?: string;
    scheme?: string;
    user_agent: string;
    enabled?: boolean;
    is_default: boolean;
    template: string;
    output_format: string;
    download_link: DownloadLink;
  };

  type CreateSubscribeGroupRequest = {
    name: string;
    description: string;
  };

  type CreateSubscribeRequest = {
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
  };

  type CreateTicketFollowRequest = {
    ticket_id: number;
    from?: string;
    type: number;
    content?: string;
    images?: string[];
  };

  type CreateUserAuthMethodRequest = {
    user_id: number;
    auth_type: string;
    auth_identifier: string;
  };

  type CreateUserRequest = {
    email: string;
    password: string;
    product_id: number;
    duration: number;
    referral_percentage: number;
    only_first_purchase: boolean;
    referer_user: string;
    refer_code: string;
    balance: number;
    commission: number;
    gift_amount: number;
    is_admin: boolean;
  };

  type CreateUserSubscribeRequest = {
    user_id: number;
    expired_at: number;
    traffic: number;
    subscribe_id: number;
  };

  type IssueUserSubscribeOrderRequest = {
    user_id: number;
    subscribe_id: number;
    quantity: number;
    amount_mode: "complimentary" | "current_price";
    trade_no?: string;
  };

  type IssueUserSubscribeOrderQuote = {
    product_id: number;
    product_name: string;
    quantity: number;
    unit_price: number;
    list_price: number;
    current_amount: number;
    recorded_amount: number;
    discount: number;
    traffic: number;
    speed_limit: number;
    device_limit: number;
    start_time: number;
    expire_time: number;
    amount_mode: "complimentary" | "current_price";
    user_enabled: boolean;
  };

  type IssueUserSubscribeOrderResponse = {
    order_id: number;
    order_no: string;
    order_status: number;
    activation_queued: boolean;
    quote: IssueUserSubscribeOrderQuote;
  };

  type BatchResetUserSubscribeTrafficRequest = {
    scope: "all" | "users" | "subscriptions";
    user_ids?: number[];
    user_subscribe_ids?: number[];
    confirmation?: string;
  };

  type BatchResetUserSubscribeTrafficResponse = {
    batch_id: string;
    reset_at: number;
    reset_count: number;
    reactivated: number;
  };

  type CreateXrayTemplateRequest = {
    name: string;
    type: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    config: JsonObject;
    description?: string;
    enabled?: boolean;
    config_template?: string;
    variables_schema?: JsonObject;
    default_variables?: JsonObject;
    subscription_meta?: JsonObject;
  };

  type CurrencyConfig = {
    access_key: string;
    currency_unit: string;
    currency_symbol: string;
  };

  type DeleteAdsRequest = {
    id: number;
  };

  type DeleteAnnouncementRequest = {
    id: number;
  };

  type DeleteCouponRequest = {
    id: number;
  };

  type DeleteDocumentRequest = {
    id: number;
  };

  type DeleteNodeRequest = {
    id: number;
  };

  type DeletePaymentMethodRequest = {
    id: number;
  };

  type DeleteServerRequest = {
    id: number;
  };

  type DeleteSubscribeApplicationRequest = {
    id: number;
  };

  type DeleteSubscribeGroupRequest = {
    id: number;
  };

  type DeleteSubscribeRequest = {
    id: number;
  };

  type DeleteUserAuthMethodRequest = {
    user_id: number;
    auth_type: string;
  };

  type DeleteUserDeivceRequest = {
    id: number;
  };

  type DeleteUserParams = {
    id: number;
  };

  type DeleteUserSubscribeRequest = {
    user_subscribe_id: number;
  };

  type DeleteXrayTemplateRequest = {
    id: number;
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

  type DocumentListItem = {
    id: number;
    title: string;
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

  type FilterBalanceLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterBalanceLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterXrayTemplateListParams = {
    page: number;
    size: number;
    type?: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    search?: string;
    enabled?: boolean;
  };

  type FilterXrayTemplateListResponse = {
    total: number;
    list: XrayTemplateListItem[];
  };

  type QueryXrayTemplateCatalogParams = {
    page: number;
    size: number;
    type?: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    search?: string;
    enabled?: boolean;
  };

  type QueryXrayTemplateCatalogRequest = {
    page: number;
    size: number;
    type?: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    search?: string;
    enabled?: boolean;
  };

  type QueryXrayTemplateCatalogResponse = {
    total: number;
    list: XrayTemplateCatalogItem[];
  };

  type FilterBalanceLogResponse = {
    total: number;
    list: BalanceLog[];
  };

  type FilterCommissionLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterCommissionLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterCommissionLogResponse = {
    total: number;
    list: CommissionLog[];
  };

  type FilterEmailLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
  };

  type FilterEmailLogResponse = {
    total: number;
    list: MessageLog[];
  };

  type FilterGiftLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterGiftLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterGiftLogResponse = {
    total: number;
    list: GiftLog[];
  };

  type FilterLoginLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterLoginLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterLoginLogResponse = {
    total: number;
    list: LoginLog[];
  };

  type FilterLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
  };

  type FilterProxyAccessLogParams = {
    start_time?: number;
    end_time?: number;
    server_id?: number;
    user_id?: number;
    user_subscribe_id?: number;
    source_ip?: string;
    destination?: string;
    status?: string;
    size?: number;
    cursor?: string;
  };

  type FilterProxyAccessLogResponse = {
    list: ProxyAccessLog[];
    next_cursor?: string;
    has_more: boolean;
  };

  type FilterNodeListParams = {
    page: number;
    size: number;
    search?: string;
  };

  type FilterNodeListRequest = {
    page: number;
    size: number;
    search?: string;
  };

  type FilterNodeListResponse = {
    total: number;
    list: NodeListItem[];
  };

  type FilterRegisterLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterRegisterLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
  };

  type FilterRegisterLogResponse = {
    total: number;
    list: RegisterLog[];
  };

  type FilterResetSubscribeLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_subscribe_id?: number;
  };

  type FilterResetSubscribeLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_subscribe_id?: number;
  };

  type FilterResetSubscribeLogResponse = {
    total: number;
    list: ResetSubscribeLog[];
  };

  type FilterServerListParams = {
    page: number;
    size: number;
    search?: string;
  };

  type FilterServerListRequest = {
    page: number;
    size: number;
    search?: string;
  };

  type FilterServerListResponse = {
    total: number;
    list: ServerListItem[];
  };

  type QueryServerNetworkRateHistoryParams = {
    server_ids: string;
    start_time?: number;
    end_time?: number;
    interval?: string;
  };

  type QueryServerNetworkRateHistoryRequest = {
    server_ids: string;
    start_time?: number;
    end_time?: number;
    interval?: string;
  };

  type QueryServerNetworkRateHistoryResponse = {
    start_time: number;
    end_time: number;
    interval_ms: number;
    series: ServerNetworkRateSeries[];
  };

  type QueryServerNodeTokenParams = {
    id: number;
  };

  type QueryServerNodeTokenResponse = {
    token: string;
  };

  type FilterServerTrafficLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    server_id?: number;
  };

  type FilterServerTrafficLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    server_id?: number;
  };

  type FilterServerTrafficLogResponse = {
    total: number;
    list: ServerTrafficLog[];
  };

  type FilterSubscribeLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
    user_subscribe_id?: number;
  };

  type FilterSubscribeLogRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
    user_subscribe_id?: number;
  };

  type FilterSubscribeLogResponse = {
    total: number;
    list: SubscribeLog[];
  };

  type FilterSubscribeTrafficRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    user_id?: number;
    user_subscribe_id?: number;
  };

  type FilterSubscribeTrafficResponse = {
    total: number;
    list: UserSubscribeTrafficLog[];
  };

  type FilterTrafficLogDetailsParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    server_id?: number;
    subscribe_id?: number;
    user_id?: number;
  };

  type FilterTrafficLogDetailsRequest = {
    page?: number;
    size?: number;
    date?: string;
    search?: string;
    server_id?: number;
    subscribe_id?: number;
    user_id?: number;
  };

  type FilterTrafficLogDetailsResponse = {
    total: number;
    list: TrafficLogDetails[];
  };

  type FilterUserSubscribeTrafficLogParams = {
    page: number;
    size: number;
    date?: string;
    search?: string;
    user_id?: number;
    user_subscribe_id?: number;
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

  type GetAdsDetailParams = {
    id: number;
  };

  type GetAdsDetailRequest = {
    id: number;
  };

  type GetAdsListParams = {
    page: number;
    size: number;
    status?: number;
    search?: string;
  };

  type GetAdsListRequest = {
    page: number;
    size: number;
    status?: number;
    search?: string;
  };

  type GetAdsListResponse = {
    total: number;
    list: AdsListItem[];
  };

  type GetAnnouncementListParams = {
    page: number;
    size: number;
    show?: boolean;
    pinned?: boolean;
    popup?: boolean;
    search?: string;
  };

  type GetAnnouncementListRequest = {
    page: number;
    size: number;
    show?: boolean;
    pinned?: boolean;
    popup?: boolean;
    search?: string;
  };

  type GetAnnouncementListResponse = {
    total: number;
    list: AnnouncementListItem[];
  };

  type GetAnnouncementParams = {
    id: number;
  };

  type GetAnnouncementRequest = {
    id: number;
  };

  type GetAuthMethodConfigParams = {
    method: string;
  };

  type GetAuthMethodConfigRequest = {
    method: string;
  };

  type GetAuthMethodListResponse = {
    list: AuthMethodSummary[];
  };

  type GetAvailablePaymentMethodsResponse = {
    list: PaymentMethod[];
  };

  type GetCouponParams = {
    id: number;
  };

  type GetCouponRequest = {
    id: number;
  };

  type GetCouponListParams = {
    page: number;
    size: number;
    subscribe?: number;
    search?: string;
  };

  type GetCouponListRequest = {
    page: number;
    size: number;
    subscribe?: number;
    search?: string;
  };

  type GetCouponListResponse = {
    total: number;
    list: CouponListItem[];
  };

  type GetDetailRequest = {
    id: number;
  };

  type GetDocumentDetailParams = {
    id: number;
  };

  type GetDocumentDetailRequest = {
    id: number;
  };

  type GetDocumentListParams = {
    page: number;
    size: number;
    tag?: string;
    search?: string;
  };

  type GetDocumentListRequest = {
    page: number;
    size: number;
    tag?: string;
    search?: string;
  };

  type GetDocumentListResponse = {
    total: number;
    list: DocumentListItem[];
  };

  type GetMessageLogListParams = {
    page: number;
    size: number;
    type: number;
    search?: string;
  };

  type GetMessageLogListRequest = {
    page: number;
    size: number;
    type: number;
    search?: string;
  };

  type GetMessageLogListResponse = {
    total: number;
    list: MessageLog[];
  };

  type GetNodeMultiplierResponse = {
    periods: TimePeriod[];
  };

  type GetOrderListParams = {
    page: number;
    size: number;
    user_id?: number;
    status?: number;
    subscribe_id?: number;
    search?: string;
  };

  type GetOrderDetailParams = {
    id: number;
  };

  type GetOrderDetailRequest = {
    id: number;
  };

  type GetOrderListRequest = {
    page: number;
    size: number;
    user_id?: number;
    status?: number;
    subscribe_id?: number;
    search?: string;
  };

  type GetOrderListResponse = {
    total: number;
    list: OrderListItem[];
  };

  type GetPaymentMethodListParams = {
    page: number;
    size: number;
    platform?: string;
    search?: string;
    enable?: boolean;
  };

  type GetPaymentMethodListRequest = {
    page: number;
    size: number;
    platform?: string;
    search?: string;
    enable?: boolean;
  };

  type GetPaymentMethodListResponse = {
    total: number;
    list: PaymentMethodListItem[];
  };

  type GetPaymentMethodParams = {
    id: number;
  };

  type GetPaymentMethodRequest = {
    id: number;
  };

  type GetSubscribeApplicationListParams = {
    page: number;
    size: number;
    search?: string;
    enabled?: boolean;
  };

  type GetSubscribeApplicationListRequest = {
    page: number;
    size: number;
    search?: string;
    enabled?: boolean;
  };

  type GetSubscribeApplicationListResponse = {
    total: number;
    list: SubscribeApplicationListItem[];
  };

  type GetSubscribeDetailsParams = {
    id: number;
  };

  type GetSubscribeDetailsRequest = {
    id: number;
  };

  type GetSubscribeGroupListResponse = {
    list: SubscribeGroup[];
    total: number;
  };

  type GetSubscribeListParams = {
    page: number;
    size: number;
    language?: string;
    search?: string;
  };

  type GetSubscribeListRequest = {
    page: number;
    size: number;
    language?: string;
    search?: string;
  };

  type GetSubscribeListResponse = {
    list: SubscribeItem[];
    total: number;
  };

  type GetTicketListParams = {
    page: number;
    size: number;
    user_id?: number;
    status?: number;
    search?: string;
  };

  type GetTicketListRequest = {
    page: number;
    size: number;
    user_id?: number;
    status?: number;
    search?: string;
  };

  type GetTicketListResponse = {
    total: number;
    list: TicketListItem[];
  };

  type GetTicketParams = {
    id: number;
  };

  type GetTicketRequest = {
    id: number;
  };

  type GetUserAuthMethodParams = {
    user_id: number;
  };

  type GetUserAuthMethodRequest = {
    user_id: number;
  };

  type GetUserAuthMethodResponse = {
    auth_methods: UserAuthMethod[];
  };

  type RevealUserPrivateIdentityRequest = {
    user_id: number;
    reason:
      | "customer_support"
      | "account_verification"
      | "security_incident"
      | "billing_dispute"
      | "compliance";
  };

  type RevealUserPrivateIdentityResponse = {
    user_id: number;
    email: string;
    auth_methods: UserAuthMethod[];
    revealed_at: number;
    expires_at: number;
  };

  type GetUserDetailParams = {
    id: number;
  };

  type GetUserListParams = {
    page: number;
    size: number;
    search?: string;
    status?: "enabled" | "disabled" | "deleted";
    user_id?: number;
    unscoped?: boolean;
    subscribe_id?: number;
    user_subscribe_id?: number;
  };

  type GetUserListRequest = {
    page: number;
    size: number;
    search?: string;
    status?: string;
    user_id?: number;
    unscoped?: boolean;
    subscribe_id?: number;
    user_subscribe_id?: number;
  };

  type GetUserListResponse = {
    total: number;
    list: UserListItem[];
  };

  type GetUserLoginLogsParams = {
    page: number;
    size: number;
    user_id: number;
  };

  type GetUserLoginLogsRequest = {
    page: number;
    size: number;
    user_id: number;
  };

  type GetUserLoginLogsResponse = {
    list: UserLoginLog[];
    total: number;
  };

  type GetUserSubscribeByIdParams = {
    id: number;
  };

  type GetUserSubscribeByIdRequest = {
    id: number;
  };

  type GetUserSubscribeDevicesParams = {
    page: number;
    size: number;
    user_id: number;
    subscribe_id: number;
  };

  type GetUserSubscribeDevicesRequest = {
    page: number;
    size: number;
    user_id: number;
    subscribe_id: number;
  };

  type GetUserSubscribeDevicesResponse = {
    list: UserDevice[];
    total: number;
  };

  type GetUserSubscribeAccessParams = {
    user_subscribe_id: number;
  };

  type GetUserSubscribeAccessRequest = {
    user_subscribe_id: number;
  };

  type GetUserSubscribeListRequest = {
    page: number;
    size: number;
    user_id: number;
  };

  type GetUserSubscribeListResponse = {
    list: UserSubscribeListItem[];
    total: number;
  };

  type GetUserSubscribeLogsParams = {
    page: number;
    size: number;
    user_id: number;
    subscribe_id?: number;
  };

  type GetUserSubscribeLogsRequest = {
    page: number;
    size: number;
    user_id: number;
    subscribe_id?: number;
  };

  type GetUserSubscribeLogsResponse = {
    list: UserSubscribeLog[];
    total: number;
  };

  type GetUserSubscribeParams = {
    page: number;
    size: number;
    user_id: number;
  };

  type GetUserSubscribeResetTrafficLogsParams = {
    page: number;
    size: number;
    user_subscribe_id: number;
  };

  type GetUserSubscribeResetTrafficLogsRequest = {
    page: number;
    size: number;
    user_subscribe_id: number;
  };

  type GetUserSubscribeResetTrafficLogsResponse = {
    list: ResetSubscribeTrafficLog[];
    total: number;
  };

  type GetUserSubscribeTrafficLogsParams = {
    page: number;
    size: number;
    user_id: number;
    subscribe_id: number;
    start_time: number;
    end_time: number;
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

  type GetUserSubscribeTrafficSeriesRequest = {
    user_id: number;
    user_subscribe_id: number;
    period?: "5m" | "15m" | "1h" | "24h" | "7d" | "30d";
  };

  type GiftLog = {
    type: number;
    user_id: number;
    order_no: string;
    subscribe_id: number;
    amount: number;
    balance: number;
    remark?: string;
    timestamp: number;
  };

  type HasMigrateSeverNodeResponse = {
    has_migrate: boolean;
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

  type KickOfflineRequest = {
    id: number;
  };

  type LoginLog = {
    user_id: number;
    method: string;
    login_ip: string;
    country_short?: string;
    region?: string;
    location?: string;
    user_agent: string;
    success: boolean;
    timestamp: number;
  };

  type LogResponse = {
    list: SystemLogEntry[];
  };

  type SystemLogEntry = {
    timestamp?: string;
    level?: string;
    caller?: string;
    content?: JsonValue;
    duration?: string;
    trace?: string;
    span?: string;
    truncated?: boolean;
    fields?: JsonObject;
  };

  type LogSetting = {
    auto_clear: boolean;
    clear_days: number;
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

  type MigrateServerNodeResponse = {
    succee: number;
    fail: number;
    message?: string;
  };

  type Node = {
    id: number;
    name: string;
    tags: string[];
    port: number;
    address: string;
    server_id: number;
    inbound_alias: string;
    protocol?: string;
    enabled: boolean;
    sort?: number;
    created_at: number;
    updated_at: number;
  };

  type NodeListItem = {
    id: number;
    name: string;
    tags: string[];
    port: number;
    address: string;
    server_id: number;
    inbound_alias: string;
    protocol?: string;
    enabled: boolean;
    sort?: number;
    created_at: number;
    updated_at: number;
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

  type OrderListItem = {
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
    coupon_discount: number;
    payment: PaymentMethodSummary;
    fee_amount: number;
    trade_no: string;
    status: number;
    subscribe_id: number;
    product_name: string;
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
    product_name: string;
    membership_plan?: MembershipPlan;
    created_at: number;
    updated_at: number;
  };

  type OrdersStatistics = {
    date?: string;
    amount_total: number;
    new_order_amount: number;
    renewal_order_amount: number;
    membership_amount: number;
    reset_traffic_amount: number;
  };

  type StatisticsPeriodParams = {
    period?: string;
  };

  type StatisticsPeriodRequest = {
    period?: string;
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

  type PaymentMethodSummary = {
    id: number;
    name: string;
    platform: string;
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

  type PreRenewalOrderRequest = {
    user_subscribe_id: number;
    quantity: number;
    payment: number;
    coupon?: string;
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

  type PreViewNodeMultiplierResponse = {
    current_time: string;
    ratio: number;
  };

  type PreviewSubscribeTemplateParams = {
    id: number;
  };

  type GetSubscribeApplicationParams = {
    id: number;
  };

  type GetSubscribeApplicationRequest = {
    id: number;
  };

  type PreviewSubscribeTemplateRequest = {
    id: number;
  };

  type PreviewSubscribeTemplateResponse = {
    /** 预览的模板内容 */
    template: string;
  };

  type PrivacyPolicyConfig = {
    privacy_policy: string;
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

  type QueryIPLocationParams = {
    ip: string;
  };

  type QueryIPLocationRequest = {
    ip: string;
  };

  type QueryIPLocationResponse = {
    country: string;
    country_short?: string;
    region?: string;
    city: string;
    location?: string;
  };

  type QueryNodeTagResponse = {
    tags: string[];
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

  type RegisterLog = {
    user_id: number;
    auth_method: string;
    identifier: string;
    register_ip: string;
    country_short?: string;
    region?: string;
    location?: string;
    user_agent: string;
    timestamp: number;
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

  type ResetAllSubscribeTokenResponse = {
    success: boolean;
  };

  type ResetSortRequest = {
    sort: SortItem[];
  };

  type ResetSubscribeLog = {
    type: number;
    user_id: number;
    user_subscribe_id: number;
    order_no?: string;
    timestamp: number;
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

  type ResetTrafficOrderResponse = {
    order_no: string;
  };

  type ResetUserSubscribeTokenRequest = {
    user_subscribe_id: number;
  };

  type ResetUserSubscribeTrafficRequest = {
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

  type RevenueStatisticsResponse = {
    period: string;
    granularity: string;
    series: OrdersStatistics[];
    today: OrdersStatistics;
    monthly: OrdersStatistics;
    all: OrdersStatistics;
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

  type Server = {
    id: number;
    name: string;
    country: string;
    city: string;
    address: string;
    sort: number;
    last_reported_at: number;
    status: ServerStatus;
    created_at: number;
    updated_at: number;
  };

  type ServerListItem = {
    id: number;
    name: string;
    country: string;
    city: string;
    address: string;
    sort: number;
    last_reported_at: number;
    status: ServerStatus;
    node_count: number;
    template_count: number;
    created_at: number;
    updated_at: number;
  };

  type ServerXrayTemplateBinding = {
    template_id: number;
    sort?: number;
    enabled?: boolean;
    alias?: string;
    variables?: JsonObject;
    subscription_enabled?: boolean;
    subscription_name?: string;
    subscription_variables?: JsonObject;
  };

  type BindServerXrayTemplatesRequest = {
    server_id: number;
    bindings: ServerXrayTemplateBinding[];
  };

  type PreviewServerXrayConfigRequest = {
    server_id: number;
    bindings: ServerXrayTemplateBinding[];
  };

  type QueryServerXrayConfigResponse = {
    inbounds: JsonValue[];
    outbounds: JsonValue[];
    dns?: JsonValue;
    routing?: JsonValue;
    geodata?: JsonValue;
    templates: XrayTemplate[];
  };

  type PreviewServerXrayConfigResponse = QueryServerXrayConfigResponse & {
    subscription_proxies: JsonValue[];
    variables: Record<
      string,
      {
        type: string;
        alias: string;
        template_id: number;
        template_name: string;
        sort: number;
        config_variables: JsonObject;
        default_variables: JsonObject;
        binding_variables: JsonObject;
        merged_variables: JsonObject;
        subscription_variables: JsonObject;
        subscription_merged_variables: JsonObject;
        ref: JsonObject;
      }
    >;
  };

  type QueryServerXrayTemplateListParams = {
    server_id: number;
    include_template?: boolean;
    include_status?: boolean;
    include_desired_config?: boolean;
  };

  type ServerXrayTemplate = {
    id: number;
    server_id: number;
    template_id: number;
    sort: number;
    enabled: boolean;
    alias: string;
    variables: JsonObject;
    subscription_enabled: boolean;
    subscription_name: string;
    subscription_variables: JsonObject;
    template?: XrayTemplate;
    created_at: number;
    updated_at: number;
  };

  type QueryServerXrayTemplateListResponse = {
    list: ServerXrayTemplate[];
    desired_config_hash?: string;
    desired_config_error?: string;
    status?: ServerStatus;
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

  type ServerStatus = {
    cpu: number;
    mem: number;
    disk: number;
    status: string;
    net_rx_bps?: number;
    net_tx_bps?: number;
    net_rx_bytes?: number;
    net_tx_bytes?: number;
    xray_rx_bps?: number;
    xray_tx_bps?: number;
    xray_rx_bytes?: number;
    xray_tx_bytes?: number;
    xray_stats_error?: string;
    connections?: number;
    system_connections?: number;
    system_inbound_connections?: number;
    system_outbound_connections?: number;
    xray_connections?: number;
    xray_inbound_connections?: number;
    xray_outbound_connections?: number;
    online_users?: number;
    xray_running?: boolean;
    xray_version?: string;
    updated_at?: number;
    last_seen?: number;
    load1?: number;
    load5?: number;
    load15?: number;
    uptime?: number;
    agent_version?: string;
    config_version?: string;
    config_sync_status?: string;
    config_apply_status?: string;
    last_config_hash?: string;
    running_config_hash?: string;
    pending_config_hash?: string;
    last_sync_at?: number;
    last_apply_at?: number;
    last_config_error?: string;
    last_apply_error?: string;
  };

  type ServerNetworkRatePoint = {
    timestamp: number;
    net_rx_bps: number;
    net_tx_bps: number;
    xray_rx_bps: number;
    xray_tx_bps: number;
  };

  type ServerNetworkRateSeries = {
    server_id: number;
    points: ServerNetworkRatePoint[];
  };

  type DashboardRealtimeResponse = {
    type: string;
    full?: boolean;
    seq?: number;
    reason?: string;
    updated_at: number;
    online_users: number;
    online_user_series: {
      last_day: Array<{
        timestamp: number;
        value: number;
      }>;
      last_7_days: Array<{
        timestamp: number;
        value: number;
      }>;
    };
    pending_tickets: number;
    servers: {
      total: number;
      online: number;
      offline: number;
      xray_running: number;
      xray_stopped: number;
      config_ok: number;
      config_pending: number;
      config_failed: number;
      stats_error: number;
      reported: number;
      not_reported: number;
    };
    network: {
      system_rx_bps: number;
      system_tx_bps: number;
      xray_rx_bps: number;
      xray_tx_bps: number;
    };
    network_series: Array<{
      timestamp: number;
      system_rx_bps: number;
      system_tx_bps: number;
      xray_rx_bps: number;
      xray_tx_bps: number;
    }>;
    connections: {
      system_inbound: number;
      system_outbound: number;
      xray_inbound: number;
      xray_outbound: number;
    };
    resources: {
      avg_cpu: number;
      avg_mem: number;
      avg_disk: number;
      max_cpu: number;
      max_mem: number;
      max_disk: number;
    };
    business: {
      revenue_today: number;
      new_order_amount: number;
      renewal_order_amount: number;
      membership_amount: number;
      reset_traffic_amount: number;
      register_today: number;
      new_order_count: number;
      renewal_order_count: number;
      membership_order_count: number;
    };
    traffic: {
      today_upload: number;
      today_download: number;
      monthly_upload: number;
      monthly_download: number;
      today_series: Array<{
        timestamp: number;
        upload: number;
        download: number;
        total: number;
      }>;
      server_ranking_today: ServerTrafficData[];
      user_ranking_today: UserTrafficData[];
      server_ranking_monthly: ServerTrafficData[];
      user_ranking_monthly: UserTrafficData[];
    };
    alerts: Array<{
      level: string;
      server_id?: number;
      title: string;
      message?: string;
    }>;
    activities: Array<{
      id: string;
      type: string;
      title: string;
      subject?: string;
      detail?: string;
      ip?: string;
      location?: string;
      status?: string;
      amount?: number;
      user_id?: number;
      created_at: number;
    }>;
  };

  type ServerTrafficData = {
    server_id: number;
    name: string;
    upload: number;
    download: number;
  };

  type ServerTrafficLog = {
    /** Server ID */
    server_id: number;
    /** Upload traffic in bytes */
    upload: number;
    /** Download traffic in bytes */
    download: number;
    /** Total traffic in bytes (Upload + Download) */
    total: number;
    /** Date in YYYY-MM-DD format */
    date: string;
    /** Whether to show detailed traffic */
    details: boolean;
  };

  type SetNodeMultiplierRequest = {
    periods: TimePeriod[];
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
    display_template?: "standard" | "popular" | "premium" | "compact";
    badge?: string;
    min_quantity?: number;
    max_quantity?: number;
    sale_starts_at?: number;
    sale_ends_at?: number;
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

  type SubscribeApplication = {
    id: number;
    name: string;
    description?: string;
    icon?: string;
    scheme?: string;
    user_agent: string;
    enabled: boolean;
    is_default: boolean;
    template?: string;
    output_format: string;
    download_link: DownloadLink;
    created_at: number;
    updated_at: number;
  };

  type SubscribeApplicationListItem = {
    id: number;
    name: string;
    description?: string;
    icon?: string;
    scheme?: string;
    user_agent: string;
    enabled: boolean;
    is_default: boolean;
    output_format: string;
    download_link: DownloadLink;
    created_at: number;
    updated_at: number;
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

  type SubscribeItem = {
    id: number;
    name: string;
    language: string;
    unit_price: number;
    unit_time: string;
    replacement: number;
    inventory: number;
    traffic: number;
    speed_limit: number;
    device_limit: number;
    quota: number;
    show: boolean;
    user_visible: boolean;
    sell: boolean;
    allow_renewal: boolean;
    reset_cycle: number;
    display_template?: "standard" | "popular" | "premium" | "compact";
    badge?: string;
    min_quantity?: number;
    max_quantity?: number;
    sale_starts_at?: number;
    sale_ends_at?: number;
    sort: number;
    created_at: number;
    updated_at: number;
    sold: number;
  };

  type SubscribeLog = {
    user_id: number;
    token: string;
    user_agent: string;
    client_ip: string;
    country_short?: string;
    region?: string;
    location?: string;
    user_subscribe_id: number;
    timestamp: number;
  };

  type SubscribeSortRequest = {
    sort: SortItem[];
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

  type TestEmailSendRequest = {
    email: string;
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

  type TimePeriod = {
    start_time: string;
    end_time: string;
    multiplier: number;
  };

  type ToggleNodeStatusRequest = {
    id: number;
    enable: boolean;
  };

  type ToggleUserSubscribeStatusRequest = {
    user_subscribe_id: number;
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

  type TrafficLogDetails = {
    id: number;
    server_id: number;
    user_id: number;
    subscribe_id: number;
    download: number;
    upload: number;
    timestamp: number;
  };

  type TrafficSeriesPoint = {
    timestamp: number;
    download: number;
    upload: number;
  };

  type TrafficSeriesResponse = {
    start_time: number;
    end_time: number;
    interval_seconds: number;
    download: number;
    upload: number;
    list: TrafficSeriesPoint[];
  };

  type ProxyAccessLog = {
    event_id: string;
    timestamp: number;
    server_id: number;
    user_id?: number;
    user_subscribe_id?: number;
    source_ip?: string;
    source_port?: number;
    destination?: string;
    destination_port?: number;
    network?: string;
    status: string;
    outbound_tag?: string;
    reason?: string;
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

  type UpdateAdsRequest = {
    id: number;
    title?: string;
    type?: string;
    content?: string;
    description?: string;
    target_url?: string;
    start_time?: number;
    end_time?: number;
    status?: number;
  };

  type UpdateAnnouncementEnableRequest = {
    id: number;
    enable: boolean;
  };

  type UpdateAnnouncementRequest = {
    id: number;
    title?: string;
    content?: string;
    show?: boolean;
    pinned?: boolean;
    popup?: boolean;
  };

  type UpdateAuthMethodConfigRequest = {
    id: number;
    method: string;
    config?: JsonValue;
    enabled?: boolean;
  };

  type UpdateCouponRequest = {
    id: number;
    name: string;
    code?: string;
    count?: number;
    type: number;
    discount: number;
    start_time: number;
    expire_time: number;
    user_limit?: number;
    subscribe?: number[];
    used_count?: number;
    enable?: boolean;
  };

  type UpdateDocumentRequest = {
    id: number;
    title: string;
    content: string;
    tags?: string[];
    show: boolean;
  };

  type UpdateMembershipPlanRequest = {
    id: number;
    name: string;
    description?: string;
    unit_price: number;
    duration_unit: string;
    duration_value: number;
    enabled: boolean;
    policy: MembershipPolicy;
  };

  type UpdateNodeRequest = {
    id: number;
    name: string;
    tags?: string[];
    port: number;
    address: string;
    server_id: number;
    inbound_alias: string;
    protocol?: string;
    enabled: boolean;
  };

  type UpdateOrderStatusRequest = {
    id: number;
    status: number;
    payment_id?: number;
    trade_no?: string;
  };

  type UpdatePaymentMethodRequest = {
    id: number;
    name?: string;
    platform?: string;
    description?: string;
    icon?: string;
    domain?: string;
    config?: JsonValue;
    fee_mode?: number;
    fee_percent?: number;
    fee_amount?: number;
    enable?: boolean;
  };

  type UpdateServerRequest = {
    id: number;
    name: string;
    country?: string;
    city?: string;
    address: string;
    sort?: number;
  };

  type UpdateSubscribeApplicationRequest = {
    id: number;
    name?: string;
    description?: string;
    icon?: string;
    scheme?: string;
    user_agent?: string;
    enabled?: boolean;
    is_default?: boolean;
    template?: string;
    output_format?: string;
    download_link?: DownloadLink;
  };

  type UpdateSubscribeGroupRequest = {
    id: number;
    name: string;
    description: string;
  };

  type UpdateSubscribeRequest = {
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
    display_template?: "standard" | "popular" | "premium" | "compact";
    badge?: string;
    min_quantity?: number;
    max_quantity?: number;
    sale_starts_at?: number;
    sale_ends_at?: number;
  };

  type UpdateTicketStatusRequest = {
    id: number;
    status: number;
  };

  type UpdateUserAuthMethodRequest = {
    user_id: number;
    auth_type: string;
    auth_identifier: string;
  };

  type UpdateUserBasiceInfoRequest = {
    user_id: number;
    password: string;
    avatar: string;
    balance: number;
    commission: number;
    referral_percentage: number;
    only_first_purchase: boolean;
    gift_amount: number;
    telegram: number;
    refer_code: string;
    referer_id: number;
    enable: boolean;
    is_admin: boolean;
  };

  type UpdateUserMembershipRequest = {
    user_id: number;
    is_member: boolean;
    member_expired_at: number;
  };

  type UpdateUserNotifySettingRequest = {
    user_id: number;
    enable_balance_notify: boolean;
    enable_login_notify: boolean;
    enable_subscribe_notify: boolean;
    enable_trade_notify: boolean;
  };

  type UpdateUserSubscribeRequest = {
    user_subscribe_id: number;
    subscribe_id: number;
    traffic: number;
    expired_at: number;
    upload: number;
    download: number;
  };

  type UpdateXrayTemplateRequest = {
    id: number;
    name?: string;
    type?: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    config?: JsonObject;
    description?: string;
    enabled?: boolean;
    config_template?: string;
    variables_schema?: JsonObject;
    default_variables?: JsonObject;
    subscription_meta?: JsonObject;
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

  type UserListItem = {
    id: number;
    balance: number;
    commission: number;
    gift_amount: number;
    member_expired_at: number;
    is_member: boolean;
    refer_code: string;
    referer_id: number;
    enable: boolean;
    primary_auth_method: UserAuthMethod;
    created_at: number;
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

  type UserStatistics = {
    date?: string;
    register: number;
    new_order_users: number;
    renewal_order_users: number;
    membership_users: number;
  };

  type UserStatisticsResponse = {
    period: string;
    granularity: string;
    series: UserStatistics[];
    today: UserStatistics;
    monthly: UserStatistics;
    all: UserStatistics;
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

  type UserSubscribeProductSummary = {
    id: number;
    name: string;
    speed_limit: number;
    device_limit: number;
  };

  type UserSubscribeListItem = {
    id: number;
    user_id: number;
    subscribe_id: number;
    subscribe: UserSubscribeProductSummary;
    product_name: string;
    start_time: number;
    expire_time: number;
    reset_time: number;
    traffic: number;
    download: number;
    upload: number;
    used_traffic: number;
    remaining_traffic: number;
    status: number;
    effective_status: number;
    status_reason: string;
    created_at: number;
    updated_at: number;
  };

  type UserSubscribeAccess = {
    token: string;
    short: string;
  };

  type UserSubscribeDetail = {
    id: number;
    user_id: number;
    user: User;
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
    used_traffic: number;
    remaining_traffic: number;
    token: string;
    status: number;
    effective_status: number;
    status_reason: string;
    order_no: string;
    note: string;
    presets: UserSubscribePreset[];
    created_at: number;
    updated_at: number;
  };

  type UserSubscribePresetItem = {
    node_id: number;
    inbound_alias: string;
    display_name?: string;
  };

  type UserSubscribePreset = {
    id: number;
    user_subscribe_id: number;
    name: string;
    preset_key: string;
    mode: string;
    profile_key: string;
    items: UserSubscribePresetItem[];
    enabled: boolean;
    is_default: boolean;
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

  type UserSubscribeTrafficLog = {
    /** Subscribe ID */
    subscribe_id: number;
    /** User ID */
    user_id: number;
    /** Upload traffic in bytes */
    upload: number;
    /** Download traffic in bytes */
    download: number;
    /** Total traffic in bytes (Upload + Download) */
    total: number;
    /** Date in YYYY-MM-DD format */
    date: string;
    /** Whether to show detailed traffic */
    details: boolean;
  };

  type UserTrafficData = {
    sid: number;
    upload: number;
    download: number;
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

  type VersionResponse = {
    version: string;
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

  type GetXrayTemplateDetailParams = {
    id: number;
  };

  type XrayTemplate = {
    id: number;
    name: string;
    type: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    config?: JsonObject;
    config_template?: string;
    variables_schema?: JsonObject;
    default_variables?: JsonObject;
    subscription_meta?: JsonObject;
    description: string;
    enabled: boolean;
    created_at: number;
    updated_at: number;
  };

  type XrayTemplateListItem = {
    id: number;
    name: string;
    type: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    description: string;
    enabled: boolean;
    binding_count: number;
    subscription_binding_count: number;
    updated_at: number;
  };

  type XrayTemplateCatalogItem = {
    id: number;
    name: string;
    type: "inbound" | "outbound" | "dns" | "routing" | "geodata";
    config: JsonObject;
    config_template?: string;
    variables_schema: JsonObject;
    default_variables: JsonObject;
    subscription_meta: JsonObject;
    description: string;
    enabled: boolean;
  };

  type ForwarderRuleStatus = {
    rule_id: string;
    listen_ip?: string;
    listen_port: number;
    target_host: string;
    target_port: number;
    node_id?: number;
    node_name?: string;
    server_id?: number;
    server_name?: string;
    inbound?: string;
    status: string;
    error?: string;
    active_connections: number;
    accepted_connections: number;
    failed_connections: number;
    blocked_connections: number;
    upload_bps: number;
    download_bps: number;
    upload_bytes: number;
    download_bytes: number;
  };

  type ForwarderStatus = {
    type?: string;
    instance_id: string;
    boot_id: string;
    online: boolean;
    version?: string;
    uptime: number;
    active_connections: number;
    accepted_connections: number;
    failed_connections: number;
    blocked_connections: number;
    upload_bps: number;
    download_bps: number;
    upload_bytes: number;
    download_bytes: number;
    desired_version?: string;
    desired_hash?: string;
    applied_version?: string;
    applied_hash?: string;
    apply_status: string;
    apply_error?: string;
    last_seen: number;
    rules: ForwarderRuleStatus[];
  };

  type ForwarderInstance = {
    id: number;
    instance_id: string;
    boot_id: string;
    name: string;
    hostname: string;
    public_ip: string;
    version: string;
    started_at: number;
    first_seen_at: number;
    last_seen_at: number;
    desired_version: string;
    desired_hash: string;
    applied_version: string;
    applied_hash: string;
    apply_status: string;
    apply_error: string;
    status: ForwarderStatus;
  };

  type ForwarderListParams = { search?: string };
  type ForwarderListResponse = {
    list: ForwarderInstance[];
    total: number;
  };

  type ForwarderConfigParams = { instance_id: string };
  type ForwarderRule = {
    rule_id: string;
    listen_ip?: string;
    listen_port: number;
    target_host: string;
    target_port: number;
    node_id?: number;
    node_name?: string;
    server_id?: number;
    server_name?: string;
    inbound?: string;
  };
  type ForwarderConfig = {
    instance_id: string;
    public_ip: string;
    version: string;
    hash: string;
    generated_at: number;
    rules: ForwarderRule[];
    warnings: string[];
  };

  type ForwarderConnectionParams = {
    instance_id?: string;
    connection_id?: string;
    start_time?: number;
    end_time?: number;
    listen_port?: number;
    source_ip?: string;
    destination?: string;
    event_type?: string;
    size?: number;
    cursor?: string;
  };
  type ForwarderConnectionEvent = {
    event_id: string;
    connection_id: string;
    instance_id: string;
    boot_id: string;
    event_time: number;
    event_type: string;
    source_ip: string;
    source_port: number;
    listen_ip: string;
    listen_port: number;
    configured_target_host: string;
    configured_target_port: number;
    resolved_target_ip: string;
    resolved_target_port: number;
    forwarder_outbound_ip: string;
    forwarder_outbound_port: number;
    rule_id: string;
    server_id: number;
    node_id: number;
    inbound: string;
    config_version: string;
    connected_at: number;
    closed_at: number;
    duration_ms: number;
    upload_bytes: number;
    download_bytes: number;
    close_initiator: string;
    close_reason: string;
    error: string;
  };
  type ForwarderConnectionPage = {
    list: ForwarderConnectionEvent[];
    next_cursor: string;
    has_more: boolean;
  };
}
