import { Button } from "@heroui/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, Download } from "lucide-react";

export const Route = createFileRoute("/guide")({ component: Guide });

// 各平台二进制由后端 /api/cli 分发（build-cli.sh 交叉编译产出）
const PLATFORMS: { label: string; file: string }[] = [
  { label: "Windows (x64)", file: "anypaste-windows-amd64.exe" },
  { label: "Windows (ARM64)", file: "anypaste-windows-arm64.exe" },
  { label: "macOS (Apple Silicon)", file: "anypaste-darwin-arm64" },
  { label: "macOS (Intel)", file: "anypaste-darwin-amd64" },
  { label: "Linux (x64)", file: "anypaste-linux-amd64" },
  { label: "Linux (ARM64)", file: "anypaste-linux-arm64" },
];

const COMMANDS: { cmd: string; desc: string }[] = [
  {
    cmd: "anypaste login --server <URL>",
    desc: "Authenticate and store the token locally. You will be prompted for the password (or pass --password / set ANYPASTE_PASSWORD).",
  },
  {
    cmd: "anypaste ls",
    desc: "List your pastes: id, type (text/file), and a content preview or file name.",
  },
  {
    cmd: 'anypaste up -m "some text"',
    desc: "Create a text paste. Use -m - to read the content from stdin.",
  },
  {
    cmd: "anypaste up ./report.pdf",
    desc: "Create a paste and upload a file. Combine with -m to set the text content too.",
  },
  {
    cmd: "anypaste down <id> -o ./out.pdf",
    desc: "Download the file of a paste by id. Without -o the server-provided file name is used. Downloading only needs the id.",
  },
  {
    cmd: "anypaste logout",
    desc: "Forget the stored token.",
  },
];

function CodeBlock(props: { children: string }) {
  return (
    <pre className="overflow-x-auto rounded-md bg-slate-900 p-4 text-sm leading-relaxed text-slate-100">
      <code>{props.children}</code>
    </pre>
  );
}

function Guide() {
  // CLI 应指向 Web 应用同一个 API 根（origin + /api），dev/prod 都能经同一代理到后端
  const apiBase = `${window.location.origin}${import.meta.env.BASE_URL}api`;

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-8 p-8">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">CLI guide</h1>
        <Link to="/">
          <Button variant="secondary" size="sm">
            <ArrowLeft className="h-4 w-4" /> Back
          </Button>
        </Link>
      </div>

      <p className="text-muted">
        <code>anypaste</code> is a tiny, dependency-free command-line client. It
        lets you create pastes, upload and download files, and list your pastes
        straight from a terminal. A single small static binary, no install
        needed.
      </p>

      <p className="text-sm text-muted">
        Prefer the terminal? The same instructions are available as plain text:{" "}
        <code>curl {window.location.origin}/help</code>
      </p>

      {/* Downloads */}
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">1. Download</h2>
        <p className="text-sm text-muted">
          Pick the build for your platform:
        </p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {PLATFORMS.map((p) => (
            <a
              key={p.file}
              href={`/cli/${p.file}`}
              download
              className="flex items-center justify-between gap-2 rounded-md border border-default px-4 py-2.5 text-sm transition-colors hover:bg-accent-soft hover:text-accent-hover"
            >
              <span>{p.label}</span>
              <Download className="h-4 w-4 shrink-0 opacity-70" />
            </a>
          ))}
        </div>
      </section>

      {/* Setup */}
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">2. Make it runnable</h2>
        <p className="text-sm text-muted">
          On macOS / Linux, rename it and mark it executable; on Windows just
          run the <code>.exe</code>.
        </p>
        <CodeBlock>{`# macOS / Linux
mv anypaste-linux-amd64 anypaste
chmod +x anypaste
sudo mv anypaste /usr/local/bin/   # optional: put it on your PATH`}</CodeBlock>
      </section>

      {/* Login */}
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">3. Log in</h2>
        <p className="text-sm text-muted">
          Authenticate once with the same password you use here. The token is
          stored in your user config directory.
        </p>
        <CodeBlock>{`anypaste login --server ${apiBase}`}</CodeBlock>
      </section>

      {/* Commands */}
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">4. Commands</h2>
        <div className="flex flex-col divide-y divide-default rounded-md border border-default">
          {COMMANDS.map((c) => (
            <div key={c.cmd} className="flex flex-col gap-1 p-4">
              <code className="text-sm font-semibold text-accent-hover">
                {c.cmd}
              </code>
              <span className="text-sm text-muted">{c.desc}</span>
            </div>
          ))}
        </div>
        <p className="text-sm text-muted">
          Run <code>anypaste help</code> for the full usage text.
        </p>
      </section>
    </div>
  );
}
