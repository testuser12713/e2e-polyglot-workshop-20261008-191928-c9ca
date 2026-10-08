import type { ReactNode } from "react";

export interface TableColumn<Row> {
  key: string;
  header: ReactNode;
  render: (row: Row) => ReactNode;
  align?: "left" | "right";
}

export interface TableProps<Row> {
  columns: TableColumn<Row>[];
  rows: Row[];
  rowKey: (row: Row) => string;
  emptyMessage?: ReactNode;
  caption?: string;
}

/**
 * Table primitive (Design.md OrderListRow / Table): header 12px/600 uppercase,
 * rows with a 1px divider. Phone width turns each row into a stacked card via
 * the `.table--stack` class the later list tickets use.
 */
export default function Table<Row>({
  columns,
  rows,
  rowKey,
  emptyMessage = "Keine Einträge",
  caption,
}: TableProps<Row>) {
  if (rows.length === 0) {
    return <p className="table__empty">{emptyMessage}</p>;
  }

  return (
    <div className="table__wrap">
      <table className="table">
        {caption && <caption className="visually-hidden">{caption}</caption>}
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key} className={column.align === "right" ? "num" : undefined}>
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={rowKey(row)}>
              {columns.map((column) => (
                <td key={column.key} className={column.align === "right" ? "num" : undefined}>
                  {column.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
