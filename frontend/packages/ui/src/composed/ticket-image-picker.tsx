import { Button } from "@workspace/ui/components/button";
import { Input } from "@workspace/ui/components/input";
import { cn } from "@workspace/ui/lib/utils";
import {
  MAX_TICKET_IMAGES,
  mergeTicketImages,
  prepareTicketImages,
} from "@workspace/ui/utils/ticket-images";
import { ImagePlus, LoaderCircle, X } from "lucide-react";
import { useId, useState } from "react";
import { TicketImageGallery } from "./ticket-image-gallery";

interface TicketImagePickerProps {
  images: readonly string[];
  onChange: (images: string[]) => void;
  disabled?: boolean;
  className?: string;
  onError?: (message: string) => void;
  addLabel?: string;
}

export function TicketImagePicker({
  images,
  onChange,
  disabled,
  className,
  onError,
  addLabel = "Add images",
}: TicketImagePickerProps) {
  const inputId = useId();
  const [processing, setProcessing] = useState(false);

  const addFiles = async (files: File[]) => {
    if (files.length === 0 || processing) return;
    setProcessing(true);
    try {
      const prepared = await prepareTicketImages(files, images.length);
      onChange(mergeTicketImages(images, prepared));
    } catch (error) {
      onError?.(error instanceof Error ? error.message : "图片处理失败");
    } finally {
      setProcessing(false);
    }
  };

  return (
    <div className={cn("space-y-3", className)}>
      {images.length > 0 ? (
        <div className="relative">
          <TicketImageGallery images={images} />
          <div className="mt-2 flex flex-wrap gap-2">
            {images.map((image, index) => (
              <Button
                aria-label={`移除图片 ${index + 1}`}
                disabled={disabled || processing}
                key={`${image.slice(-24)}-${index}`}
                onClick={() =>
                  onChange(images.filter((_, itemIndex) => itemIndex !== index))
                }
                size="sm"
                type="button"
                variant="outline"
              >
                <X className="size-3.5" />
                {index + 1}
              </Button>
            ))}
          </div>
        </div>
      ) : null}
      <Input
        accept="image/png,image/jpeg,image/webp,image/gif"
        className="hidden"
        disabled={disabled || processing || images.length >= MAX_TICKET_IMAGES}
        id={inputId}
        multiple
        onChange={async (event) => {
          const files = Array.from(event.target.files || []);
          event.target.value = "";
          await addFiles(files);
        }}
        type="file"
      />
      <Button
        asChild
        className="w-fit"
        disabled={disabled || processing || images.length >= MAX_TICKET_IMAGES}
        size="sm"
        type="button"
        variant="outline"
      >
        <label htmlFor={inputId}>
          {processing ? (
            <LoaderCircle className="size-4 animate-spin" />
          ) : (
            <ImagePlus className="size-4" />
          )}
          {addLabel} {images.length}/{MAX_TICKET_IMAGES}
        </label>
      </Button>
    </div>
  );
}
