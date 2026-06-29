import { Github } from "lucide-react";
import { useEffect, useState } from "react";

const REPO = "yunyuyuan/anypaste";
const REPO_URL = `https://github.com/${REPO}`;
const RELEASES_URL = `${REPO_URL}/releases`;
const LATEST_API = `https://api.github.com/repos/${REPO}/releases/latest`;

// 缓存最新版本号，避免每次刷新都打 GitHub（未鉴权接口每小时 60 次限流）
const CACHE_KEY = "anypaste_latest_release";
const CACHE_TTL = 6 * 60 * 60 * 1000; // 6h

// 构建时注入的版本号（release 的 git tag）；dev 下为 undefined
const CURRENT = import.meta.env.VITE_APP_VERSION;

// 比较两个 vX.Y.Z：latest 比 current 新返回 true
function isNewer(latest: string, current: string): boolean {
  const nums = (v: string) =>
    v
      .replace(/^v/, "")
      .split(".")
      .map((n) => parseInt(n, 10) || 0);
  const a = nums(latest);
  const b = nums(current);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const x = a[i] ?? 0;
    const y = b[i] ?? 0;
    if (x !== y) return x > y;
  }
  return false;
}

// 取最新 release tag：先看缓存（6h 内直接用），否则调 GitHub 并回写缓存
async function fetchLatest(): Promise<string | null> {
  try {
    const raw = localStorage.getItem(CACHE_KEY);
    if (raw) {
      const { tag, ts } = JSON.parse(raw) as { tag: string; ts: number };
      if (Date.now() - ts < CACHE_TTL) return tag;
    }
  } catch {
    // 缓存损坏：忽略，走网络
  }
  const res = await fetch(LATEST_API);
  if (!res.ok) throw new Error(`github ${res.status}`);
  const data = (await res.json()) as { tag_name?: string };
  const tag = data.tag_name ?? null;
  if (tag) {
    try {
      localStorage.setItem(CACHE_KEY, JSON.stringify({ tag, ts: Date.now() }));
    } catch {
      // 写缓存失败（隐私模式/配额）：无所谓
    }
  }
  return tag;
}

// 页脚：GitHub 链接 + 当前版本，并检查是否有新版本可用
export default function Footer() {
  const [latest, setLatest] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    fetchLatest()
      .then((tag) => {
        if (alive) setLatest(tag);
      })
      .catch(() => {
        // 离线/限流：安静跳过更新检查
      });
    return () => {
      alive = false;
    };
  }, []);

  const updateAvailable = !!(CURRENT && latest && isNewer(latest, CURRENT));

  return (
    <footer className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 border-t border-default px-4 py-3 text-sm opacity-70">
      <a
        href={REPO_URL}
        target="_blank"
        rel="noreferrer"
        className="flex items-center gap-1 hover:opacity-100"
      >
        <Github className="h-4 w-4" />
        {REPO}
      </a>
      {CURRENT ? <span>· {CURRENT}</span> : null}
      {updateAvailable ? (
        <a
          href={RELEASES_URL}
          target="_blank"
          rel="noreferrer"
          className="font-medium text-warning underline hover:opacity-100"
        >
          · Update available: {latest}
        </a>
      ) : null}
    </footer>
  );
}
