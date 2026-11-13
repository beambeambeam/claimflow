import { AppShell } from "@astryxdesign/core/AppShell";
import {
  createFileRoute,
  Outlet,
  redirect,
  useRouteContext,
} from "@tanstack/react-router";

import { AppHeader } from "#/features/auth/components/app-header";
import { isUnauthorized } from "#/features/auth/session";
import { getMeOptions } from "#/lib/api/@tanstack/react-query.gen";

const AuthLayout = () => {
  const { user } = useRouteContext({ from: "/_auth" });

  return (
    <AppShell contentPadding={6} topNav={<AppHeader user={user} />}>
      <Outlet />
    </AppShell>
  );
};

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
      if (!isUnauthorized(error)) {
        throw error;
      }

      // TanStack Router needs the redirect Response thrown; it is not an Error.
      // oxlint-disable-next-line typescript/only-throw-error
      throw redirect({ to: "/login", search: { redirect: location.href } });
    }
  },
  component: AuthLayout,
});
