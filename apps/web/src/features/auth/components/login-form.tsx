import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";
import { VStack } from "@astryxdesign/core/Layout";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";

import { loginMutation } from "#/lib/api/@tanstack/react-query.gen";
import type { CurrentUserEnvelope } from "#/lib/api/types.gen";
import { zLoginRequest } from "#/lib/api/zod.gen";
import { useAppForm } from "#/lib/form";
import { isProblem } from "#/lib/problem";

// Contract limits from the generated schema, with field messages for empty input.
const loginSchema = z.object({
  username: z
    .string()
    .min(1, "Enter your username.")
    .pipe(zLoginRequest.shape.username),
  password: z
    .string()
    .min(1, "Enter your password.")
    .pipe(zLoginRequest.shape.password),
});

export const LoginForm = ({
  onLoggedIn,
}: {
  onLoggedIn: (envelope: CurrentUserEnvelope) => void;
}) => {
  const login = useMutation({
    ...loginMutation(),
    onSuccess: (envelope) => {
      onLoggedIn(envelope);
    },
  });

  const form = useAppForm({
    defaultValues: { username: "", password: "" },
    validators: { onSubmit: loginSchema },
    onSubmit: ({ value }) => {
      login.mutate(
        { body: value },
        {
          onError: (error) => {
            if (isProblem(error) && error.status === 401) {
              form.setFieldValue("password", "");
            }
          },
        }
      );
    },
  });

  const submit = () => {
    void form.handleSubmit();
  };

  const problem = isProblem(login.error) ? login.error : null;

  return (
    <VStack
      as="form"
      gap={4}
      hAlign="stretch"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <form.AppField name="username">
        {(field) => (
          <field.TextField
            autoCapitalize="none"
            autoComplete="username"
            hasAutoCorrect={false}
            label="Username"
            size="lg"
          />
        )}
      </form.AppField>

      <form.AppField name="password">
        {(field) => (
          <field.TextField
            autoComplete="current-password"
            label="Password"
            size="lg"
            type="password"
          />
        )}
      </form.AppField>

      {login.isError && problem?.status === 401 ? (
        <Banner status="error" title="Invalid username or password." />
      ) : null}

      {login.isError && problem?.status !== 401 ? (
        <Banner
          description={
            problem?.requestId === undefined
              ? undefined
              : `Request ID: ${problem.requestId}`
          }
          endContent={<Button label="Try again" onClick={submit} size="sm" />}
          status="error"
          title="Something went wrong."
        />
      ) : null}

      <Button
        isLoading={login.isPending}
        label="Log in"
        size="lg"
        type="submit"
        variant="primary"
      />
    </VStack>
  );
};
