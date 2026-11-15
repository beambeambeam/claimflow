import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter as createTanStackRouter } from "@tanstack/react-router";

import { endLocalSession, isUnauthorized } from "#/features/auth/session";

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
  queryClient.getQueryCache().subscribe((event) => {
    if (
      event.type === "updated" &&
      event.action.type === "error" &&
      isUnauthorized(event.action.error)
    ) {
      endLocalSession(queryClient, router, { keepLocation: true });
    }
  });

  // The same holds for a mutation, except on the login page: a failed login is a 401 that the form shows inline.
  queryClient.getMutationCache().subscribe((event) => {
    if (
      event.type === "updated" &&
      event.action.type === "error" &&
      isUnauthorized(event.action.error) &&
      router.latestLocation.pathname !== "/login"
    ) {
      endLocalSession(queryClient, router, { keepLocation: true });
    }
  });

  return router;
};

declare module "@tanstack/react-router" {
  interface Register {
    router: ReturnType<typeof getRouter>;
  }
}
