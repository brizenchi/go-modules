import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, readdirSync, statSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(scriptDir, "..");
const outDir = mkdtempSync(path.join(os.tmpdir(), "go-modules-quickstart-nextjs-test-"));
const tscCli = path.join(rootDir, "node_modules", "typescript", "lib", "tsc.js");

function collectTestFiles(dir) {
  const entries = readdirSync(dir);
  const files = [];
  for (const entry of entries) {
    const fullPath = path.join(dir, entry);
    const stats = statSync(fullPath);
    if (stats.isDirectory()) {
      files.push(...collectTestFiles(fullPath));
      continue;
    }
    if (entry.endsWith(".test.js")) {
      files.push(fullPath);
    }
  }
  return files.sort();
}

try {
  execFileSync(
    process.execPath,
    [tscCli, "-p", "tsconfig.test.json", "--outDir", outDir, "--pretty", "false"],
    {
      cwd: rootDir,
      stdio: "inherit"
    }
  );

  const testsDir = path.join(outDir, "tests");
  const testFiles = collectTestFiles(testsDir);
  if (testFiles.length === 0) {
    throw new Error(`no compiled test files found under ${testsDir}`);
  }

  execFileSync(process.execPath, ["--test", ...testFiles], {
    cwd: rootDir,
    env: {
      ...process.env,
      NODE_PATH: path.join(rootDir, "node_modules")
    },
    stdio: "inherit"
  });
} finally {
  rmSync(outDir, { recursive: true, force: true });
}
