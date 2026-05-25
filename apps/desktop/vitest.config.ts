import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
  resolve: {
    alias: {
      "@contracts": path.resolve(__dirname, "../../shared/generated"),
    },
  },
  test: {
    environment: "node",
    include: [
      "electron/**/__tests__/**/*.test.ts",
      "renderer/**/__tests__/**/*.test.ts",
      "renderer/**/__tests__/**/*.test.tsx",
    ],
    environmentMatchGlobs: [
      ["renderer/**", "happy-dom"],
    ],
  },
});
