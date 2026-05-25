import path from "path";

/**
 * Resolves the core-go directory from an electron-vite main process output dir.
 * electron-vite builds main to <project>/out/main/; from there:
 *   out/main/ -> out/ -> apps/desktop/ -> apps/ -> repo root -> core-go
 * That's 4 levels up from out/main/ plus "core-go".
 */
export function resolveCoreGoPath(baseDir: string): string {
  return path.resolve(baseDir, "../../../../core-go");
}

export interface CoreSpawnSpec {
  command: string;
  args: string[];
  cwd: string;
}

/**
 * Resolves how to spawn the Go sidecar.
 * Dev: `go run ./cmd/skillbox-core` from the repo core-go dir.
 * Packaged: the bundled binary under process.resourcesPath (outside ASAR),
 * so it needs no `go`, repo checkout, or dev PATH.
 */
export function resolveCoreSpawn(opts: {
  isPackaged: boolean;
  baseDir: string;
  resourcesPath: string;
}): CoreSpawnSpec {
  if (opts.isPackaged) {
    const bin = path.join(opts.resourcesPath, "core", "skillbox-core");
    return { command: bin, args: [], cwd: path.dirname(bin) };
  }
  const cwd = resolveCoreGoPath(opts.baseDir);
  return { command: "go", args: ["run", "./cmd/skillbox-core"], cwd };
}
