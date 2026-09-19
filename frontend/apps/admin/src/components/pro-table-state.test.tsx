/** @vitest-environment jsdom */

import type { ColumnDef } from "@tanstack/react-table";
import {
  type ProTableActions,
  ProTable,
} from "@/components/pro-table/pro-table";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { useCallback, useEffect, useRef, useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("ahooks", () => ({
  useSize: () => ({ height: 600, width: 1024 }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  }),
}));

type TestRow = {
  id: number;
  name: string;
};

type TestFilters = Record<string, never>;

afterEach(cleanup);

function StatefulRowAction({ rowId }: { rowId: number }) {
  const [open, setOpen] = useState(false);

  return (
    <button onClick={() => setOpen((value) => !value)} type="button">
      {open ? `open-${rowId}` : `closed-${rowId}`}
    </button>
  );
}

function TestTable({
  refreshVersion,
  rows,
}: {
  refreshVersion: number;
  rows: TestRow[];
}) {
  const actionRef = useRef<ProTableActions>(null);
  const rowsRef = useRef(rows);
  rowsRef.current = rows;

  const request = useCallback(async () => {
    return {
      list: rowsRef.current,
      total: rowsRef.current.length,
    };
  }, []);

  useEffect(() => {
    if (refreshVersion > 0) {
      actionRef.current?.refresh();
    }
  }, [refreshVersion]);

  const columns: ColumnDef<TestRow, TestFilters>[] = [
    {
      accessorKey: "name",
      header: "Name",
    },
  ];

  return (
    <ProTable<TestRow, TestFilters>
      action={actionRef}
      actions={{
        render: (row) => [
          <StatefulRowAction key="stateful-action" rowId={row.id} />,
        ],
      }}
      columns={columns}
      header={{ hidden: true }}
      request={request}
    />
  );
}

describe("ProTable row action state", () => {
  it("survives parent rerenders and data refreshes for the same row id", async () => {
    const initialRows = [
      { id: 1, name: "One" },
      { id: 2, name: "Two" },
    ];
    const view = render(<TestTable refreshVersion={0} rows={initialRows} />);

    fireEvent.click(await screen.findByRole("button", { name: "closed-1" }));
    expect(screen.getByRole("button", { name: "open-1" })).toBeTruthy();

    view.rerender(<TestTable refreshVersion={1} rows={initialRows} />);
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "open-1" })).toBeTruthy();
    });

    view.rerender(
      <TestTable
        refreshVersion={2}
        rows={[
          { id: 2, name: "Two" },
          { id: 1, name: "One" },
        ]}
      />
    );
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "open-1" })).toBeTruthy();
      expect(screen.getByRole("button", { name: "closed-2" })).toBeTruthy();
    });
  });
});

describe("ProTable workspace states", () => {
  const columns: ColumnDef<TestRow, Record<string, unknown>>[] = [
    { accessorKey: "name", header: "Name" },
  ];

  it("shows a recoverable request failure instead of an empty success state", async () => {
    const request = vi
      .fn()
      .mockRejectedValueOnce(new Error("private upstream response"))
      .mockResolvedValue({ list: [{ id: 1, name: "Recovered" }], total: 1 });
    render(<ProTable columns={columns} request={request} />);
    expect(await screen.findByRole("alert")).toBeTruthy();
    expect(screen.queryByText("private upstream response")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("Recovered")).toBeTruthy();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("returns to the first page when a filter changes", async () => {
    const request = vi.fn(async () => ({
      list: [{ id: 1, name: "One" }],
      total: 30,
    }));
    render(
      <ProTable
        columns={columns}
        params={[{ key: "search", placeholder: "Search names" }]}
        request={request}
      />
    );
    await screen.findByText("One");
    fireEvent.click(screen.getByRole("button", { name: "Go to next page" }));
    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith({ page: 2, size: 10 }, {})
    );
    fireEvent.change(screen.getByRole("textbox", { name: "Search names" }), {
      target: { value: "Alice" },
    });
    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith(
        { page: 1, size: 10 },
        { search: "Alice" }
      )
    );
    fireEvent.click(screen.getByRole("button", { name: "Clear filter" }));
    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith({ page: 1, size: 10 }, {})
    );
  });

  it("uses bounded pagination controls even with millions of pages", async () => {
    const request = vi.fn(async () => ({
      list: [{ id: 1, name: "One" }],
      total: 10_000_000,
    }));
    render(<ProTable columns={columns} request={request} />);
    await screen.findByText("One");
    const pageInput = screen.getByRole("spinbutton", {
      name: "Select page number",
    }) as HTMLInputElement;
    expect(pageInput.max).toBe("1000000");
    expect(screen.getAllByRole("combobox")).toHaveLength(1);
    fireEvent.change(pageInput, { target: { value: "999999" } });
    fireEvent.keyDown(pageInput, { key: "Enter" });
    await waitFor(() =>
      expect(request).toHaveBeenLastCalledWith({ page: 999_999, size: 10 }, {})
    );
    fireEvent.change(pageInput, { target: { value: "1000001" } });
    fireEvent.blur(pageInput);
    expect(pageInput.value).toBe("999999");
  });

  it("allows clearing a bulk selection without changing filters", async () => {
    const onSelectionChange = vi.fn();
    render(
      <ProTable
        columns={columns}
        actions={{
          batchRender: () => [
            <button key="export" type="button">
              Export
            </button>,
          ],
        }}
        onSelectionChange={onSelectionChange}
        request={async () => ({ list: [{ id: 1, name: "One" }], total: 1 })}
      />
    );
    await screen.findByText("One");
    fireEvent.click(screen.getByRole("checkbox", { name: "Select row" }));
    expect(screen.getByRole("region", { name: "Bulk actions" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Clear selection" }));
    expect(screen.queryByRole("region", { name: "Bulk actions" })).toBeNull();
    expect(onSelectionChange).toHaveBeenLastCalledWith([]);
  });
});
