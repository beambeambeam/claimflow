import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";

import { endLocalSession } from "#/features/auth/session";
import { logoutMutation } from "#/lib/api/@tanstack/react-query.gen";

// The session ends in this browser only when the server confirms it (204). A 401 means it already ended,
// and the MutationCache handler in `router.tsx` deals with it. Any other error leaves the session alone.
export const useLogout = () => {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    ...logoutMutation(),
    onSuccess: () => {
      endLocalSession(queryClient, router, { keepLocation: false });
    },
  });
};
