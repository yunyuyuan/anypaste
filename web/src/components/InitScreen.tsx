import { initPassword } from "#/req";
import { Button, Input, Label, TextField, toast } from "@heroui/react";
import { Loader2 } from "lucide-react";
import { useState } from "react";

// 首次运行：还没有管理员密码时显示，让用户设置一个 → 写入配置文件 → 直接登录
export default function InitScreen(props: { onDone: () => void }) {
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pending, setPending] = useState(false);

  const mismatch = confirm.length > 0 && confirm !== password;
  const canSubmit = password.length > 0 && password === confirm;

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (pending || !canSubmit) return;
    setPending(true);
    try {
      await initPassword(password);
      props.onDone();
    } catch (err) {
      toast.danger("Setup failed", {
        description: err instanceof Error ? err.message : String(err),
      });
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <form
        onSubmit={submit}
        className="flex w-80 flex-col gap-4 rounded-lg border border-default p-6 shadow-md"
      >
        <div className="flex flex-col gap-1">
          <h1 className="text-lg font-semibold">Welcome to AnyPaste</h1>
          <p className="text-sm text-muted">
            Set an admin password to finish setup.
          </p>
        </div>
        <TextField
          name="password"
          type="password"
          value={password}
          onChange={setPassword}
          autoFocus
          isRequired
        >
          <Label>Password</Label>
          <Input placeholder="choose a password" />
        </TextField>
        <TextField
          name="confirm"
          type="password"
          value={confirm}
          onChange={setConfirm}
          isRequired
          isInvalid={mismatch}
        >
          <Label>Confirm password</Label>
          <Input placeholder="repeat the password" />
          {mismatch ? (
            <span className="text-sm text-danger">Passwords do not match</span>
          ) : null}
        </TextField>
        <Button type="submit" isPending={pending} isDisabled={!canSubmit}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          Create password
        </Button>
      </form>
    </div>
  );
}
