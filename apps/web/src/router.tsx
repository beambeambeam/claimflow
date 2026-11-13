import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter as createTanStackRouter } from "@tanstack/react-router";

import { isProblem } from "#/lib/problem";

import { routeTree } from "./routeTree.gen";

export interface RouterContext {
  queryClient: QueryClient;
}

export const getRouter = () => {
  const queryClient = new QueryClient();

  const router = createTanStackRouter({
    routeTree,
    context: { queryClient },
    scrollRestoration: true,
    defaultPreload: "intent",
    // TanStack Query caches loader data, so the router does not cache it again.
    defaultPreloadStaleTime: 0,
    Wrap: ({ children }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    ),
  });

  // A 401 from any query means the session is gone: drop every cached answer and go to login (F01-AC-21).
  // A failed login is a mutation, so it does not come through here.
  queryClient.getQueryCache().subscribe((event) => {
    if (
      event.type !== "updated" ||
      event.action.type !== "error" ||
      !isProblem(event.action.error) ||
      event.action.error.status !== 401
    ) {
      return;
    }

    queryClient.clear();

    const { href, pathname } = router.latestLocation;

    if (pathname !== "/login") {
      void router.navigate({ to: "/login", search: { redirect: href } });
    }
  });

  return router;
};

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>;
  }
}
