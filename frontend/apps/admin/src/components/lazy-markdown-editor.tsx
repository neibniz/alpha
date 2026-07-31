import { Skeleton } from "@workspace/ui/components/skeleton";
import type { MonacoEditorProps } from "@workspace/ui/composed/editor/monaco-editor";
import { lazy, Suspense } from "react";

const MarkdownEditor = lazy(() =>
  import("@workspace/ui/composed/editor/markdown").then((module) => ({
    default: module.MarkdownEditor,
  }))
);

export function LazyMarkdownEditor(props: MonacoEditorProps) {
  return (
    <Suspense
      fallback={
        <div aria-busy="true" className="space-y-2">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-72 w-full" />
        </div>
      }
    >
      <MarkdownEditor {...props} />
    </Suspense>
  );
}
