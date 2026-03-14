import { type Message, type Registry } from "@bufbuild/protobuf";

import { type FieldGroup } from "./field";

export type Env = {
  resolver(): Resolver;
  onGroupCompleted(group: FieldGroup): Promise<void>;
};

export type Resolver = {
  invokeMethod(name: string, req: Message): Promise<Message>;
} & Registry;
