import { createMutableRegistry } from "@bufbuild/protobuf";
import { file_google_protobuf_wrappers } from "@bufbuild/protobuf/wkt";
import { type ConnectRouter, type HandlerContext, type Interceptor } from "@connectrpc/connect";
import { fastifyConnectPlugin } from "@connectrpc/connect-fastify";
import { createValidateInterceptor } from "@connectrpc/validate";
import { fastify } from "fastify";

import { type CheckConfigRequest } from "../proto/dtkt/base/v1beta1/messages_pb";
import { BaseService } from "../proto/dtkt/base/v1beta1/services_pb";

// Example usage:
// import { runServer } from "./integrationsdk/server";
// void runServer();

const registry = createMutableRegistry(
  file_google_protobuf_wrappers,
);

const DefaultHost = "localhost";
const DefaultPort = 9090;

const BindHost = (typeof process !== "undefined" ? process.env.HOST ?? DefaultHost : DefaultHost);
const BindPort = typeof process !== "undefined" ? process.env.PORT : undefined;

function resolveHost(): string {
  return BindHost;
}

function resolvePort(): number {
  const envPort = parseInt(BindPort ?? "");
  if (envPort > 0) {
    return envPort;
  }
  return DefaultPort;
}

const logger: Interceptor = (next) => async (req) => {
  console.log(`received message on ${req.url}`);
  return await next(req);
};

export type ServerConfig = {
  host: string;
  port: number;
};

export class Server {
  private server;
  private stopPromise!: Promise<void>;
  private config: ServerConfig;

  constructor(
    config: ServerConfig = {
      host: resolveHost(),
      port: resolvePort(),
    },
  ) {
    this.config = config;
    this.server = fastify({});

    this.server.register(fastifyConnectPlugin, {
      interceptors: [
        logger,
        createValidateInterceptor(),
      ],
      routes: (router: ConnectRouter) => {
        router.service(BaseService, {
          checkConfig(_: CheckConfigRequest, __: HandlerContext) {
            return {};
          },
        }, {
          jsonOptions: {
            registry: registry,
          },
        });
      },
    });
  }

  /**
   * Start listening and set up graceful shutdown.
   * @param port TCP port to bind to
   * @param host Host/address to bind to
   */
  async serve(): Promise<void> {
    await this.server.listen({ ...this.config });

    // Prepare stop Promise for graceful shutdown
    this.stopPromise = new Promise<void>((resolve) => {
      const shutdown = () => {
        // this.http2Server.close(() => resolve());
        this.server.close(() => resolve());
      };
      process.on("SIGINT", shutdown);
      process.on("SIGTERM", shutdown);
    });

    // Wait until shutdown is triggered
    await this.stopPromise;
  }

  /**
   * Trigger shutdown programmatically.
   */
  stop(): Promise<void> {
    process.kill(process.pid, "SIGTERM");
    return this.stopPromise;
  }
}

export async function runServer() {
  await (new Server()).serve();
}
