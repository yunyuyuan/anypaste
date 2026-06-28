import { Code, ConnectError  } from "@connectrpc/connect";
import type {Interceptor} from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { QueryCache, QueryClient } from "@tanstack/react-query";

const API_BASE = `${import.meta.env.BASE_URL}api`;

const TOKEN_COOKIE = "anypaste_token";

export const getToken = (): string | null => {
  const m = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${TOKEN_COOKIE}=([^;]+)`),
  );
  return m ? decodeURIComponent(m[1]) : null;
};

export const hasToken = (): boolean => getToken() !== null;

export const setToken = (token: string) => {
  // 与后端 tokenTTL 一致：7 天。SameSite=Lax；非 HttpOnly —— 前端要读出来塞进 Authorization 头
  const maxAge = 7 * 24 * 60 * 60;
  document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=${maxAge}; samesite=lax`;
};

export const clearToken = () => {
  document.cookie = `${TOKEN_COOKIE}=; path=/; max-age=0`;
};

// 每个 RPC 自动带上 Authorization 头（token 存在 cookie 里）
const authInterceptor: Interceptor = (next) => (req) => {
  const token = getToken();
  if (token) {
    req.header.set("Authorization", `Bearer ${token}`);
  }
  return next(req);
};

export const rpcTransport = createConnectTransport({
  baseUrl: API_BASE,
  interceptors: [authInterceptor],
});

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (err) => {
      // token 缺失/失效：清掉 cookie 并刷新，回到登录页（先清再刷新，避免死循环）
      if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
        clearToken();
        window.location.reload();
      }
    },
  }),
});

export const parseApiPath = (path: string) => {
  return `${API_BASE}/${path.replace(/^\//, "")}`;
};

// 首次运行检查：后端是否已设置管理员密码。未初始化则前端跳设置页。
export const getStatus = async (): Promise<{ initialized: boolean }> => {
  const res = await fetch(parseApiPath("/status"));
  if (!res.ok) throw new Error("status check failed");
  return res.json();
};

// 初始化：设置管理员密码，后端写入配置文件并直接下发 token（即登录）。
export const initPassword = async (password: string): Promise<void> => {
  const res = await fetch(parseApiPath("/init"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!res.ok) {
    throw new Error((await res.text()).trim() || "Setup failed");
  }
  const data: { token: string } = await res.json();
  setToken(data.token);
};

// 登录：校验密码，拿到 token 写入 cookie。失败时抛出后端返回的错误信息。
export const login = async (password: string): Promise<void> => {
  const res = await fetch(parseApiPath("/login"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!res.ok) {
    throw new Error((await res.text()).trim() || "Login failed");
  }
  const data: { token: string } = await res.json();
  setToken(data.token);
};
