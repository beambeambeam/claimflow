import type { QueryClient } from "@tanstack/react-query";
import type { AnyRouter } from "@tanstack/react-router";

import type { Problem } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

// A 401 means the server no longer knows this session (F01-AC-21).
export const isUnauthorized = (failure: unknown): failure is Problem =>
  isProblem(failure) && failure.status === 401;

// Ends the session in this browser: drops every cached answer and opens the login page.
// With `keepLocation`, the login page gets `?redirect=<current location>` so the user returns after login.
// On the login page itself the location stays, so a failed login keeps its inline error.
export const endLocalSession = (
  queryClient: QueryClient,
  router: AnyRouter,
  { keepLocation }: { keepLocation: boolean }
) => {
  queryClient.clear();

  const { href, pathname } = router.latestLocation;

  if (pathname === "/login") {
    return;
  }

  void router.navigate({
    to: "/login",
    search: keepLocation ? { redirect: href } : {},
  });
};
