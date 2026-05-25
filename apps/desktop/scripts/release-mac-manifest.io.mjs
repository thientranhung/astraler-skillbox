import { promises as defaultFs } from "node:fs";
import path from "node:path";

/**
 * Atomically writes content by writing a temporary sibling file then renaming it
 * over the final path. A failed write leaves the previous final path untouched.
 *
 * @param {string} finalPath
 * @param {string} content
 * @param {{ fsImpl?: typeof defaultFs, tempPath?: string }} [opts]
 */
export async function atomicWrite(finalPath, content, opts = {}) {
  const fsImpl = opts.fsImpl ?? defaultFs;
  const dir = path.dirname(finalPath);
  const tmpPath =
    opts.tempPath ??
    path.join(dir, `.tmp-${Date.now()}-${Math.random().toString(36).slice(2)}`);

  try {
    await fsImpl.writeFile(tmpPath, content, "utf8");
    await fsImpl.rename(tmpPath, finalPath);
  } catch (err) {
    try {
      await fsImpl.unlink(tmpPath);
    } catch {
      // ignore cleanup errors
    }
    throw err;
  }
}
