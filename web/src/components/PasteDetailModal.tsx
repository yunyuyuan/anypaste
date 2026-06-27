import { updatePaste } from "#/gen/paste/v1/paste-PasteService_connectquery";
import type { PasteItem } from "#/gen/paste/v1/paste_pb";
import { parseApiPath, queryClient } from "#/req";
import { useMutation } from "@connectrpc/connect-query";
import {
  Button,
  Modal,
  TextArea,
  TextField,
  Tooltip,
  toast,
} from "@heroui/react";
import { Copy, Download, FileText, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";

export function copyText(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    toast("copied");
  });
}

// 直接导航到下载地址，交给浏览器自带的下载器（支持断点续传/重试）
export function downloadFile(item: PasteItem) {
  const a = document.createElement("a");
  a.href = parseApiPath(`/file/download/${item.id}`);
  a.download = item.fileName ?? "";
  a.click();
}

// 详情/编辑弹窗：可改内容，保存调用 UpdatePaste
export default function PasteDetailModal(props: {
  item: PasteItem;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { item, isOpen, onOpenChange } = props;
  const [draft, setDraft] = useState(item.content);
  const { mutateAsync: updateItem, isPending: isSaving } =
    useMutation(updatePaste);

  // 每次打开弹窗时，用最新内容重置编辑草稿
  useEffect(() => {
    if (isOpen) setDraft(item.content);
  }, [isOpen, item.content]);

  const isDirty = draft !== item.content;

  const save = async () => {
    await updateItem({ id: item.id, content: draft });
    queryClient.invalidateQueries({
      queryKey: ["connect-query", { methodName: "ListPastes" }],
    });
    onOpenChange(false);
  };

  return (
    <Modal isOpen={isOpen} onOpenChange={onOpenChange}>
      <Modal.Backdrop>
        <Modal.Container>
          <Modal.Dialog>
            <Modal.Header>
              <Modal.Heading>Note detail</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <div className="flex flex-col gap-3">
                {item.fileName ? (
                  // 点击文件名即下载，带下划线 + hover 高亮，末尾一个下载图标
                  <Tooltip delay={300}>
                    <Tooltip.Trigger
                      onClick={() => downloadFile(item)}
                      className="group/file flex cursor-pointer items-center gap-2 self-start rounded-md px-2 py-1 text-sm text-muted transition-colors hover:bg-accent-soft hover:text-accent-hover"
                    >
                      <FileText className="h-4 w-4 shrink-0" />
                      <span className="break-all underline underline-offset-2">
                        {item.fileName}
                      </span>
                      <Download className="h-4 w-4 shrink-0 opacity-60 transition-opacity group-hover/file:opacity-100" />
                    </Tooltip.Trigger>
                    <Tooltip.Content>Download</Tooltip.Content>
                  </Tooltip>
                ) : null}
                <TextField name="content">
                  <TextArea
                    rows={8}
                    value={draft}
                    placeholder="please input content"
                    onChange={(event) => setDraft(event.target.value)}
                  />
                </TextField>
              </div>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">
                Close
              </Button>
              <Button onClick={() => copyText(draft)}>
                <Copy /> Copy
              </Button>
              <Button onClick={save} isPending={isSaving} isDisabled={!isDirty}>
                {isSaving ? <Loader2 className="animate-spin" /> : null}
                Save
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  );
}
