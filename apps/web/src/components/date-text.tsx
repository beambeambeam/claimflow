import { Text } from "@astryxdesign/core/Text";

import { formatDate, formatDateTime } from "#/lib/format";

// A timestamp or date in the Asia/Bangkok calendar: `5 Nov 2026`, or with `withTime` `5 Nov 2026, 10:00`.
// A missing date, such as a claim that was never submitted, shows as a dash.
export const DateText = ({
  value,
  withTime = false,
}: {
  value: string | null;
  withTime?: boolean;
}) => (
  <Text type="body">
    {value === null ? "–" : (withTime ? formatDateTime : formatDate)(value)}
  </Text>
);
