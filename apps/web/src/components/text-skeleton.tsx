import { Skeleton } from "@astryxdesign/core/Skeleton";

// A placeholder one line of text tall. The height comes from the type style's size and leading tokens, so it matches
// the text it stands in for under any theme.
export const TextSkeleton = ({
  type,
  width = "100%",
}: {
  type: "body" | "large" | "heading-1";
  width?: string;
}) => (
  <Skeleton
    height={`calc(var(--text-${type}-size) * var(--text-${type}-leading))`}
    width={width}
  />
);
