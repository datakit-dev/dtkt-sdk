import { type AnyDesc, type DescFile } from "@bufbuild/protobuf";
import { getComments } from "@bufbuild/protoplugin";

export function getDescription(desc: Exclude<AnyDesc, DescFile>, multiline?: boolean): string {
  const sl = getComments(desc);
  const comments = [...sl.leadingDetached];

  if (sl.leading) {
    comments.push(sl.leading.trim());
  }

  if (sl.trailing) {
    comments.push(sl.trailing.trim());
  }

  if (multiline) {
    return comments.join("\n");
  }

  return comments.join(" ");
}
