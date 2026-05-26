// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: {
    getSkill: vi.fn(),
  },
}));

import { useSkillDetail } from "../use-skill-detail.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockGetSkill = methods.getSkill as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useSkillDetail", () => {
  it("does not fetch when skillId is null", async () => {
    const { result } = renderHook(() => useSkillDetail(null), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isPending).toBe(true));
    expect(mockGetSkill).not.toHaveBeenCalled();
  });

  it("fetches skill.get with the given skillId", async () => {
    const fakeResponse = {
      skill: {
        id: 10,
        name: "my-skill",
        relativePath: ".agents/skills/my-skill",
        absolutePath: "/host/.agents/skills/my-skill",
        status: "available" as const,
        sourceLabel: null,
        hostPath: "/host",
        lastScannedAt: null,
      },
      projects: [],
    };
    mockGetSkill.mockResolvedValue(fakeResponse);

    const { result } = renderHook(() => useSkillDetail(10), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetSkill).toHaveBeenCalledWith({ skillId: 10 });
    expect(result.current.data).toEqual(fakeResponse);
  });
});
