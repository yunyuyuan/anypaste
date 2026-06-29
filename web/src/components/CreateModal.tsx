import { createPaste } from '#/gen/paste/v1/paste-PasteService_connectquery';
import { getToken, parseApiPath, queryClient } from '#/req';
import { useMutation } from '@connectrpc/connect-query';
import { Modal, Button, Label, TextField, TextArea, ProgressBar, toast } from '@heroui/react';
import { Loader2, Plus } from 'lucide-react';
import { useRef, useState } from 'react';

// 分片续传协议常量，与 internal/uploadproto 保持一致
const CHUNK_SIZE = 5 * 1024 * 1024; // 5 MiB
const H_OFFSET = "Upload-Offset";
const H_LENGTH = "Upload-Length";
const H_FILENAME = "Upload-Filename";
const MAX_CHUNK_RETRIES = 5;

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

const authHeaders = (): Record<string, string> => {
  const token = getToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
};

// uploadOffset：HEAD 查询服务器已收到的字节数，作为（续传）起点
const uploadOffset = async (id: string): Promise<number> => {
  const res = await fetch(parseApiPath(`/file/upload/${id}`), {
    method: "HEAD",
    headers: authHeaders(),
  });
  if (!res.ok) {
    throw new Error((await res.text()).trim() || `Resume check failed (${res.status})`);
  }
  return Number(res.headers.get(H_OFFSET) ?? 0);
};

// postChunk：用 XHR 发送单个分片（以便监听进度），返回服务器的新偏移量
const postChunk = (
  id: string,
  blob: Blob,
  offset: number,
  total: number,
  filename: string,
  onProgress: (loaded: number) => void,
): Promise<number> =>
  new Promise<number>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", parseApiPath(`/file/upload/${id}`));
    const token = getToken();
    if (token) {
      xhr.setRequestHeader("Authorization", `Bearer ${token}`);
    }
    xhr.setRequestHeader(H_OFFSET, String(offset));
    xhr.setRequestHeader(H_LENGTH, String(total));
    xhr.setRequestHeader(H_FILENAME, encodeURIComponent(filename));
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) {
        onProgress(event.loaded);
      }
    };
    xhr.onload = () =>
      xhr.status >= 200 && xhr.status < 300
        ? resolve(Number(xhr.getResponseHeader(H_OFFSET) ?? offset + blob.size))
        : reject(
          new Error(
            // 服务器用 http.Error 把真实原因写在 body 里，优先展示它
            xhr.responseText.trim() ||
            xhr.statusText ||
            `Upload failed (${xhr.status})`,
          ),
        );
    xhr.onerror = () => reject(new Error("Network error"));
    xhr.send(blob);
  });

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

  // 分片续传：HEAD 取已传偏移量后逐块 POST，可抵御 CF 这类代理的体积/超时限制，断点续传
  const uploadFile = async (id: string, theFile: File) => {
    const total = theFile.size;
    let offset = await uploadOffset(id);
    setUploadProgress(total > 0 ? Math.round((offset / total) * 100) : 0);

    let attempts = 0;
    while (offset < total) {
      const base = offset;
      const end = Math.min(base + CHUNK_SIZE, total);
      try {
        offset = await postChunk(id, theFile.slice(base, end), base, total, theFile.name, (loaded) =>
          setUploadProgress(Math.round(((base + loaded) / total) * 100)),
        );
        attempts = 0;
        setUploadProgress(Math.round((offset / total) * 100));
      } catch (err) {
        // 单块失败：有限次重试，每次先向服务器重新对齐偏移量（覆盖 409/半包）
        if (++attempts > MAX_CHUNK_RETRIES) throw err;
        await sleep(attempts * 500);
        offset = await uploadOffset(id).catch(() => base);
      }
    }
    // 空文件也要发一次请求来创建并 finalize
    if (total === 0) {
      await postChunk(id, theFile.slice(0, 0), 0, 0, theFile.name, () => {});
    }
  };

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
