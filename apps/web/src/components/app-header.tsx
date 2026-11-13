import { Link } from "@tanstack/react-router";

import type { CurrentUser } from "#/lib/api/types.gen";

type Role = CurrentUser["role"];

// Role labels from F-01 §2.2.
const roleLabels: Record<Role, string> = {
  requester: "Requester",
  reviewer: "Reviewer",
  finance: "Finance Officer",
};

// Links are gated by permission, not role name (F-01 §1.6).
const can = (user: CurrentUser, resource: string, action: string) =>
  user.permissions[resource]?.includes(action);

export const AppHeader = ({ user }: { user: CurrentUser }) => (
  <header className="flex items-center justify-between gap-4 border-b p-4">
    <nav className="flex gap-4">
      {can(user, "claim", "list") ? <Link to="/claims">My Claims</Link> : null}
      {can(user, "claim", "create") ? (
        // No /claims/new route until F-02, so a plain anchor avoids a broken router link.
        <a href="/claims/new">New Claim</a>
      ) : null}
      {can(user, "review", "list") ? (
        <Link to="/reviews">Review Queue</Link>
      ) : null}
    </nav>
    <div className="flex items-center gap-4">
      <div className="text-right">
        <p className="font-medium">{user.name}</p>
        <p className="text-sm text-gray-600">{roleLabels[user.role]}</p>
      </div>
      <button type="button">Log out</button>
    </div>
  </header>
);
