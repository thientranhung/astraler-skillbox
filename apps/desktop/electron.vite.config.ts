import { defineConfig } from "electron-vite";
import react from "@vitejs/plugin-react";

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
    build: {
      rollupOptions: {
        input: {
          index: "renderer/index.html",
        },
      },
    },
    plugins: [react()],
  },
});
