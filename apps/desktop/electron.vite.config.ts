import path from "path";
import { defineConfig } from "electron-vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  main: {
    build: {
      rollupOptions: {
        input: {
          index: "electron/main/index.ts",
        },
      },
    },
  },
  preload: {
    build: {
      rollupOptions: {
        input: {
          index: "electron/preload/index.ts",
        },
      },
    },
  },
  renderer: {
    root: "renderer",
    resolve: {
      alias: {
        "@contracts": path.resolve(__dirname, "../../shared/generated"),
      },
    },
    build: {
      rollupOptions: {
        input: {
          index: "renderer/index.html",
        },
      },
    },
    plugins: [tailwindcss(), react()],
  },
});
