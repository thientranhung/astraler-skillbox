export function displayPath(path: string | null | undefined): string {
  if (path == null) return "";
  if (path === "/") return path;
  return path.replace(/\/+$/, "");
}
