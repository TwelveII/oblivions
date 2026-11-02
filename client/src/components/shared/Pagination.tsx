/** Pagination — shared paged list footer (page size + page numbers + range info). */

import { useTranslation } from "react-i18next";

type PaginationProps = {
  /** 0-indexed current page. */
  page: number;
  /** Total row count (across all pages). */
  total: number;
  /** Rows per page. */
  pageSize: number;
  /** Available page size options. Defaults to [10, 25, 50, 100]. */
  pageSizeOptions?: number[];
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
};

const DEFAULT_OPTIONS = [10, 25, 50, 100];

/** 1-indexed page range with ellipsis: [1, "...", 4, 5, 6, "...", 16]. */
function buildPageRange(current1: number, totalPages: number): (number | "ellipsis-l" | "ellipsis-r")[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const out: (number | "ellipsis-l" | "ellipsis-r")[] = [1];
  if (current1 > 3) out.push("ellipsis-l");
  const from = Math.max(2, current1 - 1);
  const to = Math.min(totalPages - 1, current1 + 1);
  for (let p = from; p <= to; p++) out.push(p);
  if (current1 < totalPages - 2) out.push("ellipsis-r");
  out.push(totalPages);
  return out;
}

function Pagination({ page, total, pageSize, pageSizeOptions = DEFAULT_OPTIONS, onPageChange, onPageSizeChange }: PaginationProps) {
  const { t } = useTranslation("common");
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const current1 = page + 1;
  const from = total === 0 ? 0 : page * pageSize + 1;
  const to = Math.min(total, (page + 1) * pageSize);

  const range = buildPageRange(current1, totalPages);

  return (
    <div className="pagination">
      <div className="pagination-left">
        <label className="pagination-size">
          <span className="pagination-size-label">{t("pagination.rowsPerPage")}</span>
          <select
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
            className="pagination-size-select"
          >
            {pageSizeOptions.map((opt) => (
              <option key={opt} value={opt}>{opt}</option>
            ))}
          </select>
        </label>
        <span className="pagination-info">
          {t("pagination.showing", { from, to, total })}
        </span>
      </div>

      <div className="pagination-right">
        <button
          className="pagination-arrow"
          disabled={page === 0}
          onClick={() => onPageChange(page - 1)}
          title={t("pagination.previous")}
        >
          &#x2039;
        </button>
        {range.map((p, i) =>
          p === "ellipsis-l" || p === "ellipsis-r" ? (
            <span key={`${p}-${i}`} className="pagination-ellipsis">…</span>
          ) : (
            <button
              key={p}
              className={`pagination-page${p === current1 ? " active" : ""}`}
              onClick={() => onPageChange(p - 1)}
            >
              {p}
            </button>
          )
        )}
        <button
          className="pagination-arrow"
          disabled={page >= totalPages - 1}
          onClick={() => onPageChange(page + 1)}
          title={t("pagination.next")}
        >
          &#x203A;
        </button>
      </div>
    </div>
  );
}

export default Pagination;
