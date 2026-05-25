import { execFileSync } from "node:child_process";
import { mkdirSync, chmodSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const desktop = path.resolve(here, "..");          // apps/desktop
const repoRoot = path.resolve(desktop, "../..");   // repo root
const coreGo = path.join(repoRoot, "core-go");
const outDir = path.join(desktop, "resources", "core");
const outBin = path.join(outDir, "skillbox-core");

mkdirSync(outDir, { recursive: true });

// Pure-Go SQLite (modernc.org/sqlite) => CGO_ENABLED=0, no toolchain at runtime.
execFileSync("go", ["build", "-o", outBin, "./cmd/skillbox-core"], {
  cwd: coreGo,
  stdio: "inherit",
  env: { ...process.env, GOOS: "darwin", GOARCH: "arm64", CGO_ENABLED: "0" },
});

chmodSync(outBin, 0o755);
console.log(`[build:core] built ${outBin}`);
