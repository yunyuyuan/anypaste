import { createPaste } from '#/gen/paste/v1/paste-PasteService_connectquery';
import { getToken, parseApiPath, queryClient } from '#/req';
import { useMutation } from '@connectrpc/connect-query';
import { Modal, Button, Label, TextField, TextArea, ProgressBar, toast } from '@heroui/react';
import { Loader2, Plus } from 'lucide-react';
import { useRef, useState } from 'react';

export default function () {
  const [isOpen, setIsOpen] = useState(false);

  const [content, setContent] = useState<string>("");
  const [file, setFile] = useState<File>();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  // 字节已全部发出、等待服务器处理响应（此时进度无法再增长，切成不确定态）
  const [processing, setProcessing] = useState(false);

  const { mutate: createItem } = useMutation(createPaste);

  // 用 XMLHttpRequest 上传，以便监听上传进度
  const uploadFile = (id: string, theFile: File) =>
    new Promise<void>((resolve, reject) => {
      const formData = new FormData();
      formData.append("file", theFile);
      const xhr = new XMLHttpRequest();
      xhr.open("POST", parseApiPath(`/file/upload/${id}`));
      // 上传走原生 XHR，需手动带上鉴权头（token 存在 cookie）
      const token = getToken();
      if (token) {
        xhr.setRequestHeader("Authorization", `Bearer ${token}`);
      }
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable) {
          setUploadProgress(Math.round((event.loaded / event.total) * 100));
        }
      };
      // 字节发送完毕，但服务器还在处理：切到不确定态
      xhr.upload.onload = () => setProcessing(true);
      xhr.onload = () =>
        xhr.status >= 200 && xhr.status < 300
          ? resolve()
          : reject(
            new Error(
              // 服务器用 http.Error 把真实原因写在 body 里，优先展示它
              xhr.responseText.trim() ||
              xhr.statusText ||
              `Upload failed (${xhr.status})`,
            ),
          );
      xhr.onerror = () => reject(new Error("Network error"));
      xhr.send(formData);
    });

  // 上传文件，失败时弹 toast 让用户重试（服务器支持重试）
  const runUpload = (id: string, theFile: File, onDone: () => void) => {
    setUploading(true);
    setProcessing(false);
    setUploadProgress(0);
    uploadFile(id, theFile)
      .then(onDone)
      .catch((err: unknown) => {
        toast.danger("Upload failed", {
          description: err instanceof Error ? err.message : String(err),
          actionProps: {
            children: "Retry",
            onPress: () => runUpload(id, theFile, onDone),
          },
        });
      })
      .finally(() => {
        setUploading(false);
        setProcessing(false);
      });
  };

  const onCreate = () => {
    const onSuccess = () => {
      queryClient.invalidateQueries({
        queryKey: ['connect-query', {
          methodName: "ListPastes",
        }]
      });
      setUploading(false);
      setProcessing(false);
      setIsOpen(false);
    };
    // 只要开始调用接口就进入 uploading；创建阶段没有进度，先用不确定态
    setUploading(true);
    setProcessing(true);
    setUploadProgress(0);
    createItem({
      content,
    }, {
      onSuccess: (res) => {
        // 选了文件就走上传，否则是纯文本 paste
        if (res.success && res.id && file) {
          runUpload(res.id, file, onSuccess);
        } else {
          onSuccess();
        }
      },
      onError: (err: unknown) => {
        toast.danger("Create failed", {
          description: err instanceof Error ? err.message : String(err),
        });
        setUploading(false);
        setProcessing(false);
      },
    });
  };

  return (
    <Modal isOpen={isOpen} onOpenChange={setIsOpen}>
      <Button>
        <Plus />
        New
      </Button>
      <Modal.Backdrop isDismissable={false}>
        <Modal.Container>
          <Modal.Dialog>
            <Modal.Header>
              <Modal.Heading>New Paste</Modal.Heading>
            </Modal.Header>
            <Modal.Body>
              <div className="flex flex-col gap-2">
                <TextField isRequired name="content">
                  <Label>Content</Label>
                  <TextArea placeholder="please input content" rows={4} value={content} onChange={event => setContent(event.target.value)} />
                </TextField>
                <div className="flex flex-col gap-1">
                  <Label>File (optional)</Label>
                  <input
                    ref={fileInputRef}
                    type="file"
                    className="hidden"
                    onChange={event => setFile(event.target.files?.[0])}
                  />
                  <div className="flex items-center gap-2">
                    <Button
                      variant="secondary"
                      onClick={() => fileInputRef.current?.click()}
                      className="m-1.5"
                    >
                      Choose file
                    </Button>
                    <span className="text-sm opacity-70 truncate">
                      {file ? file.name : "No file chosen"}
                    </span>
                  </div>
                </div>
                {uploading ? (
                  <ProgressBar
                    aria-label="Upload progress"
                    value={processing ? undefined : uploadProgress}
                    isIndeterminate={processing}
                    className="flex flex-col gap-1"
                  >
                    <div className="flex justify-between text-sm">
                      <Label>{processing ? "Processing…" : "Uploading…"}</Label>
                      <ProgressBar.Output />
                    </div>
                    <ProgressBar.Track>
                      <ProgressBar.Fill />
                    </ProgressBar.Track>
                  </ProgressBar>
                ) : null}
              </div>
            </Modal.Body>
            <Modal.Footer>
              <Button slot="close" variant="secondary">
                Cancel
              </Button>
              <Button onClick={onCreate} isPending={uploading}>
                {uploading ? <Loader2 className="animate-spin" /> : null}
                Confirm
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  );
}
