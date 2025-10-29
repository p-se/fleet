import * as fs from "fs";
import * as core from "@actions/core";

function readProcStat() {
  const ln = fs
    .readFileSync("/proc/stat", "utf8")
    .split("\n")
    .find((l) => l.startsWith("cpu "));
  const parts = ln.trim().split(/\s+/).slice(1).map(Number);
  const [user, nice, system, idle, iowait, irq, softirq, steal] = parts;
  const total = user + nice + system + idle + iowait + irq + softirq + steal;
  return { total, steal };
}

try {
  const { total, steal } = readProcStat();
  core.saveState("start_total", String(total));
  core.saveState("start_steal", String(steal));
  core.info(
    `Steal Meter: captured start snapshot (total=${total}, steal=${steal})`,
  );
} catch (e) {
  core.setFailed(`Failed to read /proc/stat: ${e}`);
}
