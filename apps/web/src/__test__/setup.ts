import { configure } from "@testing-library/react";
import { vi } from "vite-plus/test";

// The default wait of findBy and waitFor is 1 s. The first render of a file can take longer on a slow or busy machine,
// which fails the test before the page shows. A longer wait changes nothing for a test that passes.
configure({ asyncUtilTimeout: 15_000 });

// Third-party dev UI. Its unmount throws under jsdom, which breaks cleanup for every later test.
vi.mock("@tanstack/react-devtools", () => ({
  TanStackDevtools: () => null,
}));

vi.mock("@tanstack/react-router-devtools", () => ({
  TanStackRouterDevtoolsPanel: () => null,
}));

// jsdom has no modal dialog. This stand-in opens and closes the element, which is all the dialog tests read.
HTMLDialogElement.prototype.showModal = function showModal() {
  this.setAttribute("open", "");
};

HTMLDialogElement.prototype.close = function close() {
  this.removeAttribute("open");
  this.dispatchEvent(new Event("close"));
};
