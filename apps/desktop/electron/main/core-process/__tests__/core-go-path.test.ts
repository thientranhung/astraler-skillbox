import { describe, it, expect } from "vitest";
import path from "path";
import { resolveCoreGoPath } from "../core-go-path.js";
import { resolveCoreSpawn } from "../core-go-path.js";

describe("resolveCoreGoPath", () => {
  it("maps electron-vite out/main/ to repo core-go", () => {
    // electron-vite builds main process to <project>/out/main/
    const outMain = "/Users/dev/astraler-skillbox/apps/desktop/out/main";
    const result = resolveCoreGoPath(outMain);
    expect(result).toBe(path.normalize("/Users/dev/astraler-skillbox/core-go"));
  });

  it("resolves correctly on an arbitrary nested path", () => {
    const outMain = "/home/ci/repo/apps/desktop/out/main";
    expect(resolveCoreGoPath(outMain)).toBe(path.normalize("/home/ci/repo/core-go"));
  });
});

describe("resolveCoreSpawn", () => {
  it("uses `go run` from repo core-go in dev mode", () => {
    const spec = resolveCoreSpawn({
      isPackaged: false,
      baseDir: "/Users/dev/astraler-skillbox/apps/desktop/out/main",
      resourcesPath: "/ignored",
    });
    expect(spec.command).toBe("go");
    expect(spec.args).toEqual(["run", "./cmd/skillbox-core"]);
    expect(spec.cwd).toBe(path.normalize("/Users/dev/astraler-skillbox/core-go"));
  });

  it("uses bundled binary from resourcesPath in packaged mode", () => {
    const resources = "/Applications/Astraler Skillbox.app/Contents/Resources";
    const spec = resolveCoreSpawn({
      isPackaged: true,
      baseDir: "/ignored",
      resourcesPath: resources,
    });
    const expected = path.join(resources, "core", "skillbox-core");
    expect(spec.command).toBe(expected);
    expect(spec.args).toEqual([]);
    expect(spec.cwd).toBe(path.dirname(expected));
  });
});
