import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "fs";
import { tmpdir } from "os";
import { join } from "path";
import { parseFrontmatter, sectionLines, validate } from "./cli.ts";

// --- pure helpers ---

describe("parseFrontmatter", () => {
  it("returns empty object when no frontmatter", () => {
    expect(parseFrontmatter("# just a heading\nsome content")).toEqual({});
  });

  it("parses key: value pairs", () => {
    expect(parseFrontmatter("---\npipeline: frontend\ndomain: auth\n---")).toEqual({
      pipeline: "frontend",
      domain: "auth",
    });
  });

  it("handles values containing colons", () => {
    expect(parseFrontmatter("---\nurl: http://localhost:7777\n---")).toEqual({
      url: "http://localhost:7777",
    });
  });

  it("returns empty object when closing --- is missing", () => {
    expect(parseFrontmatter("---\npipeline: frontend\n")).toEqual({});
  });

  it("trims whitespace from keys and values", () => {
    expect(parseFrontmatter("---\n pipeline : frontend \n---")).toEqual({
      pipeline: "frontend",
    });
  });
});

describe("sectionLines", () => {
  const doc = [
    "# Role",
    "",
    "## Skills",
    "- `/component-review` does thing",
    "- `/accessibility-check` other thing",
    "",
    "## Plugins",
    "- `browser` a plugin",
    "",
    "## Other",
    "ignored",
  ].join("\n");

  it("returns lines within the named section", () => {
    const lines = sectionLines(doc, "Skills");
    expect(lines).toContain("- `/component-review` does thing");
    expect(lines).toContain("- `/accessibility-check` other thing");
  });

  it("stops at the next ## heading", () => {
    const lines = sectionLines(doc, "Skills");
    expect(lines.join("\n")).not.toContain("browser");
  });

  it("returns empty array when section does not exist", () => {
    expect(sectionLines(doc, "Nonexistent")).toEqual([]);
  });

  it("works for a section at end of file with no following heading", () => {
    const lines = sectionLines(doc, "Other");
    expect(lines).toContain("ignored");
  });
});

// --- validate (file-system) ---

let tmpDir: string;
let origCwd: string;

function write(rel: string, content: string): void {
  const full = join(tmpDir, rel);
  mkdirSync(full.slice(0, full.lastIndexOf("/")), { recursive: true });
  writeFileSync(full, content);
}

function agentsJson(overrides: object = {}): string {
  return JSON.stringify({
    workers: [{ id: "bob", role: "testrole" }],
    pipelines: [{ name: "testpipe", stages: [{ name: "build", role: "testrole" }] }],
    ...overrides,
  });
}

const ROLE_CONTENT = "# Test Role\n\n## Skills\n\n## Plugins\n";
const ROLE_WITH_SKILL = "# Test Role\n\n## Skills\n- `/test-skill` does thing\n\n## Plugins\n";
const ROLE_WITH_PLUGIN = "# Test Role\n\n## Skills\n\n## Plugins\n- `some-plugin` a plugin\n";

beforeEach(() => {
  origCwd = process.cwd();
  tmpDir = mkdtempSync(join(tmpdir(), "cli-test-"));
  process.chdir(tmpDir);
});

afterEach(() => {
  process.chdir(origCwd);
  rmSync(tmpDir, { recursive: true, force: true });
});

describe("validate: config file", () => {
  it("fails when agents.json is missing", async () => {
    expect(await validate(["--config=.claude/agents.json"])).toBe(false);
  });

  it("fails when agents.json is invalid JSON", async () => {
    write(".claude/agents.json", "{ bad json }");
    expect(await validate([])).toBe(false);
  });

  it("fails when workers array is empty", async () => {
    write(".claude/agents.json", JSON.stringify({ workers: [], pipelines: [] }));
    expect(await validate([])).toBe(false);
  });
});

describe("validate: workers", () => {
  it("fails when role file is missing", async () => {
    write(".claude/agents.json", agentsJson());
    expect(await validate([])).toBe(false);
  });

  it("passes when role file exists in .claude/roles/", async () => {
    write(".claude/agents.json", agentsJson());
    write(".claude/roles/testrole.md", ROLE_CONTENT);
    expect(await validate([])).toBe(true);
  });

  it("fails when a skill listed in the role file is missing", async () => {
    write(".claude/agents.json", agentsJson());
    write(".claude/roles/testrole.md", ROLE_WITH_SKILL);
    // test-skill not created — should fail
    expect(await validate([])).toBe(false);
  });

  it("passes when a skill listed in the role file exists", async () => {
    write(".claude/agents.json", agentsJson());
    write(".claude/roles/testrole.md", ROLE_WITH_SKILL);
    write(".claude/skills/test-skill.md", "# Test Skill\n");
    expect(await validate([])).toBe(true);
  });

  it("passes with a warning when a plugin is listed", async () => {
    write(".claude/agents.json", agentsJson());
    write(".claude/roles/testrole.md", ROLE_WITH_PLUGIN);
    expect(await validate([])).toBe(true);
  });
});

describe("validate: pipelines", () => {
  it("passes with a warning when no pipelines are defined", async () => {
    write(".claude/agents.json", JSON.stringify({
      workers: [{ id: "bob", role: "testrole" }],
      pipelines: [],
    }));
    write(".claude/roles/testrole.md", ROLE_CONTENT);
    expect(await validate([])).toBe(true);
  });

  it("fails when a pipeline stage references a missing role", async () => {
    write(".claude/agents.json", JSON.stringify({
      workers: [{ id: "bob", role: "testrole" }],
      pipelines: [{ name: "p", stages: [{ name: "build", role: "missingrole" }] }],
    }));
    write(".claude/roles/testrole.md", ROLE_CONTENT);
    expect(await validate([])).toBe(false);
  });
});

describe("validate: --job", () => {
  beforeEach(() => {
    write(".claude/agents.json", agentsJson());
    write(".claude/roles/testrole.md", ROLE_CONTENT);
  });

  it("fails when job file does not exist", async () => {
    expect(await validate(["--job=missing-job"])).toBe(false);
  });

  it("fails when job is already in done/", async () => {
    write(".claude/jobs/done/my-feature.md", "---\npipeline: testpipe\ndomain: auth\n---\n");
    expect(await validate(["--job=my-feature"])).toBe(false);
  });

  it("fails when pipeline frontmatter is missing", async () => {
    write(".claude/jobs/my-feature.md", "---\ndomain: auth\n---\n# Feature\n");
    expect(await validate(["--job=my-feature"])).toBe(false);
  });

  it("fails when domain frontmatter is missing", async () => {
    write(".claude/jobs/my-feature.md", "---\npipeline: testpipe\n---\n# Feature\n");
    expect(await validate(["--job=my-feature"])).toBe(false);
  });

  it("fails when the referenced pipeline is not in agents.json", async () => {
    write(".claude/jobs/my-feature.md", "---\npipeline: nopipe\ndomain: auth\n---\n");
    expect(await validate(["--job=my-feature"])).toBe(false);
  });

  it("passes with valid job file and matching pipeline", async () => {
    write(".claude/jobs/my-feature.md", "---\npipeline: testpipe\ndomain: auth\n---\n# Feature\n");
    expect(await validate(["--job=my-feature"])).toBe(true);
  });
});
