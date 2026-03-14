import React from 'react';
import './Table.css';

export interface Column<T> {
  key: string;
  title: string;
  render?: (record: T) => React.ReactNode;
  width?: string | number;
}

export interface TableProps<T> {
  columns: Column<T>[];
  data: T[];
  loading?: boolean;
  rowKey?: string | ((record: T) => string);
  emptyText?: string;
  className?: string;
}

export function Table<T>({
  columns,
  data,
  loading = false,
  rowKey = 'id',
  emptyText = '暂无数据',
  className = '',
}: TableProps<T>) {
  const getRowKey = (record: T, index: number): string => {
    if (typeof rowKey === 'function') {
      return rowKey(record);
    }
    // @ts-expect-error: Record indexing requires string index signature
    return record[rowKey] || index;
  };

  return (
    <div className={`table-container ${className}`}>
      <table className="table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th key={col.key} style={{ width: col.width }}>
                {col.title}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <tr>
              <td colSpan={columns.length} className="table-loading">
                加载中...
              </td>
            </tr>
          ) : data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="table-empty">
                {emptyText}
              </td>
            </tr>
          ) : (
            data.map((record, index) => (
              <tr key={getRowKey(record, index)}>
                {columns.map((col) => (
                  <td key={col.key}>
                    {col.render ? col.render(record) : (record as Record<string, unknown>)[col.key] as React.ReactNode}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
