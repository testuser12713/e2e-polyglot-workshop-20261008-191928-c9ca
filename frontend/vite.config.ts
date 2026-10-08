import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./vitest.setup.ts"],
    css: true,
    // Only the product's own unit tests. The office's Playwright smoke file
    // lives in e2e/ and is run by its own harness, not by vitest.
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
