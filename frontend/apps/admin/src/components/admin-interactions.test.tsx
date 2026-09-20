/** @vitest-environment jsdom */
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { Form } from "@workspace/ui/components/form";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, test, vi } from "vitest";
import { AdminForm } from "./admin-form";
import { ConfirmButton } from "./confirm-button";
import { useFormCloseGuard } from "./use-form-close-guard";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  }),
}));
afterEach(cleanup);

describe("Admin confirmations", () => {
  test("waits for completion and blocks repeated submissions", async () => {
    let finish!: () => void;
    const onConfirm = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        })
    );
    render(
      <ConfirmButton
        title="Delete record"
        description="This cannot be undone."
        trigger={<button type="button">Delete</button>}
        onConfirm={onConfirm}
      />
    );
    fireEvent.click(screen.getByText("Delete"));
    fireEvent.click(screen.getByText("Confirm"));
    expect(screen.getByRole("alertdialog")).toBeTruthy();
    expect((screen.getByText("Processing") as HTMLButtonElement).disabled).toBe(
      true
    );
    fireEvent.click(screen.getByText("Processing"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
    finish();
    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
  });

  test("keeps failure visible without exposing upstream details", async () => {
    render(
      <ConfirmButton
        title="Delete record"
        description="This cannot be undone."
        trigger={<button type="button">Delete</button>}
        onConfirm={async () => {
          throw new Error("private-secret");
        }}
      />
    );
    fireEvent.click(screen.getByText("Delete"));
    fireEvent.click(screen.getByText("Confirm"));
    await screen.findByRole("alert");
    expect(screen.getByRole("alertdialog")).toBeTruthy();
    expect(screen.queryByText("private-secret")).toBeNull();
  });
});

function GuardHarness({
  dirty = true,
  busy = false,
}: {
  dirty?: boolean;
  busy?: boolean;
}) {
  const [open, setOpen] = useState(true);
  const guard = useFormCloseGuard({ open, dirty, busy, onOpenChange: setOpen });
  return (
    <>
      {guard.dialog}
      <p>{open ? "Editor open" : "Editor closed"}</p>
      <button type="button" onClick={() => guard.requestOpenChange(false)}>
        Close editor
      </button>
    </>
  );
}

describe("Admin edit protection", () => {
  test("supports keeping edits and explicitly discarding them", async () => {
    render(<GuardHarness />);
    fireEvent.click(screen.getByText("Close editor"));
    fireEvent.click(screen.getByText("Keep editing"));
    expect(screen.getByText("Editor open")).toBeTruthy();
    fireEvent.click(screen.getByText("Close editor"));
    fireEvent.click(screen.getByText("Discard changes"));
    await screen.findByText("Editor closed");
  });
  test("closes clean forms without a confirmation", () => {
    render(<GuardHarness dirty={false} />);
    fireEvent.click(screen.getByText("Close editor"));
    expect(screen.getByText("Editor closed")).toBeTruthy();
    expect(screen.queryByRole("alertdialog")).toBeNull();
  });
  test("does not close while saving", () => {
    render(<GuardHarness busy />);
    fireEvent.click(screen.getByText("Close editor"));
    expect(screen.getByText("Editor open")).toBeTruthy();
    expect(screen.queryByRole("alertdialog")).toBeNull();
  });
});

function InvalidForm() {
  const form = useForm({ defaultValues: { title: "" } });
  return (
    <Form {...form}>
      <AdminForm onSubmit={form.handleSubmit(() => undefined)}>
        <input
          aria-label="Title"
          {...form.register("title", { required: "A title is required" })}
        />
        <button type="submit">Save</button>
      </AdminForm>
    </Form>
  );
}

test("summarizes validation errors after submitting a form", async () => {
  render(<InvalidForm />);
  expect(screen.queryByRole("alert")).toBeNull();
  fireEvent.click(screen.getByText("Save"));
  await screen.findByText("A title is required");
});
