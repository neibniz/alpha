import { describe, expect, it } from "vitest";
import { createSubscribeClientFormValues } from "./client-form-data";

describe("createSubscribeClientFormValues", () => {
  it("normalizes omitted optional API fields into controlled form values", () => {
    const values = createSubscribeClientFormValues({
      id: 1,
      name: "Shadowrocket",
      user_agent: "Shadowrocket",
      enabled: true,
      is_default: false,
      output_format: "base64",
      download_link: {},
      created_at: 1,
      updated_at: 1,
    });

    expect(values).toEqual({
      name: "Shadowrocket",
      description: "",
      icon: "",
      user_agent: "Shadowrocket",
      enabled: true,
      scheme: "",
      template: "",
      output_format: "base64",
      download_link: {
        windows: "",
        mac: "",
        linux: "",
        ios: "",
        android: "",
        harmony: "",
      },
    });
  });

  it("returns complete defaults for a new client", () => {
    expect(createSubscribeClientFormValues()).toMatchObject({
      name: "",
      icon: "",
      enabled: true,
      output_format: "",
    });
  });
});
