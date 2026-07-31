import { extractDomain, renderAppSubLink } from "@workspace/ui/utils/index";
import { create } from "zustand";

export interface GlobalStore {
  common: API.GetGlobalConfigResponse;
  user?: API.User;
  setCommon: (common: Partial<API.GetGlobalConfigResponse>) => void;
  setUser: (user?: API.User) => void;
  getUserSubscribe: (short: string, token: string, type?: string) => string[];
  getAppSubLink: (url: string, schema?: string) => string;
}

export const useGlobalStore = create<GlobalStore>((set, get) => ({
  common: {
    site: {
      host: "",
      site_name: "",
      site_desc: "",
      site_logo: "",
      keywords: "",
      custom_html: "",
      custom_data: "",
    },
    verify: {
      turnstile_site_key: "",
      enable_login_verify: false,
      enable_register_verify: false,
      enable_reset_password_verify: false,
    },
    auth: {
      email: {
        enable: false,
        enable_verify: false,
        enable_domain_suffix: false,
        domain_suffix_list: [],
      },
      register: {
        stop_register: false,
        enable_ip_register_limit: false,
        ip_register_limit: 0,
        ip_register_limit_duration: 0,
      },
    },
    invite: {
      forced_invite: false,
      referral_percentage: 0,
      only_first_purchase: false,
    },
    currency: {
      currency_unit: "USD",
      currency_symbol: "$",
    },
    subscribe: {
      single_model: false,
      subscribe_path: "",
      subscribe_domain: "",
      pan_domain: false,
      user_agent_limit: false,
      user_agent_list: [],
    },
    verify_code: {
      verify_code_expire_time: 5,
      verify_code_limit: 15,
      verify_code_interval: 60,
    },
    oauth_methods: [],
    web_ad: false,
  },
  user: undefined,
  setCommon: (common) =>
    set((state) => ({
      common: {
        ...state.common,
        ...common,
      },
    })),
  setUser: (user) => set({ user }),
  getUserSubscribe: (short: string, token: string, type?: string) => {
    const { pan_domain, subscribe_domain, subscribe_path } =
      get().common.subscribe || {};
    const domains = subscribe_domain
      ? subscribe_domain.split("\n")
      : [extractDomain(window.location.origin, pan_domain)];

    return domains.map((domain) => {
      if (pan_domain) {
        if (type)
          return `https://${short}.${type}.${domain}${subscribe_path}?token=${token}&type=${type}`;
        return `https://${short}.${domain}${subscribe_path}?token=${token}`;
      }
      if (type)
        return `https://${domain}${subscribe_path}?token=${token}&type=${type}`;
      return `https://${domain}${subscribe_path}?token=${token}`;
    });
  },
  getAppSubLink: (url: string, schema?: string) => {
    const name = get().common?.site?.site_name || "";
    return renderAppSubLink(schema, { url, name });
  },
}));
