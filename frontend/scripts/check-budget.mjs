#!/usr/bin/env node
/**
 * RESPECT budget gate (WP-4.2): fails the build when any route's
 * First Load JS exceeds the budget.
 *
 * Usage:
 *   node scripts/check-budget.mjs                 # builds and checks
 *   node scripts/check-budget.mjs --file build.log  # checks an existing build log
 *   node scripts/check-budget.mjs --budget-kb 500   # custom budget
 *
 * Parses the `next build` route table — every "First Load JS" cell under
 * the budget is green, any overage exits 1 with the offenders listed. The
 * check is honest: it reads the real compiled output, never an estimate.
 */
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const args = process.argv.slice(2);

function flag(name) {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : undefined;
}

const logFile = flag("--file");
const budgetKB = Number(flag("--budget-kb") ?? "500");
if (!Number.isFinite(budgetKB) || budgetKB <= 0) {
  console.error(`check-budget: invalid --budget-kb "${flag("--budget-kb")}"`);
  process.exit(2);
}

// Route table line example (Next.js 14):
//   ├ ○ /dashboard                           10.2 kB         174 kB
// The First Load JS is the LAST size token on the line.
const rowRe = /^[├└│┌] .*?(\/[^\s]*)\s+[\d.]+\s*kB\s+([\d.]+)\s*(kB|MB)\s*$/;

function parseBuildLog(text) {
  const rows = [];
  for (const line of text.split("\n")) {
    const m = line.match(rowRe);
    if (!m) continue;
    const [, route, size, unit] = m;
    rows.push({ route, kb: Number(size) * (unit === "MB" ? 1000 : 1) });
  }
  return rows;
}

let output;
if (logFile) {
  output = readFileSync(logFile, "utf8");
  finish(output);
} else {
  console.log(`check-budget: running "npm run build" (budget ${budgetKB} kB First Load JS)...\n`);
  const build = spawn("npm", ["run", "build"], { cwd: root });
  let buf = "";
  build.stdout.on("data", (d) => {
    buf += d;
    process.stdout.write(d);
  });
  build.stderr.on("data", (d) => process.stderr.write(d));
  build.on("close", (code) => {
    if (code !== 0) process.exit(1);
    finish(buf);
  });
}

function finish(text) {
  const rows = parseBuildLog(text);
  if (rows.length === 0) {
    console.error("check-budget: no route table found in build output — cannot verify the budget.");
    process.exit(2);
  }
  const offenders = rows.filter((r) => r.kb > budgetKB);
  console.log(`\ncheck-budget: ${rows.length} routes checked against ${budgetKB} kB First Load JS budget`);
  for (const r of rows) {
    const mark = r.kb > budgetKB ? "  OVER" : "  ok";
    console.log(`${mark}  ${r.route.padEnd(28)} ${r.kb.toFixed(1).padStart(8)} kB`);
  }
  if (offenders.length > 0) {
    console.error(
      `\ncheck-budget: FAIL — ${offenders.length} route(s) exceed ${budgetKB} kB:\n` +
        offenders.map((r) => `  ${r.route}: ${r.kb.toFixed(1)} kB`).join("\n") +
        "\nHint: lazy-load heavy sections, or move rarely-used code into dynamic imports."
    );
    process.exit(1);
  }
  console.log("check-budget: PASS — all routes within budget");
}