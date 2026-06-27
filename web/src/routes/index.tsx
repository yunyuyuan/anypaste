import Header from '#/components/Header';
import PasteCard from '#/components/PasteCard';
import { listPastes } from '#/gen/paste/v1/paste-PasteService_connectquery';
import { queryClient, rpcTransport } from '#/req';
import { TransportProvider, useQuery } from '@connectrpc/connect-query';
import { QueryClientProvider } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/')({ component: Home });

function Home() {
  const { data } = useQuery(listPastes);

  return (
    <TransportProvider transport={rpcTransport}>
      <QueryClientProvider client={queryClient}>
        <Header />
        <div className="flex flex-wrap gap-12 p-8">
          {data?.list.map((i) => (
            <PasteCard key={i.id} item={i} />
          ))}
        </div>
      </QueryClientProvider>
    </TransportProvider>
  );
}
