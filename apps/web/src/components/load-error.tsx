import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";

import { isProblem } from "#/lib/problem";

// The inline error of a failed read (5xx or network). It names the request when the API answered, and offers a retry.
export const LoadError = ({
  error,
  onRetry,
}: {
  error: unknown;
  onRetry: () => void;
}) => (
  <Banner
    description={
      isProblem(error) ? `Request ID: ${error.requestId}` : undefined
    }
    endContent={<Button label="Try again" onClick={onRetry} size="sm" />}
    status="error"
    title="Something went wrong."
  />
);
