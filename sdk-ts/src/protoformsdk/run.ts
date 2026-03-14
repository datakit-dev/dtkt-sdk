import { create, type Message } from "@bufbuild/protobuf";

import { type Env } from "./form/env";

export type Driver = {
  runProtoform(msg: Message): Promise<void>;
} & Env;

export async function runWithMessage(drv: Driver, msg: Message): Promise<void> {
  return drv.runProtoform(msg);
}

export async function runWithMessageName(
  drv: Driver,
  name: string,
): Promise<Message> {
  const msgType = drv.resolver().getMessage(name);
  if (msgType === undefined) throw new Error(`message type not found: ${name}`);

  const msg = create(msgType);
  await drv.runProtoform(msg);

  return msg;
}
