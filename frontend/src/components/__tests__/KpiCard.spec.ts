import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import KpiCard from "../KpiCard.vue";

describe("KpiCard", () => {
  it("renders title and value", () => {
    const wrapper = mount(KpiCard, {
      props: {
        title: "Revenue",
        value: "1000"
      }
    });

    expect(wrapper.text()).toContain("Revenue");
    expect(wrapper.text()).toContain("1000");
  });
});
