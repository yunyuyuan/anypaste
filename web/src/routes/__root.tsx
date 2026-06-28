import { Outlet, createRootRoute } from '@tanstack/react-router';
import '../styles.css';
import { QueryClientProvider } from '@tanstack/react-query';
import { getStatus, hasToken, queryClient, rpcTransport } from '#/req';
import { TransportProvider } from '@connectrpc/connect-query';
import { ToastProvider } from '@heroui/react';
import { useEffect, useState } from 'react';
import LoginScreen from '#/components/LoginScreen';
import InitScreen from '#/components/InitScreen';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  // 有 cookie 才放行；token 过期时 RPC 会 401，queryClient 会统一清 cookie 并刷新回到此处
  const [authed, setAuthed] = useState(hasToken);
  // 未登录时需先问后端是否已初始化（设过管理员密码）：null=未知，决定显示设置页还是登录页
  const [initialized, setInitialized] = useState<boolean | null>(null);

  useEffect(() => {
    if (authed) return;
    getStatus()
      .then((s) => setInitialized(s.initialized))
      .catch(() => setInitialized(true)); // 查询失败时退回登录页
  }, [authed]);

  let screen;
  if (authed) {
    screen = <Outlet />;
  } else if (initialized === null) {
    screen = null; // 等待状态查询
  } else if (!initialized) {
    screen = <InitScreen onDone={() => setAuthed(true)} />;
  } else {
    screen = <LoginScreen onSuccess={() => setAuthed(true)} />;
  }

  return (
    <>
      <TransportProvider transport={rpcTransport}>
        <QueryClientProvider client={queryClient}>
          {screen}
        </QueryClientProvider>
      </TransportProvider>
      <ToastProvider placement="top" />
      {/* <TanStackDevtools
        config={{
          position: 'bottom-right',
        }}
        plugins={[
          {
            name: 'TanStack Router',
            render: <TanStackRouterDevtoolsPanel />,
          },
        ]}
      /> */}
    </>
  );
}
