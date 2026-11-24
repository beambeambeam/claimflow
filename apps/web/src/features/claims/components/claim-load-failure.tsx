import { Text } from "@astryxdesign/core/Text";

import { LoadError } from "#/components/load-error";
import { isProblem } from "#/lib/problem";

export const ClaimAccessDenied = () => (
  <Text type="body" color="secondary">
    You don&apos;t have access to this claim.
  </Text>
);

export const ClaimLoadFailure = ({
  error,
  onRetry,
}: {
  error: unknown;
  onRetry: () => void;
}) => {
  if (isProblem(error)) {
    if (error.status === 403) {
      return <ClaimAccessDenied />;
    }

    if (error.status === 404) {
      return (
        <Text type="body" color="secondary">
          Claim not found.
        </Text>
      );
    }
  }

  return <LoadError error={error} onRetry={onRetry} />;
};
