import "@astryxdesign/core/reset.css";
import "@astryxdesign/core/astryx.css";
import { LinkProvider } from "@astryxdesign/core/Link";
import { Theme } from "@astryxdesign/core/theme";
import { TanStackDevtools } from "@tanstack/react-devtools";
import { Outlet, createRootRouteWithContext } from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";

import { defaultThemeSlug, themes } from "#/astryx-themes";
import { RouterLink } from "#/components/router-link";
import type { RouterContext } from "#/router";

const RootComponent = () => (
  <Theme theme={themes[defaultThemeSlug]}>
    <LinkProvider component={RouterLink}>
      <Outlet />
    </LinkProvider>
    <TanStackDevtools
      config={{
        position: "bottom-right",
      }}
      plugins={[
        {
          name: "TanStack Router",
          render: <TanStackRouterDevtoolsPanel />,
        },
      ]}
    />
  </Theme>
);

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootComponent,
});
