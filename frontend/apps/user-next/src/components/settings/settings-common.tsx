"use client";

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@workspace/ui/components/alert";
import { Button } from "@workspace/ui/components/button";
import { CircleAlert } from "lucide-react";

export function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback;
}

export function QueryFailure({
  message,
  retry,
}: Readonly<{ message: string; retry: () => void }>) {
  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>{message}</AlertTitle>
      <AlertDescription>
        <Button className="h-auto px-0 py-0" onClick={retry} variant="link">
          重新加载
        </Button>
      </AlertDescription>
    </Alert>
  );
}
