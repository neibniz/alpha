import { Input } from "@workspace/ui/components/input";
import { Label } from "@workspace/ui/components/label";
import { cn } from "@workspace/ui/lib/utils";
import { Upload } from "lucide-react";
import { useState } from "react";

type ReturnType = "base64" | "file";

interface UploadImageProps {
  onChange: (value: string | File) => void;
  returnType?: ReturnType;
  id?: string;
  children?: React.ReactNode;
  className?: string;
  maxSize?: number; // Maximum file size in MB
}

export const UploadImage = ({
  onChange,
  returnType = "base64",
  id = "image-upload",
  children,
  className,
  maxSize = 1,
}: UploadImageProps) => {
  const [isDragging, setIsDragging] = useState(false);

  const toBase64 = (file: File): Promise<string> =>
    new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => resolve(reader.result as string);
      reader.onerror = (error) => reject(error);
    });

  const validateFileSize = (file: File): boolean => {
    const maxSizeInBytes = maxSize * 1024 * 1024;
    if (file.size > maxSizeInBytes) {
      alert(`File size exceeds the limit (${maxSize}MB)`);
      return false;
    }
    return true;
  };

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      if (!validateFileSize(file)) return;

      if (returnType === "base64") {
        const base64 = await toBase64(file);
        onChange(base64);
      } else {
        onChange(file);
      }
    } catch (error) {
      console.error(error);
    }
  };

  const handleDragOver = (e: React.DragEvent<HTMLLabelElement>) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent<HTMLLabelElement>) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = async (e: React.DragEvent<HTMLLabelElement>) => {
    e.preventDefault();
    setIsDragging(false);

    const file = e.dataTransfer.files?.[0];
    if (!file) return;

    try {
      if (!validateFileSize(file)) return;

      if (returnType === "base64") {
        const base64 = await toBase64(file);
        onChange(base64);
      } else {
        onChange(file);
      }
    } catch (error) {
      console.error(error);
    }
  };

  return (
    <>
      <Input
        accept="image/*"
        className="hidden"
        id={id}
        onChange={handleImageUpload}
        type="file"
      />
      <Label
        className={cn(
          "cursor-pointer",
          !children &&
            "flex items-center justify-center rounded-lg border-2 border-dashed p-4",
          isDragging && "border-primary bg-muted/50",
          className
        )}
        htmlFor={id}
        onDragLeave={handleDragLeave}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        {children || <Upload />}
      </Label>
    </>
  );
};
