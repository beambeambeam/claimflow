import { vi } from "vite-plus/test";

// Third-party dev UI. Its unmount throws under jsdom, which breaks cleanup for every later test.
vi.mock("@tanstack/react-devtools", () => ({
  TanStackDevtools: () => null,
}));

vi.mock("@tanstack/react-router-devtools", () => ({
  TanStackRouterDevtoolsPanel: () => null,
}));
