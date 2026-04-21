#!/usr/bin/env bun
import { existsSync, readFileSync, readdirSync, unlinkSync, mkdirSync } from "fs";
import { watch as fsWatch } from "fs";
import { resolve, join, basename } from "path";
import { fileURLToPath } from "url";
import { tmpdir } from "os";

const PLUGIN_DIR = resolve(dirname(fileURLToPath(import.meta.url)));
const PID_FILE = "/tmp/claude-agents-mcp.pid";

function dirname(path: string): string {
  return path.slice(0, path.lastIndexOf("/")) || ".";
}

// --- helpers ---

function printErr(msg: string)  { console.error(`  ✗ ${msg}`); }
function printWarn(msg: string) { console.log(`  ⚠ ${msg}`); }
function printOk(msg: string)   { console.log(`  ✓ ${msg}`); }

async function tmux(...args: string[]): Promise<void> {
  const proc = Bun.spawn(["tmux", ...args], { stdout: "inherit", stderr: "inherit" });
  await proc.exited;
}

async function tmuxOut(...args: string[]): Promise<string> {
  const proc = Bun.spawn(["tmux", ...args], { stdout: "pipe", stderr: "pipe" });
  return new Response(proc.stdout).text().then(s => s.trim());
}

function findRoleFile(role: string): string | null {
  const project = `.claude/roles/${role}.md`;
  const builtin = join(PLUGIN_DIR, `roles/${role}.md`);
  if (existsSync(project)) return project;
  if (existsSync(builtin)) return builtin;
  return null;
}

function findSkillFile(skill: string): string | null {
  const project = `.claude/skills/${skill}.md`;
  const builtin = join(PLUGIN_DIR, `skills/${skill}.md`);
  if (existsSync(project)) return project;
  if (existsSync(builtin)) return builtin;
  return null;
}

export function sectionLines(content: string, section: string): string[] {
  const lines = content.split("\n");
  const result: string[] = [];
  let inSection = false;
  for (const line of lines) {
    if (line.startsWith(`## ${section}`)) { inSection = true; continue; }
    if (inSection && line.startsWith("## ")) break;
    if (inSection) result.push(line);
  }
  return result;
}

export function parseFrontmatter(content: string): Record<string, string> {
  const match = content.match(/^---\n([\s\S]*?)\n---/);
  if (!match) return {};
  const result: Record<string, string> = {};
  for (const line of match[1].split("\n")) {
    const colonIdx = line.indexOf(":");
    if (colonIdx !== -1) {
      result[line.slice(0, colonIdx).trim()] = line.slice(colonIdx + 1).trim();
    }
  }
  return result;
}

// --- pure logic (exported for testing) ---

export interface StartArgs {
  useCurrentPane: boolean;
  configPath: string;
  jobName: string;
}

export function parseStartArgs(args: string[]): StartArgs {
  let useCurrentPane = false;
  let configPath = ".claude/agents.json";
  let jobName = "";
  for (const arg of args) {
    if (arg === "--here") useCurrentPane = true;
    else if (arg.startsWith("--config=")) configPath = arg.slice(9);
    else if (arg.startsWith("--job=")) jobName = arg.slice(6);
    else configPath = arg;
  }
  return { useCurrentPane, configPath, jobName };
}

export function applyTemplate(template: string, vars: Record<string, string>): string {
  return Object.entries(vars).reduce(
    (t, [k, v]) => t.replace(new RegExp(`\\{\\{${k}\\}\\}`, "g"), v),
    template
  );
}

export function shouldProcessFile(filePath: string, seen: Set<string>): boolean {
  if (!basename(filePath).endsWith(".md")) return false;
  if (filePath.includes("/done/")) return false;
  if (seen.has(filePath)) return false;
  return true;
}

export function buildMenuArgs(cliPath: string, workers: string[]): string[] {
  const args: string[] = [
    "Status",  "s", `run-shell 'bun run "${cliPath}" menu show status'`,
    "Queue",   "q", `run-shell 'bun run "${cliPath}" menu show queue'`,
    "Results", "r", `run-shell 'bun run "${cliPath}" menu show results'`,
    "", "", "",
  ];
  for (const w of workers) {
    args.push(`Worker ${w}`, "", `run-shell 'bun run "${cliPath}" menu show result/${w}'`);
  }
  return args;
}

// --- types ---

interface WorkerConfig {
  id: string;
  role: string;
}

interface StageConfig {
  name: string;
  role: string;
}

interface PipelineConfig {
  name: string;
  stages: StageConfig[];
}

interface AgentsConfig {
  workers: WorkerConfig[];
  pipelines: PipelineConfig[];
}

// --- validate ---

export async function validate(args: string[]): Promise<boolean> {
  let configPath = ".claude/agents.json";
  let jobName = "";
  for (const arg of args) {
    if (arg.startsWith("--config=")) configPath = arg.slice(9);
    else if (arg.startsWith("--job=")) jobName = arg.slice(6);
  }

  let errors = 0;
  let warnings = 0;

  console.log(`Checking ${configPath}...`);

  if (!existsSync(configPath)) {
    printErr(`agents.json not found: ${configPath}`);
    return false;
  }

  let config: AgentsConfig;
  try {
    config = JSON.parse(readFileSync(configPath, "utf8"));
  } catch {
    printErr("agents.json is not valid JSON");
    return false;
  }

  // workers
  console.log("\nWorkers:");
  const workers: WorkerConfig[] = config.workers ?? [];
  if (workers.length === 0) {
    printErr("no workers defined in agents.json"); errors++;
  }
  for (const { id: workerId, role } of workers) {
    const roleFile = findRoleFile(role);
    if (!roleFile) {
      printErr(`worker '${workerId}': role file not found for '${role}'`); errors++;
      continue;
    }
    printOk(`worker '${workerId}' (${role}): role file found`);

    const content = readFileSync(roleFile, "utf8");

    for (const line of sectionLines(content, "Skills")) {
      const m = line.match(/-\s+`\/([a-z0-9_-]+)`/);
      if (m) {
        if (!findSkillFile(m[1])) {
          printErr(`worker '${workerId}' (${role}): skill '/${m[1]}' not found`); errors++;
        } else {
          printOk(`worker '${workerId}' (${role}): skill '/${m[1]}' found`);
        }
      }
    }

    for (const line of sectionLines(content, "Plugins")) {
      const m = line.match(/-\s+`([a-z0-9_-]+)`/);
      if (m) {
        printWarn(`worker '${workerId}' (${role}): plugin '${m[1]}' listed — verify it is enabled in Claude Code settings`);
        warnings++;
      }
    }
  }

  // pipelines
  console.log("\nPipelines:");
  const pipelines: PipelineConfig[] = config.pipelines ?? [];
  if (pipelines.length === 0) {
    printWarn("no pipelines defined in agents.json"); warnings++;
  }
  for (const pipeline of pipelines) {
    let pipelineOk = true;
    for (const stage of pipeline.stages) {
      if (!findRoleFile(stage.role)) {
        printErr(`pipeline '${pipeline.name}', stage '${stage.name}': role '${stage.role}' has no role file`);
        errors++; pipelineOk = false;
      }
    }
    if (pipelineOk) printOk(`pipeline '${pipeline.name}': ${pipeline.stages.length} stages`);
  }

  // active job conflict check (if MCP is running)
  const port = process.env.CLAUDE_AGENTS_MCP_PORT ?? "7777";
  const mcpUrl = process.env.MCP_URL ?? `http://localhost:${port}`;
  if (jobName) {
    try {
      const res = await fetch(`${mcpUrl}/jobs`, { signal: AbortSignal.timeout(1000) });
      if (res.ok) {
        const jobs = await res.json() as Record<string, unknown>;
        if (jobName in jobs) {
          printErr(`job '${jobName}' is already active in the running MCP server — use reset_job to rerun it`);
          errors++;
        }
      }
    } catch { /* MCP not running — fine */ }
  }

  // job file
  if (jobName) {
    console.log(`\nJob: ${jobName}`);

    if (existsSync(`.claude/jobs/done/${jobName}.md`)) {
      printErr(`job '${jobName}' already completed — move from .claude/jobs/done/ to rerun`); errors++;
    } else {
      const jobFile = `.claude/jobs/${jobName}.md`;
      if (!existsSync(jobFile)) {
        printErr(`job file not found: ${jobFile}`); errors++;
      } else {
        const fm = parseFrontmatter(readFileSync(jobFile, "utf8"));
        if (!fm.pipeline) {
          printErr(`job '${jobName}': missing 'pipeline:' in frontmatter`); errors++;
        } else if (!pipelines.some(p => p.name === fm.pipeline)) {
          printErr(`job '${jobName}': pipeline '${fm.pipeline}' not defined in agents.json`); errors++;
        } else {
          printOk(`job '${jobName}': pipeline '${fm.pipeline}'`);
        }
        if (!fm.domain) {
          printErr(`job '${jobName}': missing 'domain:' in frontmatter`); errors++;
        } else {
          printOk(`job '${jobName}': domain '${fm.domain}'`);
        }
      }
    }
  }

  console.log("");
  if (errors > 0) {
    console.error(`Validation failed — ${errors} error(s), ${warnings} warning(s)`);
    return false;
  }
  console.log(warnings > 0 ? `Validation passed with ${warnings} warning(s)` : "Validation passed");
  return true;
}

// --- start-mcp ---

async function startMcp(args: string[]): Promise<void> {
  const port = args[0] ?? process.env.CLAUDE_AGENTS_MCP_PORT ?? "7777";

  if (existsSync(PID_FILE)) {
    const pid = readFileSync(PID_FILE, "utf8").trim();
    const check = Bun.spawn(["kill", "-0", pid], { stdout: "pipe", stderr: "pipe" });
    await check.exited;
    if (check.exitCode === 0) {
      console.log(`MCP server already running (pid ${pid})`);
      return;
    }
  }

  const server = Bun.spawn(
    ["bun", "run", join(PLUGIN_DIR, "mcp/server.ts"), "--port", String(port)],
    { stdout: "inherit", stderr: "inherit", detached: true }
  );
  await Bun.write(PID_FILE, String(server.pid));
  console.log(`MCP server started on port ${port} (pid ${server.pid})`);
}

// --- start ---

async function start(args: string[]): Promise<void> {
  const { useCurrentPane, configPath, jobName } = parseStartArgs(args);

  const port = process.env.CLAUDE_AGENTS_MCP_PORT ?? "7777";
  const mcpUrl = `http://localhost:${port}`;

  const validateArgs = [`--config=${configPath}`];
  if (jobName) validateArgs.push(`--job=${jobName}`);
  if (!await validate(validateArgs)) process.exit(1);

  const jobFile = jobName ? `.claude/jobs/${jobName}.md` : "";
  if (jobName) mkdirSync(".claude/jobs/done", { recursive: true });

  await startMcp([port]);
  await Bun.sleep(1000);

  let orchPane: string;
  if (useCurrentPane) {
    orchPane = process.env.TMUX_PANE ?? "";
    await tmux("setenv", "MCP_URL", mcpUrl);
    await tmux("setenv", "AGENTS_CONFIG", configPath);
    if (jobFile) await tmux("setenv", "JOB_FILE", jobFile);
  } else {
    const envArgs = ["-e", `MCP_URL=${mcpUrl}`, "-e", `AGENTS_CONFIG=${configPath}`];
    if (jobFile) envArgs.push("-e", `JOB_FILE=${jobFile}`);
    orchPane = await tmuxOut("new-window", "-P", "-F", "#{pane_id}", "-n", "agents", ...envArgs);
  }

  const prompt = applyTemplate(
    readFileSync(join(PLUGIN_DIR, "templates/orchestrator.md"), "utf8"),
    { mcp_url: mcpUrl, agents_config: configPath, job_file: jobFile }
  );

  await tmux("send-keys", "-t", orchPane, "claude", "Enter");
  await Bun.sleep(3000);

  // pipe prompt into tmux buffer via temp file
  const tmpFile = join(tmpdir(), `orch-prompt-${Date.now()}.md`);
  await Bun.write(tmpFile, prompt);
  await tmux("load-buffer", "-b", "orch-prompt", tmpFile);
  unlinkSync(tmpFile);

  await tmux("paste-buffer", "-b", "orch-prompt", "-t", orchPane);
  await tmux("send-keys", "-t", orchPane, "", "Enter");
  try { await tmux("delete-buffer", "-b", "orch-prompt"); } catch { /* ok */ }

  if ((process.env.CLAUDE_AGENTS_WATCH_JOBS ?? "false") === "true") {
    if (existsSync(".claude/jobs")) {
      await tmux("split-window", "-d", "-h", "-t", orchPane,
        "-e", `AGENTS_CONFIG=${configPath}`,
        `bun run "${join(PLUGIN_DIR, "cli.ts")}" watch "${orchPane}" .claude/jobs`);
      console.log("Job watcher started (watching .claude/jobs/)");
    } else {
      console.log("Job watcher enabled but .claude/jobs/ not found — skipping");
    }
  }

  console.log(`Orchestrator started in pane ${orchPane}. MCP: ${mcpUrl}${jobName ? `, Job: ${jobName}` : ""}`);
}

// --- watch ---

async function watch(args: string[]): Promise<void> {
  const orchPane = args[0];
  const jobsDir = args[1] ?? ".claude/jobs";
  const configPath = process.env.AGENTS_CONFIG ?? ".claude/agents.json";

  if (!orchPane) {
    console.error("usage: cli.ts watch <orch_pane_id> [jobs_dir]");
    process.exit(1);
  }

  const seen = new Set<string>();

  async function onNewFile(filePath: string): Promise<void> {
    if (!shouldProcessFile(filePath, seen)) return;
    if (!existsSync(filePath)) return;
    seen.add(filePath);

    const job = basename(filePath).slice(0, -3);
    console.log(`[watch] detected: ${job}`);

    if (await validate([`--config=${configPath}`, `--job=${job}`])) {
      console.log(`[watch] sending 'start job ${job}' to pane ${orchPane}`);
      await tmux("send-keys", "-t", orchPane, `start job ${job}`, "Enter");
    } else {
      console.error(`[watch] validation failed for '${job}' — not starting`);
      await notifyFn("watcher", "blocked");
    }
  }

  console.log(`[watch] watching ${jobsDir} (orchestrator pane: ${orchPane})`);

  const watcher = fsWatch(jobsDir, { recursive: false }, async (_event, filename) => {
    if (filename) await onNewFile(join(jobsDir, filename));
  });

  process.on("SIGINT", () => { watcher.close(); process.exit(0); });
  process.on("SIGTERM", () => { watcher.close(); process.exit(0); });
  await new Promise<never>(() => {});
}

// --- menu ---

async function menu(args: string[]): Promise<void> {
  const port = process.env.CLAUDE_AGENTS_MCP_PORT ?? "7777";
  const baseUrl = `http://localhost:${port}`;
  const cliPath = join(PLUGIN_DIR, "cli.ts");

  if (args[0] === "show") {
    const path = args[1] ?? "status";
    const title = path.replace(/\//g, " ").replace(/\b\w/g, c => c.toUpperCase());
    const cmd = `curl -sf '${baseUrl}/${path}' | python3 -m json.tool 2>/dev/null || echo 'MCP server not running'; read -r -p '' -n1`;
    await tmux("display-popup", "-E", "-T", ` ${title} `, cmd);
    return;
  }

  let workers: string[] = [];
  try {
    const res = await fetch(`${baseUrl}/status`, { signal: AbortSignal.timeout(1000) });
    if (res.ok) {
      const data = await res.json() as { workers?: Record<string, unknown> };
      workers = Object.keys(data.workers ?? {});
    }
  } catch { /* server not running */ }

  await tmux("display-menu", "-T", " Claude Agents ", ...buildMenuArgs(cliPath, workers));
}

// --- cleanup ---

async function cleanup(): Promise<void> {
  if (existsSync(PID_FILE)) {
    const pid = readFileSync(PID_FILE, "utf8").trim();
    const kill = Bun.spawn(["kill", pid], { stdout: "pipe", stderr: "pipe" });
    await kill.exited;
    if (kill.exitCode === 0) console.log(`MCP server stopped (pid ${pid})`);
    unlinkSync(PID_FILE);
  }

  const gitCheck = Bun.spawn(["git", "rev-parse", "--git-dir"], { stdout: "pipe", stderr: "pipe" });
  await gitCheck.exited;
  if (gitCheck.exitCode !== 0 || !existsSync(".worktrees")) return;

  for (const entry of readdirSync(".worktrees")) {
    const worktreePath = `.worktrees/${entry}`;
    if (!existsSync(worktreePath)) continue;

    const branchProc = Bun.spawn(
      ["git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD"],
      { stdout: "pipe", stderr: "pipe" }
    );
    const branch = (await new Response(branchProc.stdout).text()).trim();

    const rm = Bun.spawn(["git", "worktree", "remove", "--force", worktreePath], {
      stdout: "inherit", stderr: "inherit",
    });
    await rm.exited;
    console.log(`Removed worktree ${worktreePath}`);

    if (branch.startsWith("agent/")) {
      const del = Bun.spawn(["git", "branch", "-d", branch], { stdout: "pipe", stderr: "pipe" });
      await del.exited;
      console.log(del.exitCode === 0
        ? `Deleted branch ${branch}`
        : `Branch ${branch} not deleted (likely open PR — delete manually after merge)`);
    }
  }
}

// --- notify ---

async function notifyFn(workerId: string, state: string): Promise<void> {
  const isBlocked = state === "blocked";
  const sound = isBlocked ? "Basso" : "Glass";
  const msg = isBlocked ? `Worker ${workerId} is blocked` : `Worker ${workerId} finished`;
  const proc = Bun.spawn(
    ["osascript", "-e", `display notification "${msg}" with title "Claude Agent" sound name "${sound}"`],
    { stdout: "pipe", stderr: "pipe" }
  );
  await proc.exited;
}

async function notify(args: string[]): Promise<void> {
  await notifyFn(args[0] ?? "?", args[1] ?? "done");
}

// --- dispatch ---

if (!import.meta.main) { /* imported as module — skip dispatch */ }
else {

const [,, subcmd, ...rest] = process.argv;

switch (subcmd) {
  case "validate":  process.exit((await validate(rest)) ? 0 : 1);
  case "start":     await start(rest); break;
  case "start-mcp": await startMcp(rest); break;
  case "watch":     await watch(rest); break;
  case "menu":      await menu(rest); break;
  case "cleanup":   await cleanup(); break;
  case "notify":    await notify(rest); break;
  default:
    console.error("Usage: cli.ts <validate|start|start-mcp|watch|menu|cleanup|notify> [args...]");
    process.exit(1);
}

}
