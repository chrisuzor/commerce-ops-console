import { render, screen } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import { describe, expect, it } from "vitest";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders a readable status label", () => {
    render(<StatusBadge status="NEEDS_ATTENTION" />);
    expect(screen.getByText("NEEDS ATTENTION")).toBeInTheDocument();
  });
});
