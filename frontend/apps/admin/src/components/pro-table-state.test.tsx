/** @vitest-environment jsdom */

import type { ColumnDef } from "@tanstack/react-table";
import {
  type ProTableActions,
  ProTable,
} from "@workspace/ui/composed/pro-table/pro-table";
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
