import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// A default, harmless fetch so that components which probe the API on mount
// (the shell's reachability indicator) do not crash a test that does not care
// about the network. Individual tests override it with their own stub.
vi.stubGlobal(
  "fetch",
  vi.fn(() =>
    Promise.resolve({
      ok: true,
      status: 200,
      text: () => Promise.resolve(JSON.stringify({ status: "ok" })),
    }),
  ),
);

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
