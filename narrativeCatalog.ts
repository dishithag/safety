import { demoReports, type DemoReport } from "./reports";

export const narrativeCatalogURL = "/demo-api/narratives";

export type NarrativeCatalogItem = {
  cid: string;
  last_modified: string;
  size_bytes: number;
};

export type CatalogReport = DemoReport & NarrativeCatalogItem;

const knownReports = new Map(demoReports.map((report) => [report.cid, report]));

export function parseNarrativeCatalog(value: unknown): CatalogReport[] {
  if (typeof value !== "object" || value === null || !("narratives" in value)) {
    throw new Error("The demo server returned an invalid narrative catalog.");
  }

  const narratives = (value as { narratives: unknown }).narratives;
  if (!Array.isArray(narratives)) {
    throw new Error("The demo server returned an invalid narrative catalog.");
  }

  return narratives.map((narrative): CatalogReport => {
    if (
      typeof narrative !== "object" ||
      narrative === null ||
      typeof narrative.cid !== "string" ||
      narrative.cid.trim() === "" ||
      typeof narrative.last_modified !== "string" ||
      Number.isNaN(Date.parse(narrative.last_modified)) ||
      typeof narrative.size_bytes !== "number" ||
      !Number.isFinite(narrative.size_bytes) ||
      narrative.size_bytes < 0
    ) {
      throw new Error("The demo server returned invalid narrative metadata.");
    }

    return {
      cid: narrative.cid,
      last_modified: narrative.last_modified,
      size_bytes: narrative.size_bytes,
      platforms: knownReports.get(narrative.cid)?.platforms ?? [],
    };
  });
}

export function formatObjectSize(sizeBytes: number): string {
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  return `${(sizeBytes / 1024).toFixed(1)} KiB`;
}
