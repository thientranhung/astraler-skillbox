// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import React from "react";

vi.mock("../use-update-provider-paths.js", () => ({
  useUpdateProviderPaths: vi.fn(),
}));

import { ProviderPathsEditor } from "../provider-paths-editor.js";
import { useUpdateProviderPaths } from "../use-update-provider-paths.js";

const mockUseUpdateProviderPaths = useUpdateProviderPaths as ReturnType<typeof vi.fn>;

const defaultProps = {
  providerKey: "claude",
  scope: "project" as const,
  purpose: "detect" as const,
  currentPaths: [".claude"],
  onClose: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseUpdateProviderPaths.mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
    error: null,
    isError: false,
  });
});

afterEach(() => cleanup());

describe("ProviderPathsEditor", () => {
  it("renders the dialog with current paths pre-filled", () => {
    render(<ProviderPathsEditor {...defaultProps} />);
    const input = screen.getByRole("textbox") as HTMLInputElement;
    expect(input.value).toContain(".claude");
  });

  it("shows scope and purpose labels", () => {
    render(<ProviderPathsEditor {...defaultProps} />);
    expect(screen.getByText(/project/i)).not.toBeNull();
    expect(screen.getByText(/detect/i)).not.toBeNull();
  });

  it("calls mutate on save", () => {
    const mutate = vi.fn();
    mockUseUpdateProviderPaths.mockReturnValue({ mutate, isPending: false, error: null, isError: false });

    render(<ProviderPathsEditor {...defaultProps} />);
    fireEvent.click(screen.getByRole("button", { name: /save/i }));

    expect(mutate).toHaveBeenCalledWith(
      {
        providerKey: "claude",
        scope: "project",
        purpose: "detect",
        paths: [".claude"],
      },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it("calls onClose on cancel", () => {
    const onClose = vi.fn();
    render(<ProviderPathsEditor {...defaultProps} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(onClose).toHaveBeenCalled();
  });

  it("shows a metadata note that overrides are config only", () => {
    render(<ProviderPathsEditor {...defaultProps} />);
    expect(screen.getByText(/configuration metadata|behavior integration/i)).not.toBeNull();
  });
});
