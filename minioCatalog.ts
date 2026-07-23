import type { IncomingMessage, ServerResponse } from "node:http";
import { Client, type BucketItem } from "minio";
import type { Plugin } from "vite";

const catalogPath = "/demo-api/narratives";
const narrativePrefix = "summary/cids/";
const narrativeSuffix = ".md";

export type NarrativeCatalogItem = {
  cid: string;
  last_modified: string;
  size_bytes: number;
};

type CatalogConfig = {
  endpoint: string;
  bucket: string;
  accessKey: string;
  secretKey: string;
};

export function narrativeInfoFromObject(object: BucketItem): NarrativeCatalogItem | null {
  if (!("name" in object) || typeof object.name !== "string") {
    return null;
  }
  if (!object.name.startsWith(narrativePrefix) || !object.name.endsWith(narrativeSuffix)) {
    return null;
  }

  const cid = object.name.slice(narrativePrefix.length, -narrativeSuffix.length);
  if (cid === "" || cid.includes("/")) {
    return null;
  }

  return {
    cid,
    last_modified: object.lastModified.toISOString(),
    size_bytes: object.size,
  };
}

function catalogConfig(environment: NodeJS.ProcessEnv): CatalogConfig {
  return {
    endpoint:
      environment.DEMO_MINIO_ENDPOINT ?? environment.S3_ENDPOINT ?? "http://localhost:9000",
    bucket: environment.DEMO_MINIO_BUCKET ?? environment.S3_BUCKET ?? "dev",
    accessKey:
      environment.DEMO_MINIO_ACCESS_KEY ?? environment.S3_ACCESS_KEY ?? "minioadmin",
    secretKey:
      environment.DEMO_MINIO_SECRET_KEY ?? environment.S3_SECRET_KEY ?? "minioadmin",
  };
}

function newMinioClient(config: CatalogConfig): Client {
  const endpoint = new URL(config.endpoint);
  if (endpoint.pathname !== "/") {
    throw new Error("DEMO_MINIO_ENDPOINT must not contain a path");
  }

  return new Client({
    endPoint: endpoint.hostname,
    port: endpoint.port === "" ? (endpoint.protocol === "https:" ? 443 : 80) : Number(endpoint.port),
    useSSL: endpoint.protocol === "https:",
    accessKey: config.accessKey,
    secretKey: config.secretKey,
  });
}

function listNarratives(client: Client, bucket: string): Promise<NarrativeCatalogItem[]> {
  return new Promise((resolve, reject) => {
    const narratives: NarrativeCatalogItem[] = [];
    const stream = client.listObjectsV2(bucket, narrativePrefix, true);

    stream.on("data", (object) => {
      const narrative = narrativeInfoFromObject(object);
      if (narrative !== null) {
        narratives.push(narrative);
      }
    });
    stream.on("error", reject);
    stream.on("end", () => {
      narratives.sort((left, right) => {
        const modifiedOrder = right.last_modified.localeCompare(left.last_modified);
        return modifiedOrder === 0 ? left.cid.localeCompare(right.cid) : modifiedOrder;
      });
      resolve(narratives);
    });
  });
}

function catalogMiddleware(environment: NodeJS.ProcessEnv) {
  const config = catalogConfig(environment);
  const client = newMinioClient(config);

  return async (request: IncomingMessage, response: ServerResponse) => {
    if (request.method !== "GET") {
      response.statusCode = 405;
      response.setHeader("Allow", "GET");
      response.end("Method not allowed");
      return;
    }

    try {
      const narratives = await listNarratives(client, config.bucket);
      response.statusCode = 200;
      response.setHeader("Cache-Control", "no-store");
      response.setHeader("Content-Type", "application/json; charset=utf-8");
      response.end(JSON.stringify({ narratives }));
    } catch (error) {
      console.error(
        "[demo-ui] failed to list MinIO narratives:",
        error instanceof Error ? error.message : error,
      );
      response.statusCode = 502;
      response.setHeader("Cache-Control", "no-store");
      response.setHeader("Content-Type", "application/json; charset=utf-8");
      response.end(JSON.stringify({ error: "Could not list narratives from MinIO." }));
    }
  };
}

export function minioCatalogPlugin(environment: NodeJS.ProcessEnv): Plugin {
  return {
    name: "demo-minio-narrative-catalog",
    configureServer(server) {
      server.middlewares.use(catalogPath, catalogMiddleware(environment));
    },
    configurePreviewServer(server) {
      server.middlewares.use(catalogPath, catalogMiddleware(environment));
    },
  };
}
