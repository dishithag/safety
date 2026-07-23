import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const trendDirectory = path.resolve(scriptDirectory, "..", "src", "demo-data", "trends");
const fixture = JSON.parse(await readFile(path.join(trendDirectory, "portfolio-week.json"), "utf8"));

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function equalScore(actual, expected, context) {
  const normalizedActual = actual ?? null;
  const normalizedExpected = expected ?? null;
  assert(
    normalizedActual === normalizedExpected,
    `${context}: got ${normalizedActual}, want ${normalizedExpected}`,
  );
}

assert(fixture.metadata.synthetic === true, "fixture must be labelled synthetic");
assert(fixture.cids.length === fixture.metadata.cid_count, "CID count does not match metadata");

for (const cidHistory of fixture.cids) {
  assert(
    cidHistory.snapshots.length === fixture.metadata.snapshot_count,
    `${cidHistory.cid}: snapshot count does not match metadata`,
  );
  const seed = JSON.parse(
    await readFile(path.join(trendDirectory, "seed-reports", `${cidHistory.cid}.json`), "utf8"),
  ).report;
  const latest = cidHistory.snapshots.at(-1);

  assert(latest.date === fixture.metadata.period_end, `${cidHistory.cid}: final date mismatch`);
  assert(latest.reported_devices === seed.num_aids, `${cidHistory.cid}: final device count mismatch`);
  equalScore(latest.scores.overall, seed.average_overall_score, `${cidHistory.cid}: overall score`);
  equalScore(latest.scores.os, seed.average_os_score, `${cidHistory.cid}: OS score`);
  equalScore(latest.scores.sensor, seed.average_sensor_config_score, `${cidHistory.cid}: sensor score`);
  assert(latest.platforms.length === seed.platforms.length, `${cidHistory.cid}: platform count mismatch`);

  for (const platform of latest.platforms) {
    const sourcePlatform = seed.platforms.find((candidate) => candidate.name === platform.name);
    assert(sourcePlatform, `${cidHistory.cid}: unknown platform ${platform.name}`);
    assert(
      platform.reported_devices === sourcePlatform.num_aids,
      `${cidHistory.cid}/${platform.name}: device count mismatch`,
    );
    equalScore(
      platform.scores.overall,
      sourcePlatform.average_overall_score,
      `${cidHistory.cid}/${platform.name}: overall score`,
    );
    equalScore(
      platform.scores.os,
      sourcePlatform.average_os_score,
      `${cidHistory.cid}/${platform.name}: OS score`,
    );
    equalScore(
      platform.scores.sensor,
      sourcePlatform.average_sensor_config_score,
      `${cidHistory.cid}/${platform.name}: sensor score`,
    );

    for (const signal of platform.signals) {
      equalScore(
        signal.compliance,
        sourcePlatform.compliance[signal.id],
        `${cidHistory.cid}/${platform.name}/${signal.id}: compliance`,
      );
    }
  }

  for (const snapshot of cidHistory.snapshots) {
    for (const score of Object.values(snapshot.scores)) {
      assert(score == null || (score >= 0 && score <= 100), `${cidHistory.cid}: invalid score ${score}`);
    }
    for (const platform of snapshot.platforms) {
      for (const signal of platform.signals) {
        assert(
          signal.compliance >= 0 && signal.compliance <= 1,
          `${cidHistory.cid}/${platform.name}/${signal.id}: invalid compliance`,
        );
      }
    }
  }
}

console.log(
  `Validated ${fixture.cids.length} synthetic CID histories; all final snapshots match their seed reports`,
);
