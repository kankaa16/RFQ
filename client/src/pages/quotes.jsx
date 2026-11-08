
import { useCallback, useEffect, useMemo, useState } from "react";
import { apiRequest } from "../services/api";

const formatCurrency = (value) =>
  new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2,
  }).format(Number(value || 0));

const getId = (item) => item?.id || item?._id || "";

const statusStyles = {
  draft: "quote-status draft",
  pending_approval: "quote-status pending",
  approved: "quote-status approved",
  rejected: "quote-status rejected",
  sent: "quote-status sent",
};

function Quotes() {
  const [quotes, setQuotes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");

  const loadQuotes = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const response = await apiRequest("/api/quotes");
      const records = response.data;

      setQuotes(Array.isArray(records) ? records : []);
    } catch (err) {
      setError(err.message || "Unable to load quotes.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadQuotes();
  }, [loadQuotes]);

  const filteredQuotes = useMemo(() => {
    return quotes.filter((quote) => {
      const query = search.trim().toLowerCase();

      const matchesSearch =
        !query ||
        [
          quote.quote_number,
          quote.rfq_number,
          quote.customer_name,
          quote.status,
        ].some((value) =>
          String(value || "").toLowerCase().includes(query)
        );

      const matchesStatus =
        statusFilter === "all" || quote.status === statusFilter;

      return matchesSearch && matchesStatus;
    });
  }, [quotes, search, statusFilter]);

  const totalQuotes = quotes.length;

  const pendingQuotes = quotes.filter(
    (quote) => quote.status === "pending_approval"
  ).length;

  const approvedQuotes = quotes.filter(
    (quote) => quote.status === "approved"
  ).length;

  return (
    <div className="quotes-page">
      <div className="quotes-header">
        <div>
          <p className="quotes-eyebrow">COMMERCIAL</p>
          <h1>Quotes</h1>
          <p className="quotes-description">
            Track generated quotations and their approval status.
          </p>
        </div>

        <button
          className="quotes-refresh"
          onClick={loadQuotes}
          disabled={loading}
        >
          {loading ? "Refreshing..." : "↻ Refresh"}
        </button>
      </div>

      <div className="quotes-stats">
        <div className="quotes-stat">
          <span>Total quotes</span>
          <strong>{totalQuotes}</strong>
        </div>

        <div className="quotes-stat">
          <span>Pending approval</span>
          <strong>{pendingQuotes}</strong>
        </div>

        <div className="quotes-stat">
          <span>Approved</span>
          <strong>{approvedQuotes}</strong>
        </div>
      </div>

      <div className="quotes-toolbar">
        <input
          type="search"
          placeholder="Search quote, RFQ or customer..."
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />

        <select
          value={statusFilter}
          onChange={(event) => setStatusFilter(event.target.value)}
        >
          <option value="all">All statuses</option>
          <option value="draft">Draft</option>
          <option value="pending_approval">Pending approval</option>
          <option value="approved">Approved</option>
          <option value="rejected">Rejected</option>
          <option value="sent">Sent</option>
        </select>
      </div>

      {error && (
        <div className="quotes-error">
          <span>{error}</span>
          <button onClick={loadQuotes}>Retry</button>
        </div>
      )}

      <div className="quotes-table-wrap">
        <table className="quotes-table">
          <thead>
            <tr>
              <th>Quote number</th>
              <th>RFQ</th>
              <th>Customer</th>
              <th>Annual value</th>
              <th>Grand total</th>
              <th>Status</th>
            </tr>
          </thead>

          <tbody>
            {loading ? (
              <tr>
                <td colSpan="6" className="quotes-message">
                  Loading quotations...
                </td>
              </tr>
            ) : filteredQuotes.length === 0 ? (
              <tr>
                <td colSpan="6" className="quotes-message">
                  {quotes.length === 0
                    ? "No quotes have been created yet."
                    : "No quotes match your search."}
                </td>
              </tr>
            ) : (
              filteredQuotes.map((quote) => (
                <tr key={getId(quote) || quote.quote_number}>
                  <td className="quote-number">
                    {quote.quote_number || "—"}
                  </td>

                  <td>{quote.rfq_number || quote.rfq_id || "—"}</td>

                  <td>{quote.customer_name || "—"}</td>

                  <td>{formatCurrency(quote.annual_value)}</td>

                  <td>{formatCurrency(quote.grand_total)}</td>

                  <td>
                    <span
                      className={
                        statusStyles[quote.status] || "quote-status"
                      }
                    >
                      {(quote.status || "unknown").replaceAll("_", " ")}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <style>{`
        .quotes-page {
          color: var(--text-primary, #20231f);
          width: 100%;
        }

        .quotes-header {
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          gap: 16px;
          margin-bottom: 28px;
        }

        .quotes-eyebrow {
          font-size: 10px;
          letter-spacing: 1.5px;
          font-weight: 700;
          color: #858a81;
          margin: 0 0 8px;
        }

        .quotes-header h1 {
          font-size: 28px;
          margin: 0;
          font-weight: 650;
          letter-spacing: -0.8px;
        }

        .quotes-description {
          color: #858a81;
          font-size: 13px;
          margin-top: 8px;
        }

        .quotes-refresh {
          border: 1px solid #dedfd9;
          background: #fff;
          border-radius: 8px;
          padding: 9px 13px;
          font-size: 12px;
          cursor: pointer;
          color: #33382f;
        }

        .quotes-refresh:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .quotes-stats {
          display: grid;
          grid-template-columns: repeat(3, minmax(0, 1fr));
          gap: 14px;
          margin-bottom: 22px;
        }

        .quotes-stat {
          background: #fff;
          border: 1px solid #e8e9e4;
          border-radius: 11px;
          padding: 17px 19px;
        }

        .quotes-stat span {
          display: block;
          font-size: 12px;
          color: #858a81;
          margin-bottom: 10px;
        }

        .quotes-stat strong {
          font-size: 24px;
          font-weight: 650;
        }

        .quotes-toolbar {
          display: flex;
          gap: 10px;
          margin-bottom: 15px;
        }

        .quotes-toolbar input,
        .quotes-toolbar select {
          border: 1px solid #e1e3dc;
          background: #fff;
          border-radius: 8px;
          padding: 11px 13px;
          font-size: 12px;
          outline: none;
          color: #30342e;
        }

        .quotes-toolbar input {
          flex: 1;
          min-width: 0;
        }

        .quotes-toolbar input:focus,
        .quotes-toolbar select:focus {
          border-color: #858d77;
        }

        .quotes-table-wrap {
          background: #fff;
          border: 1px solid #e8e9e4;
          border-radius: 11px;
          overflow-x: auto;
        }

        .quotes-table {
          width: 100%;
          border-collapse: collapse;
          min-width: 700px;
          text-align: left;
        }

        .quotes-table th {
          background: #f8f9f6;
          color: #858a81;
          font-size: 10px;
          font-weight: 650;
          text-transform: uppercase;
          letter-spacing: .6px;
          padding: 14px 16px;
          white-space: nowrap;
        }

        .quotes-table td {
          border-top: 1px solid #eff0ec;
          padding: 15px 16px;
          font-size: 12px;
          white-space: nowrap;
        }

        .quote-number {
          font-weight: 650;
        }

        .quote-status {
          display: inline-block;
          padding: 5px 9px;
          border-radius: 20px;
          background: #f0f1ed;
          color: #5e6258;
          font-size: 10px;
          text-transform: capitalize;
        }

        .quote-status.pending {
          background: #fff3dc;
          color: #986715;
        }

        .quote-status.approved {
          background: #e8f4e9;
          color: #347444;
        }

        .quote-status.rejected {
          background: #fbeaea;
          color: #a34343;
        }

        .quote-status.sent {
          background: #e9effb;
          color: #42639c;
        }

        .quotes-message {
          text-align: center;
          color: #858a81;
          padding: 35px !important;
        }

        .quotes-error {
          display: flex;
          justify-content: space-between;
          gap: 12px;
          padding: 12px 15px;
          margin-bottom: 14px;
          border: 1px solid #f0cccc;
          border-radius: 8px;
          background: #fff5f5;
          color: #a34343;
          font-size: 12px;
        }

        .quotes-error button {
          background: transparent;
          border: 0;
          color: inherit;
          text-decoration: underline;
          cursor: pointer;
        }

        @media (max-width: 650px) {
          .quotes-header {
            flex-direction: column;
          }

          .quotes-stats {
            grid-template-columns: 1fr;
          }

          .quotes-toolbar {
            flex-direction: column;
          }

          .quotes-toolbar select {
            width: 100%;
          }
        }
      `}</style>
    </div>
  );
}

export default Quotes;
