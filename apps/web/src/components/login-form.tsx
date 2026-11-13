import { useMutation } from "@tanstack/react-query";
import { useState } from "react";

import { loginMutation } from "#/lib/api/@tanstack/react-query.gen";
import { readProblem } from "#/lib/problem";

interface FieldErrors {
  username?: string;
  password?: string;
}

const inputClassName =
  "rounded border border-gray-400 px-3 py-2 focus:outline-2 focus:outline-blue-700";

export const LoginForm = ({ onLoggedIn }: { onLoggedIn: () => void }) => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const login = useMutation({
    ...loginMutation(),
    onSuccess: () => {
      onLoggedIn();
    },
    onError: (error) => {
      if (readProblem(error)?.status === 401) {
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

  const problem = readProblem(login.error);

  return (
    <form
      className="flex w-full max-w-sm flex-col gap-4"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        submitCredentials();
      }}
    >
      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium" htmlFor="username">
          Username
        </label>
        <input
          autoComplete="username"
          className={inputClassName}
          id="username"
          name="username"
          onChange={(event) => {
            setUsername(event.target.value);
          }}
          type="text"
          value={username}
        />
        {fieldErrors.username === undefined ? null : (
          <p className="text-sm text-red-700">{fieldErrors.username}</p>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-sm font-medium" htmlFor="password">
          Password
        </label>
        <input
          autoComplete="current-password"
          className={inputClassName}
          id="password"
          name="password"
          onChange={(event) => {
            setPassword(event.target.value);
          }}
          type="password"
          value={password}
        />
        {fieldErrors.password === undefined ? null : (
          <p className="text-sm text-red-700">{fieldErrors.password}</p>
        )}
      </div>

      {login.isError && problem?.status === 401 ? (
        <p className="text-sm text-red-700" role="alert">
          Invalid username or password.
        </p>
      ) : null}

      {login.isError && problem?.status !== 401 ? (
        <div className="flex flex-col gap-2 text-sm text-red-700" role="alert">
          <p>Something went wrong.</p>
          {problem?.requestId === undefined ? null : (
            <p className="text-xs text-gray-600">
              Request ID: {problem.requestId}
            </p>
          )}
          <button
            className="self-start rounded border border-red-700 px-3 py-1"
            onClick={submitCredentials}
            type="button"
          >
            Try again
          </button>
        </div>
      ) : null}

      <div className="flex items-center gap-3">
        <button
          className="rounded bg-blue-700 px-4 py-2 font-medium text-white disabled:opacity-60"
          disabled={login.isPending}
          type="submit"
        >
          Log in
        </button>
        {login.isPending ? (
          <output
            aria-label="Logging in"
            className="inline-block size-4 animate-spin rounded-full border-2 border-gray-400 border-t-transparent"
          />
        ) : null}
      </div>
    </form>
  );
};
