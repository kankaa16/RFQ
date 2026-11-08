
import React, { useCallback, useEffect, useState } from "react";
import {
  Check,
  X,
  RefreshCw,
  ShieldCheck,
  Clock3,
  FileText,
  AlertCircle,
  MessageSquare
} from "lucide-react";
import { apiRequest } from "../services/api.js";

function formatCurrency(value) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2
  }).format(Number(value || 0));
}

function formatDate(value) {
  if (!value) return "—";

  return new Date(value).toLocaleDateString("en-IN", {
    day: "2-digit",
    month: "short",
    year: "numeric"
  });
}

export default function Approval() {
  const [quotes, setQuotes] = useState([]);
  const [activeTab, setActiveTab] = useState("pending");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [selectedQuote, setSelectedQuote] = useState(null);
  const [action, setAction] = useState("");
  const [comments, setComments] = useState("");
  const [rejectionReason, setRejectionReason] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const loadApprovals = useCallback(async () => {
    try {
      setLoading(true);
      setError("");

      const response = await apiRequest("/api/approvals");
      setQuotes(Array.isArray(response.data) ? response.data : []);
    } catch (err) {
      setError(err.message || "Could not load approvals.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadApprovals();
  }, [loadApprovals]);

  const pendingQuotes = quotes.filter(
    (quote) => quote.status === "pending_approval"
  );

  const approvedQuotes = quotes.filter(
    (quote) => quote.status === "approved"
  );

  const rejectedQuotes = quotes.filter(
    (quote) => quote.status === "rejected"
  );

  const reviewedQuotes = [...approvedQuotes, ...rejectedQuotes];

  const visibleQuotes =
    activeTab === "pending" ? pendingQuotes : reviewedQuotes;

  function openReview(quote, selectedAction) {
    setSelectedQuote(quote);
    setAction(selectedAction);
    setComments("");
    setRejectionReason("");
    setError("");
    setSuccess("");
  }

  function closeReview() {
    if (submitting) return;

    setSelectedQuote(null);
    setAction("");
    setComments("");
    setRejectionReason("");
  }

  async function submitReview() {
    if (!selectedQuote || !action) return;

    if (action === "rejected" && !rejectionReason.trim()) {
      setError("Please provide a rejection reason.");
      return;
    }

    try {
      setSubmitting(true);
      setError("");
      setSuccess("");

      const quoteId = selectedQuote.id || selectedQuote._id;

      const response = await apiRequest(
        `/api/quotes/${quoteId}/review`,
        {
          method: "POST",
          body: JSON.stringify({
            action,
            comments: comments.trim(),
            rejection_reason: rejectionReason.trim()
          })
        }
      );

      setSuccess(response.message || `Quote ${action} successfully.`);
      setSelectedQuote(null);
      setAction("");
      setComments("");
      setRejectionReason("");

      await loadApprovals();
    } catch (err) {
      setError(err.message || "Could not submit review.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <div className="page-heading">
        <div>
          <div className="eyebrow">
            <span className="status-dot" />
            WORKFLOW
          </div>

          <h1>Approvals</h1>
          <p>Review and decide on quotations awaiting sales approval.</p>
        </div>

        <button
          className="secondary-button"
          onClick={loadApprovals}
          disabled={loading}
        >
          <RefreshCw size={15} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="global-error">
          <span>
            <AlertCircle size={15} /> {error}
          </span>
          <button onClick={() => setError("")}>Dismiss</button>
        </div>
      )}

      {success && (
        <div
          style={{
            background: "#f1f7e9",
            border: "1px solid #dce8cd",
            color: "#64813d",
            padding: "11px 13px",
            borderRadius: 7,
            fontSize: 11,
            marginBottom: 15
          }}
        >
          <Check size={15} style={{ verticalAlign: "middle" }} /> {success}
        </div>
      )}

      {/* summary cards */}
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-top">
            <span>Pending approval</span>
            <span className="stat-icon">
              <Clock3 size={17} />
            </span>
          </div>
          <strong>{pendingQuotes.length}</strong>
          <small>Awaiting your decision</small>
        </div>

        <div className="stat-card">
          <div className="stat-top">
            <span>Approved</span>
            <span className="stat-icon">
              <Check size={17} />
            </span>
          </div>
          <strong>{approvedQuotes.length}</strong>
          <small>Successfully approved</small>
        </div>

        <div className="stat-card">
          <div className="stat-top">
            <span>Rejected</span>
            <span className="stat-icon">
              <X size={17} />
            </span>
          </div>
          <strong>{rejectedQuotes.length}</strong>
          <small>Rejected quotations</small>
        </div>

        <div className="stat-card">
          <div className="stat-top">
            <span>Total reviewed</span>
            <span className="stat-icon">
              <ShieldCheck size={17} />
            </span>
          </div>
          <strong>{reviewedQuotes.length}</strong>
          <small>Completed decisions</small>
        </div>
      </div>

      {/* approval workspace */}
      <div className="panel">
        <div className="panel-heading">
          <div>
            <h2>Approval workspace</h2>
            <p>Review quotations and their approval status.</p>
          </div>

          <span className="updated-label">
            <span />
            {quotes.length} quotations
          </span>
        </div>

        {/* tabs */}
        <div
          className="toolbar"
          style={{
            borderBottom: "1px solid #eee",
            paddingBottom: 13,
            marginBottom: 0
          }}
        >
          <button
            className={
              activeTab === "pending"
                ? "primary-button"
                : "secondary-button"
            }
            onClick={() => setActiveTab("pending")}
          >
            <Clock3 size={14} />
            Pending ({pendingQuotes.length})
          </button>

          <button
            className={
              activeTab === "history"
                ? "primary-button"
                : "secondary-button"
            }
            onClick={() => setActiveTab("history")}
          >
            <FileText size={14} />
            Review history ({reviewedQuotes.length})
          </button>
        </div>

        {loading ? (
          <div className="empty-state">
            <RefreshCw className="spin" size={23} />
            <strong>Loading approvals</strong>
            <span>Fetching quotation records...</span>
          </div>
        ) : visibleQuotes.length === 0 ? (
          <div className="empty-state" style={{ minHeight: 240 }}>
            <ShieldCheck size={34} />
            <strong>
              {activeTab === "pending"
                ? "No pending approvals"
                : "No review history"}
            </strong>
            <span>
              {activeTab === "pending"
                ? "Quotations requiring sales head approval will appear here."
                : "Approved and rejected quotations will appear here."}
            </span>
          </div>
        ) : (
          <div className="table-wrap" style={{ marginTop: 5 }}>
            <table>
              <thead>
                <tr>
                  <th>QUOTATION</th>
                  <th>RFQ ID</th>
                  <th>ANNUAL VALUE</th>
                  <th>GRAND TOTAL</th>
                  <th>SUBMITTED</th>
                  <th>STATUS</th>
                  {activeTab === "pending" && <th>ACTIONS</th>}
                </tr>
              </thead>

              <tbody>
                {visibleQuotes.map((quote) => (
                  <tr key={quote.id || quote._id}>
                    <td>
                      <div
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: 8
                        }}
                      >
                        <FileText size={16} color="#888" />
                        <strong>{quote.quote_number}</strong>
                      </div>
                    </td>

                    <td>{quote.rfq_number || quote.rfq_id || "—"}</td>

                    <td>
                      <strong>{formatCurrency(quote.annual_value)}</strong>
                    </td>

                    <td>{formatCurrency(quote.grand_total)}</td>

                    <td>{formatDate(quote.created_at)}</td>

                    <td>
                      <span
                        className={`status-badge ${
                          quote.status === "approved"
                            ? "green"
                            : quote.status === "rejected"
                              ? "red"
                              : "blue"
                        }`}
                      >
                        <span />
                        {quote.status.replaceAll("_", " ")}
                      </span>
                    </td>

                    {activeTab === "pending" && (
                      <td>
                        <div style={{ display: "flex", gap: 7 }}>
                          <button
                            className="primary-button"
                            style={{
                              padding: "7px 10px",
                              fontSize: 10,
                              background: "#edf5e4",
                              borderColor: "#dce8cd",
                              color: "#587a32"
                            }}
                            onClick={() => openReview(quote, "approved")}
                          >
                            <Check size={13} />
                            Approve
                          </button>

                          <button
                            className="secondary-button"
                            style={{
                              padding: "7px 10px",
                              fontSize: 10,
                              color: "#a45147",
                              borderColor: "#f0d9d5"
                            }}
                            onClick={() => openReview(quote, "rejected")}
                          >
                            <X size={13} />
                            Reject
                          </button>
                        </div>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* review drawer */}
      {selectedQuote && (
        <div className="drawer-backdrop" onClick={closeReview}>
          <div
            className="detail-drawer"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="drawer-head">
              <div>
                <span className="eyebrow">QUOTATION REVIEW</span>
                <h2>
                  {action === "approved"
                    ? "Approve quotation"
                    : "Reject quotation"}
                </h2>
              </div>

              <button
                className="icon-button"
                onClick={closeReview}
                disabled={submitting}
                aria-label="Close"
              >
                <X size={19} />
              </button>
            </div>

            <div className="drawer-customer">
              <span>{selectedQuote.quote_number}</span>
              <span className="status-badge blue">
                {formatCurrency(selectedQuote.annual_value)}
              </span>
            </div>

            <p
              style={{
                fontSize: 10,
                color: "#999",
                margin: "10px 0 22px"
              }}
            >
              Annual quotation value
            </p>

            <div className="drawer-section-title">
              <MessageSquare size={15} />
              REVIEW COMMENTS
            </div>

            <label
              htmlFor="approval-comments"
              style={{ fontSize: 10, color: "#666", fontWeight: 600 }}
            >
              Comments
            </label>

            <textarea
              id="approval-comments"
              value={comments}
              onChange={(event) => setComments(event.target.value)}
              placeholder="Add review comments (optional)"
              rows={4}
              style={{
                width: "100%",
                resize: "vertical",
                marginTop: 7,
                padding: 11,
                border: "1px solid #e5e5df",
                borderRadius: 7,
                font: "11px DM Sans, sans-serif"
              }}
            />

            {action === "rejected" && (
              <>
                <label
                  htmlFor="rejection-reason"
                  style={{
                    display: "block",
                    fontSize: 10,
                    color: "#666",
                    fontWeight: 600,
                    marginTop: 20
                  }}
                >
                  Rejection reason *
                </label>

                <textarea
                  id="rejection-reason"
                  value={rejectionReason}
                  onChange={(event) =>
                    setRejectionReason(event.target.value)
                  }
                  placeholder="Explain why this quotation is being rejected"
                  rows={4}
                  style={{
                    width: "100%",
                    resize: "vertical",
                    marginTop: 7,
                    padding: 11,
                    border: "1px solid #e5e5df",
                    borderRadius: 7,
                    font: "11px DM Sans, sans-serif"
                  }}
                />
              </>
            )}

            <div
              style={{
                display: "flex",
                justifyContent: "flex-end",
                gap: 9,
                marginTop: 24
              }}
            >
              <button
                className="secondary-button"
                onClick={closeReview}
                disabled={submitting}
              >
                Cancel
              </button>

              <button
                className={
                  action === "approved"
                    ? "primary-button"
                    : "secondary-button"
                }
                style={
                  action === "rejected"
                    ? {
                        color: "#a45147",
                        borderColor: "#f0d9d5"
                      }
                    : {}
                }
                onClick={submitReview}
                disabled={submitting}
              >
                {submitting
                  ? "Submitting..."
                  : action === "approved"
                    ? "Confirm approval"
                    : "Confirm rejection"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
