import i18n from "i18next";
import { initReactI18next } from "react-i18next";

if (!i18n.isInitialized) {
  i18n.use(initReactI18next).init({
    defaultNS: "components",
    fallbackLng: "zh-CN",
    initImmediate: false,
    interpolation: { escapeValue: false },
    lng: "zh-CN",
    ns: ["components"],
    react: { useSuspense: false },
    resources: {
      "zh-CN": {
        components: {
          common: { close: "关闭" },
        },
      },
    },
  });
}
