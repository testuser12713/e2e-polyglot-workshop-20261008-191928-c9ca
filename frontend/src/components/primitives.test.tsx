import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import Button from "./Button";
import FormField from "./FormField";
import StatusBadge from "./StatusBadge";

describe("Button", () => {
  it("does not fire when disabled", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(
      <Button disabled onClick={onClick}>
        Speichern
      </Button>,
    );

    await user.click(screen.getByRole("button", { name: "Speichern" }));
    expect(onClick).not.toHaveBeenCalled();
  });

  it("is disabled and busy while loading", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(
      <Button loading onClick={onClick}>
        Speichern
      </Button>,
    );

    const button = screen.getByRole("button", { name: "Speichern" });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
    await user.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });
});

describe("FormField", () => {
  it("stays neutral on an untouched form even with an error (AC-24)", () => {
    render(
      <FormField label="E-Mail" htmlFor="email" error="Pflichtfeld">
        <input id="email" />
      </FormField>,
    );

    expect(screen.queryByText("Pflichtfeld")).not.toBeInTheDocument();
  });

  it("shows the error after the field was blurred", async () => {
    const user = userEvent.setup();
    render(
      <FormField label="E-Mail" htmlFor="email" error="Pflichtfeld">
        <input id="email" />
      </FormField>,
    );

    await user.click(screen.getByLabelText("E-Mail"));
    await user.tab();

    expect(screen.getByText("Pflichtfeld")).toBeInTheDocument();
  });

  it("shows the error immediately after a submit attempt", () => {
    render(
      <FormField label="E-Mail" htmlFor="email" error="Pflichtfeld" submitted>
        <input id="email" />
      </FormField>,
    );

    expect(screen.getByText("Pflichtfeld")).toBeInTheDocument();
  });
});

describe("StatusBadge", () => {
  it("renders the fixed German label, never the raw key", () => {
    render(<StatusBadge status="in_progress" />);
    expect(screen.getByText("in Arbeit")).toBeInTheDocument();
  });
});
