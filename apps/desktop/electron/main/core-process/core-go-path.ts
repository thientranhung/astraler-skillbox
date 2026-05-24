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
