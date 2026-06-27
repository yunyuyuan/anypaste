import { Outlet, createRootRoute } from '@tanstack/react-router';
import '../styles.css';
import { QueryClientProvider } from '@tanstack/react-query';
import { hasToken, queryClient, rpcTransport } from '#/req';
import { TransportProvider } from '@connectrpc/connect-query';
import { ToastProvider } from '@heroui/react';
import { useState } from 'react';
import LoginScreen from '#/components/LoginScreen';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  // 有 cookie 才放行；token 过期时 RPC 会 401，queryClient 会统一清 cookie 并刷新回到此处
  const [authed, setAuthed] = useState(hasToken);

  return (
    <>
      <TransportProvider transport={rpcTransport}>
        <QueryClientProvider client={queryClient}>
          {authed ? (
            <Outlet />
          ) : (
            <LoginScreen onSuccess={() => setAuthed(true)} />
          )}
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
