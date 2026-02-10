import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "jsdom",
    include: ["src/components/ElectricityPrice.test.tsx"],
    clearMocks: true,
    restoreMocks: true,
    setupFiles: [],
  },
});
