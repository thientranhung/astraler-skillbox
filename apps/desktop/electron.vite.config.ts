import path from "path";
import { createRequire } from "module";
import { defineConfig } from "electron-vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const require = createRequire(import.meta.url);
const pkg = require("./package.json") as { version: string };

export default defineConfig({
  main: {
    build: {
      rollupOptions: {
        external: ["electron"],
        input: {
          index: "electron/main/index.ts",
        },
      },
    },
  },
  preload: {
    build: {
      rollupOptions: {
        external: ["electron"],
        input: {
          index: "electron/preload/index.ts",
        },
        output: {
          format: "cjs",
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
    define: {
      "import.meta.env.VITE_APP_VERSION": JSON.stringify(pkg.version),
    },
  },
});
