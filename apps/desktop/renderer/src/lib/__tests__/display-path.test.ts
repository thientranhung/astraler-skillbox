import { describe, expect, it } from "vitest";
import { displayPath } from "../display-path.js";

describe("displayPath", () => {
  it("trims trailing slashes from display paths", () => {
    expect(displayPath("/Users/test/.agents/skills/")).toBe("/Users/test/.agents/skills");
    expect(displayPath("/Users/test/.agents/skills//")).toBe("/Users/test/.agents/skills");
  });

  it("preserves root and empty/nullish inputs", () => {
    expect(displayPath("/")).toBe("/");
    expect(displayPath("")).toBe("");
    expect(displayPath(null)).toBe("");
    expect(displayPath(undefined)).toBe("");
  });
});
