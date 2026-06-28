import { deletePaste } from "#/gen/paste/v1/paste-PasteService_connectquery";
import type { PasteItem } from "#/gen/paste/v1/paste_pb";
import { queryClient } from "#/req";
import { useMutation } from "@connectrpc/connect-query";
import { Button, Card, Tooltip, toast } from "@heroui/react";
import { Copy, Download, FileText, Trash2 } from "lucide-react";
import { useState } from "react";
import PasteDetailModal, { copyText, downloadFile } from "./PasteDetailModal";

type OverlayAction = {
  key: string;
  icon: typeof Copy;
  label: string;
  onAction: () => void;
  className?: string;
};

// 没有文件时：整张卡片做成一张便签
function NoteFace(props: { content: string }) {
  return (
    <div className="absolute inset-0 flex flex-col bg-linear-to-br from-yellow-100 via-yellow-100 to-amber-200">
      {/* 顶部胶带 */}
      <div className="absolute left-1/2 top-0 h-5 w-20 -translate-x-1/2 -translate-y-1/2 -rotate-3 rounded-sm bg-white/50 shadow-sm" />
      {/* 作业本式横向虚线（横线 + 竖向虚线蒙版），淡淡的不显眼 */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          backgroundImage:
            "repeating-linear-gradient(to bottom, transparent 0, transparent 25px, rgba(120,53,15,0.12) 25px, rgba(120,53,15,0.12) 26px)",
          WebkitMaskImage:
            "repeating-linear-gradient(to right, black 0 5px, transparent 5px 9px)",
          maskImage:
            "repeating-linear-gradient(to right, black 0 5px, transparent 5px 9px)",
        }}
      />
      <div className="relative flex flex-1 items-center px-5 mb-5 mt-7">
        <p className="line-clamp-8 w-full whitespace-pre-wrap wrap-break-word text-[15px] leading-relaxed text-amber-950/90">
          {props.content}
        </p>
      </div>
      {/* 右下角卷起的折角 */}
      <div className="absolute bottom-0 right-0 h-7 w-7 bg-amber-300/80 shadow-[-2px_-2px_4px_rgba(0,0,0,0.12)] [clip-path:polygon(100%_0,0%_100%,100%_100%)]" />
    </div>
  );
}

// 有文件时：整张卡片做成一份文件
function FileFace(props: { ext: string; content: string }) {
  return (
    <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-linear-to-b from-slate-50 to-slate-200">
      {/* 右上角折角 */}
      <div className="absolute right-0 top-0 h-9 w-9 bg-slate-300 shadow-[-2px_2px_5px_rgba(0,0,0,0.1)] [clip-path:polygon(0_0,100%_0,100%_100%)]" />
      <FileText className="h-16 w-16 text-slate-400" strokeWidth={1} />
      <span className="rounded-md bg-slate-700 px-2.5 py-1 text-xs font-bold uppercase tracking-widest text-white shadow-sm">
        {props.ext}
      </span>
      {/* 弱化展示的内容预览 */}
      {props.content ? (
        <p className="line-clamp-2 max-w-[85%] whitespace-pre-wrap wrap-break-word text-center text-xs leading-snug text-slate-400">
          {props.content}
        </p>
      ) : null}
    </div>
  );
}

function HoverOverlay(props: { actions: OverlayAction[] }) {
  return (
    <div
      className="absolute inset-0 z-10 hidden items-center justify-center gap-3
                 bg-white/30 backdrop-blur-sm opacity-0 transition-opacity
                 duration-200 group-hover:opacity-100 sm:flex"
    >
      {props.actions.map((action) => (
        <Tooltip key={action.key} delay={300}>
          <Button
            isIconOnly
            variant="outline"
            size="lg"
            aria-label={action.label}
            className={action.className}
            onClick={(e) => {
              e.stopPropagation();
              action.onAction();
            }}
          >
            <action.icon />
          </Button>
          <Tooltip.Content>{action.label}</Tooltip.Content>
        </Tooltip>
      ))}
    </div>
  );
}

export default function (props: { item: PasteItem }) {
  const item = props.item;
  const [detailOpen, setDetailOpen] = useState(false);
  const { mutateAsync: deleteItem } = useMutation(deletePaste);

  const remove = async () => {
    await deleteItem({ id: item.id });
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "ListPastes" }],
    });
  };

  const actions: OverlayAction[] = [
    {
      key: "copy",
      icon: Copy,
      label: "Copy",
      onAction: () => copyText(item.content),
    },
    // 有文件时才显示下载
    ...(item.fileName
      ? [
        {
          key: "download",
          icon: Download,
          label: "Download",
          onAction: () => downloadFile(item),
        } satisfies OverlayAction,
      ]
      : []),
    {
      key: "delete",
      icon: Trash2,
      label: "Delete",
      className: "text-danger-hover",
      onAction: () =>
        toast.promise(remove(), {
          error: "Failed to delete",
          loading: "Deleting...",
          success: "Deleted",
        }),
    },
  ];

  const ext = item.fileName?.includes(".")
    ? item.fileName.split(".").pop()!.toLowerCase()
    : "file";

  return (
    <>
      <Card
        className="group relative w-60 h-60 cursor-pointer overflow-hidden p-0 shadow-md transition-shadow hover:shadow-lg"
        onClick={() => setDetailOpen(true)}
      >
        <HoverOverlay actions={actions} />
        {item.fileName ? (
          <FileFace ext={ext} content={item.content} />
        ) : (
          <NoteFace content={item.content} />
        )}
      </Card>

      <PasteDetailModal
        item={item}
        isOpen={detailOpen}
        onOpenChange={setDetailOpen}
      />
    </>
  );
}
