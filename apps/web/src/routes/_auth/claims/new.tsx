import {
  createFileRoute,
  useRouteContext,
  useRouter,
} from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { NewClaimPage } from "#/features/claims/components/new-claim-page";

const NewClaimRoute = () => {
  const { user } = useRouteContext({ from: "/_auth" });
  const router = useRouter();
  const [visit, setVisit] = useState(0);

  // The New Claim link clicked on this page reloads the same URL instead of navigating, so the page would keep the
  // confirmation of the last submit. A new key on each reload of this URL starts an empty form.
  useEffect(
    () =>
      router.subscribe("onBeforeNavigate", ({ hrefChanged, toLocation }) => {
        if (!hrefChanged && toLocation.pathname === "/claims/new") {
          setVisit((count) => count + 1);
        }
      }),
    [router]
  );

  return <NewClaimPage key={visit} user={user} />;
};

export const Route = createFileRoute("/_auth/claims/new")({
  component: NewClaimRoute,
});
