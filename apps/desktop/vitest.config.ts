import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    include: ["electron/**/__tests__/**/*.test.ts", "renderer/**/__tests__/**/*.test.ts"],
  },
});
