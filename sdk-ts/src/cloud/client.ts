import { type DocumentNode, Kind, print } from "graphql";
import { type GraphQLResponse } from "relay-runtime";

import { getSdk, type Requester } from "../graphql/generated";

const defaultHeaders = {
  "Content-Type": "application/json",
} as const;

export class GraphClient {
  private url: string;
  private headers: Record<string, string>;

  constructor(url: string, headers: Record<string, string>) {
    this.url = url;
    this.headers = {
      ...defaultHeaders,
      ...headers,
    };
  }

  async request<R>(
    doc: DocumentNode,
    variables?: unknown,
    options?: { headers?: Record<string, string> },
  ): Promise<R> {
    const query = print(doc);
    const resp = await fetch(this.url, {
      method: "post",
      headers: {
        ...this.headers,
        ...options?.headers,
      },
      body: JSON.stringify({
        query,
        variables,
      }),
    });

    const json = (await resp.json()) as GraphQLResponse;

    // GraphQL returns exceptions (for example, a missing required variable) in the "errors"
    // property of the response. If any exceptions occurred when processing the request,
    // throw an error to indicate to the developer what went wrong.
    if ("errors" in json && Array.isArray(json.errors)) {
      throw new Error(
        `Error fetching GraphQL query '${doc.definitions[0]?.kind === Kind.OPERATION_DEFINITION ? doc.definitions[0].name?.value : "unknown"
        }' with variables '${JSON.stringify(variables)}': ${JSON.stringify(
          json.errors,
        )}`,
      );
    }

    if (!("data" in json)) {
      throw new Error(
        `No data returned from GraphQL query '${doc.definitions[0]?.kind === Kind.OPERATION_DEFINITION ? doc.definitions[0].name?.value : "unknown"
        }' with variables '${JSON.stringify(variables)}'`,
      );
    }

    return json.data as R;
  }
}

export function getGraphClient(url: string, headers: Record<string, string>) {
  const client = new GraphClient(url, headers);
  const requester: Requester = <R>(
    doc: DocumentNode,
    variables?: unknown,
    options?: { headers?: Record<string, string> },
  ): Promise<R> | AsyncIterable<R> => {
    return client.request<R>(doc, variables, options);
  };

  return getSdk(requester);
}
