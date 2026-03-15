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
  pagination?: {
    total: number;
    page: number;
    pageSize: number;
    onPageChange: (page: number) => void;
    onPageSizeChange: (pageSize: number) => void;
  };
}

export function Table<T>({
  columns,
  data,
  loading = false,
  rowKey = 'id',
  emptyText = '暂无数据',
  className = '',
  pagination,
}: TableProps<T>) {
  const getRowKey = (record: T, index: number): string => {
    if (typeof rowKey === 'function') {
      return rowKey(record);
    }
    // @ts-expect-error: Record indexing requires string index signature
    return String(record[rowKey] || index);
  };
  const totalPages = pagination ? Math.max(1, Math.ceil(pagination.total / pagination.pageSize)) : 1;
  const canPrev = pagination ? pagination.page > 1 : false;
  const canNext = pagination ? pagination.page < totalPages : false;

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
      {pagination && (
        <div className="table-pagination">
          <div className="table-pagination-info">
            共 {pagination.total} 条，第 {pagination.page} / {totalPages} 页
          </div>
          <div className="table-pagination-actions">
            <button
              type="button"
              className="table-pagination-btn"
              disabled={!canPrev}
              onClick={() => pagination.onPageChange(pagination.page - 1)}
            >
              上一页
            </button>
            <button
              type="button"
              className="table-pagination-btn"
              disabled={!canNext}
              onClick={() => pagination.onPageChange(pagination.page + 1)}
            >
              下一页
            </button>
            <select
              className="table-pagination-size"
              value={pagination.pageSize}
              onChange={(event) => pagination.onPageSizeChange(Number(event.target.value))}
            >
              <option value={10}>10 条/页</option>
              <option value={15}>15 条/页</option>
              <option value={20}>20 条/页</option>
              <option value={50}>50 条/页</option>
              <option value={100}>100 条/页</option>
            </select>
          </div>
        </div>
      )}
    </div>
  );
}
