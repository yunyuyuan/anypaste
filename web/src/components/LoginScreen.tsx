import { login } from "#/req";
import { Button, Input, Label, TextField, toast } from "@heroui/react";
import { Loader2 } from "lucide-react";
import { useState } from "react";

// 未登录时的拦截页：输入密码 → 调 /login 拿 token 写入 cookie → onSuccess 放行
export default function LoginScreen(props: { onSuccess: () => void }) {
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (pending || !password) return;
    setPending(true);
    try {
      await login(password);
      props.onSuccess();
    } catch (err) {
      toast.danger("Login failed", {
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
        <h1 className="text-lg font-semibold">Sign in</h1>
        <TextField
          name="password"
          type="password"
          value={password}
          onChange={setPassword}
          autoFocus
          isRequired
        >
          <Label>Password</Label>
          <Input placeholder="please input password" />
        </TextField>
        <Button type="submit" isPending={pending} isDisabled={!password}>
          {pending ? <Loader2 className="animate-spin" /> : null}
          Sign in
        </Button>
      </form>
    </div>
  );
}
