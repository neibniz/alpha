import { Button } from "@workspace/ui/components/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@workspace/ui/components/dialog";
import { cn } from "@workspace/ui/lib/utils";
import { ImageIcon, ZoomIn } from "lucide-react";
import { useState } from "react";

interface TicketImageGalleryProps {
  images: readonly string[];
  className?: string;
}

export function TicketImageGallery({
  images,
  className,
}: TicketImageGalleryProps) {
  const [preview, setPreview] = useState<string>();
  if (images.length === 0) return null;

  return (
    <>
      <div
        className={cn(
          "grid w-full max-w-xl grid-cols-2 gap-2 sm:grid-cols-3",
          className
        )}
      >
        {images.map((image, index) => (
          <Button
            aria-label={`查看图片 ${index + 1}`}
            className="group relative aspect-[4/3] h-auto overflow-hidden rounded-md border bg-muted p-0"
            key={`${image.slice(-32)}-${index}`}
            onClick={() => setPreview(image)}
            type="button"
            variant="ghost"
          >
            <img
              alt={`工单图片 ${index + 1}`}
              className="size-full object-cover transition-transform duration-200 group-hover:scale-[1.03]"
              height={480}
              src={image}
              width={640}
            />
            <span className="absolute right-2 bottom-2 flex size-7 items-center justify-center rounded-md bg-background/90 opacity-0 shadow-sm transition-opacity group-hover:opacity-100">
              <ZoomIn className="size-4" />
            </span>
          </Button>
        ))}
      </div>
      <Dialog
        onOpenChange={(open) => !open && setPreview(undefined)}
        open={!!preview}
      >
        <DialogContent className="max-h-[92dvh] max-w-5xl overflow-hidden p-2">
          <DialogTitle className="sr-only">查看工单图片</DialogTitle>
          {preview ? (
            <img
              alt="工单图片预览"
              className="max-h-[calc(92dvh-1rem)] w-full rounded-md object-contain"
              height={1200}
              src={preview}
              width={1600}
            />
          ) : (
            <ImageIcon className="m-auto size-8 text-muted-foreground" />
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
