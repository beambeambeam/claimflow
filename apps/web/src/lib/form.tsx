import type { ISODateString } from "@astryxdesign/core/Calendar";
import { DateInput } from "@astryxdesign/core/DateInput";
import type { DateInputProps } from "@astryxdesign/core/DateInput";
import { Selector } from "@astryxdesign/core/Selector";
import type { SelectorProps } from "@astryxdesign/core/Selector";
import { TextArea } from "@astryxdesign/core/TextArea";
import type { TextAreaProps } from "@astryxdesign/core/TextArea";
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

// The Astryx `status` prop for the first error of a field, or undefined while it has none.
const errorStatus = (errors: unknown[]) => {
  const message = firstErrorMessage(errors);

  return message === undefined
    ? undefined
    : { type: "error" as const, message };
};

type TextFieldProps = Omit<
  TextInputProps,
  "onBlur" | "onChange" | "status" | "value"
>;

// Astryx TextInput bound to the surrounding form field: value, change, blur, and first error.
const TextField = (props: TextFieldProps) => {
  const field = useFieldContext<string>();

  return (
    <TextInput
      {...props}
      onBlur={field.handleBlur}
      onChange={field.handleChange}
      status={errorStatus(field.state.meta.errors)}
      value={field.state.value}
    />
  );
};

type TextAreaFieldProps = Omit<
  TextAreaProps,
  "onBlur" | "onChange" | "status" | "value"
>;

// Astryx TextArea bound to the surrounding form field.
const TextAreaField = (props: TextAreaFieldProps) => {
  const field = useFieldContext<string>();

  return (
    <TextArea
      {...props}
      onBlur={field.handleBlur}
      onChange={field.handleChange}
      status={errorStatus(field.state.meta.errors)}
      value={field.state.value}
    />
  );
};

interface SelectFieldProps {
  isLabelHidden?: boolean;
  label: string;
  options: SelectorProps["options"];
  placeholder?: string;
}

// Astryx Selector bound to the surrounding form field. An empty string means nothing is chosen.
const SelectField = (props: SelectFieldProps) => {
  const field = useFieldContext<string>();

  return (
    <Selector
      {...props}
      onChange={field.handleChange}
      status={errorStatus(field.state.meta.errors)}
      value={field.state.value === "" ? undefined : field.state.value}
    />
  );
};

type DateFieldProps = Omit<
  DateInputProps,
  "onBlur" | "onChange" | "status" | "value"
>;

const isoDate = /^\d{4}-\d{2}-\d{2}$/u;

const isIsoDate = (value: string): value is ISODateString =>
  isoDate.test(value);

// Astryx DateInput bound to the surrounding form field. The value is `YYYY-MM-DD`, or "" while empty.
const DateField = (props: DateFieldProps) => {
  const field = useFieldContext<string>();
  const { value } = field.state;

  return (
    <DateInput
      {...props}
      onChange={(date) => {
        field.handleChange(date ?? "");
      }}
      status={errorStatus(field.state.meta.errors)}
      value={isIsoDate(value) ? value : undefined}
    />
  );
};

export const { useAppForm } = createFormHook({
  fieldContext,
  fieldComponents: { DateField, SelectField, TextAreaField, TextField },
  formContext,
  formComponents: {},
});
