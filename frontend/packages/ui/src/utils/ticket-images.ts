export const MAX_TICKET_IMAGES = 6;
export const MAX_TICKET_IMAGE_DATA_URL_LENGTH = 1_400_000;
export const MAX_TICKET_SOURCE_IMAGE_BYTES = 12 * 1024 * 1024;

const ACCEPTED_IMAGE_TYPES = new Set([
  "image/gif",
  "image/jpeg",
  "image/png",
  "image/webp",
]);

export function availableTicketImageSlots(currentCount: number) {
  return Math.max(0, MAX_TICKET_IMAGES - Math.max(0, currentCount));
}

export function mergeTicketImages(
  current: readonly string[],
  incoming: readonly string[]
) {
  const unique = new Set(current);
  for (const image of incoming) {
    if (unique.size >= MAX_TICKET_IMAGES) break;
    if (image) unique.add(image);
  }
  return [...unique].slice(0, MAX_TICKET_IMAGES);
}

export async function prepareTicketImages(
  files: readonly File[],
  currentCount: number
) {
  const slots = availableTicketImageSlots(currentCount);
  if (slots === 0) throw new Error("最多添加 6 张图片");
  const selected = files.slice(0, slots);
  if (selected.some((file) => !ACCEPTED_IMAGE_TYPES.has(file.type))) {
    throw new Error("仅支持 PNG、JPEG、WebP 或 GIF 图片");
  }
  if (selected.some((file) => file.size > MAX_TICKET_SOURCE_IMAGE_BYTES)) {
    throw new Error("单张原图不能超过 12 MB");
  }
  return Promise.all(selected.map(compressTicketImage));
}

async function compressTicketImage(file: File) {
  const image = await loadImage(file);
  const dimensions = [1600, 1280, 1024, 800];
  const qualities = [0.86, 0.78, 0.7, 0.62];

  for (const maxDimension of dimensions) {
    const scale = Math.min(
      1,
      maxDimension / Math.max(image.naturalWidth, image.naturalHeight)
    );
    const width = Math.max(1, Math.round(image.naturalWidth * scale));
    const height = Math.max(1, Math.round(image.naturalHeight * scale));
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("浏览器无法处理图片");
    context.drawImage(image, 0, 0, width, height);
    for (const quality of qualities) {
      const result = canvas.toDataURL("image/webp", quality);
      if (result.length <= MAX_TICKET_IMAGE_DATA_URL_LENGTH) return result;
    }
  }
  throw new Error("图片压缩后仍然过大");
}

function loadImage(file: File) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const objectURL = URL.createObjectURL(file);
    const image = new Image();
    image.onload = () => {
      URL.revokeObjectURL(objectURL);
      resolve(image);
    };
    image.onerror = () => {
      URL.revokeObjectURL(objectURL);
      reject(new Error("图片读取失败"));
    };
    image.src = objectURL;
  });
}
