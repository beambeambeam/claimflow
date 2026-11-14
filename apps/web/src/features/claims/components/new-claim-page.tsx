import { can } from "#/features/auth/permissions";
import type { CurrentUser } from "#/lib/api/types.gen";

import { CannotCreateClaim } from "./cannot-create-claim";
import { ClaimForm } from "./claim-form";

// The New Claim screen: the claim form for a member who may create claims, or the 403 page for one who may not (F-02 §2.2).
export const NewClaimPage = ({ user }: { user: CurrentUser }) =>
  can(user, "claim", "create") ? <ClaimForm /> : <CannotCreateClaim />;
