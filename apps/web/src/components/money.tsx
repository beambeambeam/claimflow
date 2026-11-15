import { Text } from "@astryxdesign/core/Text";

import { formatMoney } from "#/lib/format";

// An API Money string such as `1250.00`, shown as `฿1,250.00`.
export const Money = ({ value }: { value: string }) => (
  <Text type="body">{formatMoney(value)}</Text>
);
