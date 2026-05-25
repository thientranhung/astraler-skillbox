// @vitest-environment happy-dom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";
import { RemoveSkillDialog } from "../remove-skill-dialog.js";

afterEach(() => cleanup());

const baseProps = {
  skillName: "documentation-writer",
  providerDisplayName: "Shared Agent Skills (.agents)",
  path: "/repo/content-lab/.agents/skills/documentation-writer",
  isPending: false,
};

describe("RemoveSkillDialog", () => {
  it("shows skill, provider, and exact path", () => {
    render(<RemoveSkillDialog {...baseProps} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText("documentation-writer")).toBeTruthy();
    expect(screen.getByText(/Shared Agent Skills/)).toBeTruthy();
    expect(screen.getByText(baseProps.path)).toBeTruthy();
    expect(screen.getByText(/not affected/i)).toBeTruthy();
  });

  it("calls onConfirm when Remove is clicked", () => {
    const onConfirm = vi.fn();
    render(<RemoveSkillDialog {...baseProps} onConfirm={onConfirm} onCancel={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: /^Remove$/ }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when Cancel is clicked", () => {
    const onCancel = vi.fn();
    render(<RemoveSkillDialog {...baseProps} onConfirm={vi.fn()} onCancel={onCancel} />);
    fireEvent.click(screen.getByRole("button", { name: /Cancel/ }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
