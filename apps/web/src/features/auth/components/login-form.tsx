import { Banner } from "@astryxdesign/core/Banner";
import { Button } from "@astryxdesign/core/Button";
import { VStack } from "@astryxdesign/core/Layout";
import { TextInput } from "@astryxdesign/core/TextInput";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";

import { loginMutation } from "#/lib/api/@tanstack/react-query.gen";
import type { CurrentUserEnvelope } from "#/lib/api/types.gen";
import { isProblem } from "#/lib/problem";

interface FieldErrors {
  username?: string;
  password?: string;
}

const errorStatus = (message: string | undefined) =>
  message === undefined ? undefined : { type: "error" as const, message };

export const LoginForm = ({
  onLoggedIn,
}: {
  onLoggedIn: (envelope: CurrentUserEnvelope) => void;
}) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const login = useMutation({
    ...loginMutation(),
    onSuccess: (envelope) => {
      onLoggedIn(envelope);
    },
    onError: (error) => {
      if (isProblem(error) && error.status === 401) {
        setPassword("");
      }
    },
  });

  const submitCredentials = () => {
    const nextErrors: FieldErrors = {};

    if (username === "") {
      nextErrors.username = "Enter your username.";
    }

    if (password === "") {
      nextErrors.password = "Enter your password.";
    }

    setFieldErrors(nextErrors);

    if (Object.keys(nextErrors).length > 0) {
      return;
    }

    login.mutate({ body: { username, password } });
  };

  const problem = isProblem(login.error) ? login.error : null;

  return (
    <VStack
      as="form"
      gap={4}
      hAlign="stretch"
      onSubmit={(event) => {
        event.preventDefault();
        submitCredentials();
      }}
    >
      <TextInput
        autoComplete="username"
        hasAutoCorrect={false}
        autoCapitalize="none"
        label="Username"
        onChange={setUsername}
        size="lg"
        status={errorStatus(fieldErrors.username)}
        value={username}
      />

      <TextInput
        autoComplete="current-password"
        label="Password"
        onChange={setPassword}
        size="lg"
        status={errorStatus(fieldErrors.password)}
        type="password"
        value={password}
      />

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
          endContent={
            <Button label="Try again" onClick={submitCredentials} size="sm" />
          }
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
