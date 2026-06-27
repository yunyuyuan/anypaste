import CreateModal from "#/components/CreateModal";
import { clearToken } from "#/req";
import { Button } from "@heroui/react";
import { useNavigate } from "@tanstack/react-router";
import { LogOut, Terminal } from "lucide-react";

// 顶部统一操作栏：左侧标题 + 新建，右侧 CLI 指南与登出
export default function Header() {
  const navigate = useNavigate();

  const logout = () => {
    // 清掉 cookie 后刷新，回到登录门（与 401 处理一致）
    clearToken();
    window.location.reload();
  };

  return (
    <header className="flex items-center justify-between gap-4 border-b border-default px-8 py-3">
      <div className="flex items-center gap-4">
        <a href={import.meta.env.BASE_URL} className="flex items-center gap-2">
          <img
            src={`${import.meta.env.BASE_URL}logo192.png`}
            alt="AnyPaste"
            className="h-10 w-10"
          />
          <span className="text-lg font-semibold">AnyPaste</span>
        </a>
        <CreateModal />
      </div>
      <nav className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => navigate({ to: "/guide" })}
        >
          <Terminal className="h-4 w-4" /> CLI
        </Button>
        <Button variant="danger-soft" size="sm" onClick={logout}>
          <LogOut className="h-4 w-4" /> Logout
        </Button>
      </nav>
    </header>
  );
}
