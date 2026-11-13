import { TextInput } from "@astryxdesign/core/TextInput";
import type { TextInputProps } from "@astryxdesign/core/TextInput";
import { createFormHook, createFormHookContexts } from "@tanstack/react-form";
import { z } from "zod";

const { fieldContext, formContext, useFieldContext } = createFormHookContexts();

// Standard Schema validators (Zod) report issues as objects with a message.
const zIssue = z.object({ message: z.string() });

const firstErrorMessage = (errors: unknown[]) => {
  const issue = zIssue.safeParse(errors[0]);

  return issue.success ? issue.data.message : undefined;
};

type TextFieldProps = Omit<
  TextInputProps,
  "onBlur" | "onChange" | "status" | "value"
>;

// Astryx TextInput bound to the surrounding form field: value, change, blur, and first error.
const TextField = (props: TextFieldProps) => {
  const field = useFieldContext<string>();
  const message = firstErrorMessage(field.state.meta.errors);

  return (
    <TextInput
      {...props}
      onBlur={field.handleBlur}
      onChange={field.handleChange}
      status={message === undefined ? undefined : { type: "error", message }}
      value={field.state.value}
    />
  );
};

export const { useAppForm } = createFormHook({
  fieldContext,
  fieldComponents: { TextField },
  formContext,
  formComponents: {},
});
