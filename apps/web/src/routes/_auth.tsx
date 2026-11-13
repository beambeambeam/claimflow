import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";

import { getMeOptions } from "#/lib/api/@tanstack/react-query.gen";
import { zProblem } from "#/lib/api/zod.gen";

// Session guard for every route under _auth. Anonymous users go to /login with the requested path.
export const Route = createFileRoute("/_auth")({
  beforeLoad: async ({ context, location }) => {
    try {
      const envelope = await context.queryClient.query({
        ...getMeOptions(),
        retry: false,
        staleTime: "static",
      });

      return { user: envelope.data };
    } catch (error) {
      // Caught values are unknown, so validate with the schema instead of readProblem.
      const problem = zProblem.safeParse(error);

      if (!problem.success || problem.data.status !== 401) {
        throw error;
      }

      // TanStack Router needs the redirect Response thrown; it is not an Error.
      // oxlint-disable-next-line typescript/only-throw-error
      throw redirect({ to: "/login", search: { redirect: location.href } });
    }
  },
  component: Outlet,
});
