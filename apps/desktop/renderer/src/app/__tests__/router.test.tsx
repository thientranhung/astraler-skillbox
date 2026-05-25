// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, waitFor } from "@testing-library/react";
import React from "react";

vi.mock("../../features/app-settings/use-app-settings.js", () => ({
  useAppSettings: vi.fn(),
}));
vi.mock("@tanstack/react-router", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    useNavigate: vi.fn(),
  };
});

import { IndexRedirector } from "../router.js";
import { useAppSettings } from "../../features/app-settings/use-app-settings.js";
import { useNavigate } from "@tanstack/react-router";

const mockUseAppSettings = useAppSettings as ReturnType<typeof vi.fn>;
const mockUseNavigate = useNavigate as ReturnType<typeof vi.fn>;

beforeEach(() => vi.clearAllMocks());

describe("IndexRedirector", () => {
  it("navigates to /dashboard when activeHost is set", async () => {
    const navigate = vi.fn();
    mockUseNavigate.mockReturnValue(navigate);
    mockUseAppSettings.mockReturnValue({
      isPending: false,
      isError: false,
      data: { activeHost: { hostId: 1, path: "/tmp/host" } },
    });
    render(<IndexRedirector />);
    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith({ to: "/dashboard", replace: true })
    );
  });

  it("navigates to /setup when activeHost is null", async () => {
    const navigate = vi.fn();
    mockUseNavigate.mockReturnValue(navigate);
    mockUseAppSettings.mockReturnValue({
      isPending: false,
      isError: false,
      data: { activeHost: null },
    });
    render(<IndexRedirector />);
    await waitFor(() =>
      expect(navigate).toHaveBeenCalledWith({ to: "/setup", replace: true })
    );
  });
});
