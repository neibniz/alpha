export type SubscribeClientFormValues = {
  name: string;
  description: string;
  icon: string;
  user_agent: string;
  enabled: boolean;
  scheme: string;
  template: string;
  output_format: string;
  download_link: {
    windows: string;
    mac: string;
    linux: string;
    ios: string;
    android: string;
    harmony: string;
  };
};

export function createSubscribeClientFormValues(
  client?: Partial<API.SubscribeApplication>
): SubscribeClientFormValues {
  return {
    name: client?.name ?? "",
    description: client?.description ?? "",
    icon: client?.icon ?? "",
    user_agent: client?.user_agent ?? "",
    enabled: client?.enabled ?? true,
    scheme: client?.scheme ?? "",
    template: client?.template ?? "",
    output_format: client?.output_format ?? "",
    download_link: {
      windows: client?.download_link?.windows ?? "",
      mac: client?.download_link?.mac ?? "",
      linux: client?.download_link?.linux ?? "",
      ios: client?.download_link?.ios ?? "",
      android: client?.download_link?.android ?? "",
      harmony: client?.download_link?.harmony ?? "",
    },
  };
}
