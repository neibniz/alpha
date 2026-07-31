import { Skeleton } from "@workspace/ui/components/skeleton";
import type { MonacoEditorProps } from "@workspace/ui/composed/editor/monaco-editor";
import { lazy, Suspense } from "react";

const MonacoEditor = lazy(() =>
  import("@workspace/ui/composed/editor/monaco-editor").then((module) => ({
    default: module.MonacoEditor,
  }))
);

export function LazyMonacoEditor(props: MonacoEditorProps) {
  return (
    <Suspense
      fallback={
        <div aria-busy="true" className="space-y-2">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-72 w-full" />
        </div>
      }
    >
      <MonacoEditor {...props} />
    </Suspense>
  );
}
