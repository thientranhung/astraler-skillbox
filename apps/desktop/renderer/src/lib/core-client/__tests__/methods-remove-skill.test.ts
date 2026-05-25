import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("../client.js", () => ({
  invoke: vi.fn().mockResolvedValue({ operationId: 51 }),
}));

import { invoke } from "../client.js";
import { methods } from "../methods.js";

const mockInvoke = invoke as ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.clearAllMocks();
});

describe("methods.removeSkill", () => {
  it("invokes remove.skill with projectId and installId", async () => {
    const res = await methods.removeSkill({ projectId: 12, installId: 88 });
    expect(mockInvoke).toHaveBeenCalledWith("remove.skill", { projectId: 12, installId: 88 });
    expect(res.operationId).toBe(51);
  });
});
