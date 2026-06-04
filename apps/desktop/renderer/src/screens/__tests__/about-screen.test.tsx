// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import React from "react";

vi.mock("../../features/app-about/use-check-app-update.js", () => ({
  useCheckAppUpdate: vi.fn(),
}));

import { AboutScreen } from "../about-screen.js";
import { useCheckAppUpdate } from "../../features/app-about/use-check-app-update.js";

const mockUseCheckAppUpdate = useCheckAppUpdate as ReturnType<typeof vi.fn>;

const idleState = {
  isPending: false,
  status: "idle" as const,
  currentVersion: "0.1.0",
  latestVersion: null,
  updateAvailable: false,
  releaseUrl: null,
  errorCode: null,
  check: vi.fn(),
};

beforeEach(() => vi.clearAllMocks());
afterEach(() => cleanup());

describe("AboutScreen", () => {
  it("renders app name and version", () => {
    mockUseCheckAppUpdate.mockReturnValue(idleState);
    render(<AboutScreen />);
    expect(screen.getByText("Skillbox")).toBeTruthy();
  });

  it("shows author links", () => {
    mockUseCheckAppUpdate.mockReturnValue(idleState);
    render(<AboutScreen />);
    expect(screen.getByText("thien.tranhung@gmail.com")).toBeTruthy();
    expect(screen.getByText("github.com/thientranhung/astraler-skillbox")).toBeTruthy();
    expect(screen.getByText("blog.thisistool.com")).toBeTruthy();
  });

  it("calls check when button is clicked", () => {
    const checkFn = vi.fn();
    mockUseCheckAppUpdate.mockReturnValue({ ...idleState, check: checkFn });
    render(<AboutScreen />);
    fireEvent.click(screen.getByText("Check for Updates"));
    expect(checkFn).toHaveBeenCalledOnce();
  });

  it("shows up-to-date message", () => {
    mockUseCheckAppUpdate.mockReturnValue({
      ...idleState,
      status: "up-to-date",
    });
    render(<AboutScreen />);
    expect(screen.getByText(/You're up to date/)).toBeTruthy();
  });

  it("shows update available with version and link", () => {
    mockUseCheckAppUpdate.mockReturnValue({
      ...idleState,
      status: "available",
      updateAvailable: true,
      latestVersion: "1.2.3",
      releaseUrl: "https://github.com/thientranhung/astraler-skillbox/releases/tag/v1.2.3",
    });
    render(<AboutScreen />);
    expect(screen.getByText(/New version available/)).toBeTruthy();
    expect(screen.getByText("(1.2.3)")).toBeTruthy();
    expect(screen.getByText("View release")).toBeTruthy();
  });

  it("disables button while checking", () => {
    mockUseCheckAppUpdate.mockReturnValue({
      ...idleState,
      isPending: true,
      status: "checking",
    });
    render(<AboutScreen />);
    const btn = screen.getByText("Check for Updates").closest("button");
    expect(btn?.disabled).toBe(true);
    expect(screen.getByText("Checking…")).toBeTruthy();
  });

  it("shows network error message for network_error code", () => {
    mockUseCheckAppUpdate.mockReturnValue({
      ...idleState,
      status: "error",
      errorCode: "network_error",
    });
    render(<AboutScreen />);
    expect(screen.getByText(/No internet connection/)).toBeTruthy();
  });

  it("shows generic error message for unknown error code", () => {
    mockUseCheckAppUpdate.mockReturnValue({
      ...idleState,
      status: "error",
      errorCode: null,
    });
    render(<AboutScreen />);
    expect(screen.getByText("Could not check for updates")).toBeTruthy();
  });

  it("shows 'App Updates' section header to distinguish from plugin updates", () => {
    mockUseCheckAppUpdate.mockReturnValue(idleState);
    render(<AboutScreen />);
    expect(screen.getByText("App Updates")).toBeTruthy();
  });
});
