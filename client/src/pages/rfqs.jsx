import React from "react";
import {
  Search,
  RefreshCw,
  ChevronRight,
  X,
  Package,
  Send,
  Trophy,
  XCircle,
  Loader2,
  FileText
} from "lucide-react";

import StatusBadge from "../components/status_badge.jsx";
import { apiRequest } from "../services/api.js";

function formatCurrency(value) {
  return `₹${Number(value || 0).toLocaleString("en-IN", {
    maximumFractionDigits: 2
  })}`;
}

function formatDate(value) {
  if (!value) return "—";

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) return "—";

  return date.toLocaleDateString("en-IN");
}

function getId(value) {
  if (!value) return "";
  if (typeof value === "string") return value;
  if (value.$oid) return value.$oid;
  if (value.hex) return value.hex;
  return "";
}

export default function RFQs({
  rfqs = [],
  loading,
  onRefresh,
  onLoadParts
}) {
  const [query, setQuery] = React.useState("");
  const [status, setStatus] = React.useState("all");

  const [selected, setSelected] = React.useState(null);
  const [parts, setParts] = React.useState([]);
  const [partsLoading, setPartsLoading] = React.useState(false);

  const [userRole, setUserRole] = React.useState("");
  const [updating, setUpdating] = React.useState(false);
  const [lossReason, setLossReason] = React.useState("");

  const [error, setError] = React.useState("");
  const [success, setSuccess] = React.useState("");

  React.useEffect(() => {
    async function loadProfile() {
      try {
        const response = await apiRequest("/api/profile");
        const profile = response.data ?? response;
        setUserRole(profile.role || "");
      } catch (err) {
        console.error("could not load user profile", err);
      }
    }

    loadProfile();
  }, []);

  // keep the selected drawer record in sync with refreshed rfq data
  React.useEffect(() => {
    if (!selected) return;

    const selectedId = getId(selected.id || selected._id);

    const refreshedRFQ = rfqs.find(
      (rfq) => getId(rfq.id || rfq._id) === selectedId
    );

    if (refreshedRFQ) {
      setSelected(refreshedRFQ);
    }
  }, [rfqs]);

  const filtered = rfqs.filter((rfq) => {
    const searchableText = [
      rfq.rfq_number,
      rfq.customer_name,
      rfq.latest_quote_number
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();

    const match = searchableText.includes(query.toLowerCase());

    return match && (status === "all" || rfq.status === status);
  });

  async function openRFQ(rfq) {
    setSelected(rfq);
    setParts([]);
    setPartsLoading(true);
    setError("");
    setSuccess("");
    setLossReason("");

    try {
      const loadedParts = await onLoadParts(
        getId(rfq.id || rfq._id)
      );

      setParts(Array.isArray(loadedParts) ? loadedParts : []);
    } catch {
      setParts([]);
    } finally {
      setPartsLoading(false);
    }
  }

  async function updateStatus(nextStatus) {
    if (!selected || updating) return;

    if (nextStatus === "lost" && !lossReason.trim()) {
      setError("Please enter a reason for losing this RFQ.");
      return;
    }

    const confirmation = {
      sent: "Mark this RFQ as sent to the customer?",
      won: "Confirm that the customer has accepted this RFQ?",
      lost: "Confirm that this RFQ has been lost?"
    };

    if (!window.confirm(confirmation[nextStatus])) return;

    setUpdating(true);
    setError("");
    setSuccess("");

    try {
      const selectedId = getId(selected.id || selected._id);

      const payload = {
        status: nextStatus
      };

      if (nextStatus === "lost") {
        payload.lost_reason = lossReason.trim();
      }

      const response = await apiRequest(
        `/api/rfqs/${selectedId}/status`,
        {
          method: "PATCH",
          body: JSON.stringify(payload)
        }
      );

      if (response.success === false) {
        throw new Error(response.message || "Status update failed.");
      }

      setSelected((current) => ({
        ...current,
        status: nextStatus,
        lost_reason:
          nextStatus === "lost"
            ? lossReason.trim()
            : current.lost_reason
      }));

      setSuccess(`RFQ successfully marked as ${nextStatus}.`);
      setLossReason("");

      await onRefresh();
    } catch (err) {
      setError(err.message || "Could not update RFQ status.");
    } finally {
      setUpdating(false);
    }
  }

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">
            <i /> WORK QUEUE
          </span>

          <h1>RFQ register</h1>

          <p>
            Track every customer request from intake to outcome.
          </p>
        </div>

        <button
          className="secondary-button"
          onClick={onRefresh}
          disabled={loading}
        >
          <RefreshCw size={15} />
          Refresh
        </button>
      </div>

      <section className="panel">
        <div className="toolbar">
          <div className="search-box">
            <Search size={16} />

            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search RFQ, customer or quote..."
            />
          </div>

          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="all">All statuses</option>
            <option value="estimating">Estimating</option>
            <option value="sent">Sent</option>
            <option value="won">Won</option>
            <option value="lost">Lost</option>
          </select>

          <span className="result-count">
            {filtered.length} records
          </span>
        </div>

        {loading ? (
          <div className="empty-state">Loading RFQs...</div>
        ) : (
          <div className="table-wrap">
            <table className="wide-table">
              <thead>
                <tr>
                  <th>RFQ NUMBER</th>
                  <th>CUSTOMER</th>
                  <th>PARTS</th>
                  <th>RECEIVED</th>
                  <th>DUE DATE</th>
                  <th>EXPECTED VALUE</th>
                  <th>RFQ STATUS</th>
                  <th>QUOTE NUMBER</th>
                  <th>QUOTE STATUS</th>
                  <th>QUOTED VALUE</th>
                  <th />
                </tr>
              </thead>

              <tbody>
                {filtered.map((rfq) => {
                  const rfqId = getId(rfq.id || rfq._id);

                  return (
                    <tr
                      key={rfqId}
                      onClick={() => openRFQ(rfq)}
                      className="clickable-row"
                    >
                      <td>
                        <strong>{rfq.rfq_number}</strong>
                      </td>

                      <td>{rfq.customer_name || "—"}</td>

                      <td>{rfq.part_count ?? "—"}</td>

                      <td>{formatDate(rfq.received_at)}</td>

                      <td>{formatDate(rfq.due_date)}</td>

                      <td>
                        {rfq.expected_annual_value
                          ? formatCurrency(rfq.expected_annual_value)
                          : "—"}
                      </td>

                      <td>
                        <StatusBadge status={rfq.status} />
                      </td>

                      <td>
                        {rfq.latest_quote_number || "—"}
                      </td>

                      <td>
                        {rfq.latest_quote_status ? (
                          <StatusBadge
                            status={rfq.latest_quote_status}
                          />
                        ) : (
                          "—"
                        )}
                      </td>

                      <td>
                        {rfq.latest_quote_grand_total
                          ? formatCurrency(rfq.latest_quote_grand_total)
                          : "—"}
                      </td>

                      <td>
                        <ChevronRight
                          size={16}
                          className="muted-icon"
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>

            {!filtered.length && (
              <div className="empty-state">
                No matching RFQs found.
              </div>
            )}
          </div>
        )}
      </section>

      {selected && (
        <div
          className="drawer-backdrop"
          onClick={() => !updating && setSelected(null)}
        >
          <aside
            className="detail-drawer"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="drawer-head">
              <div>
                <span className="eyebrow">RFQ DETAILS</span>
                <h2>{selected.rfq_number}</h2>
              </div>

              <button
                className="icon-button"
                onClick={() => setSelected(null)}
                disabled={updating}
              >
                <X size={18} />
              </button>
            </div>

            <div className="drawer-customer">
              {selected.customer_name}
              <StatusBadge status={selected.status} />
            </div>

            <div className="detail-pair">
              <span>Due date</span>
              <strong>{formatDate(selected.due_date)}</strong>
            </div>

            <div className="detail-pair">
              <span>Annual value</span>
              <strong>
                {formatCurrency(selected.expected_annual_value)}
              </strong>
            </div>

            {selected.status === "lost" && selected.lost_reason && (
              <div className="detail-pair">
                <span>Loss reason</span>
                <strong>{selected.lost_reason}</strong>
              </div>
            )}

            <div className="drawer-section-title">
              <FileText size={16} />
              QUOTATION DETAILS
            </div>

            <div className="detail-pair">
              <span>Latest quote</span>
              <strong>
                {selected.latest_quote_number || "Not created"}
              </strong>
            </div>

            <div className="detail-pair">
              <span>Quote status</span>
              <strong>
                {selected.latest_quote_status ? (
                  <StatusBadge status={selected.latest_quote_status} />
                ) : (
                  "—"
                )}
              </strong>
            </div>

            <div className="detail-pair">
              <span>Quoted value (including GST)</span>
              <strong>
                {selected.latest_quote_grand_total
                  ? formatCurrency(selected.latest_quote_grand_total)
                  : "—"}
              </strong>
            </div>

            <div className="detail-pair">
              <span>Annual quotation value</span>
              <strong>
                {selected.latest_quote_annual_value
                  ? formatCurrency(selected.latest_quote_annual_value)
                  : "—"}
              </strong>
            </div>

            <div className="detail-pair">
              <span>Quote created</span>
              <strong>{formatDate(selected.latest_quoted_at)}</strong>
            </div>

            <div className="drawer-section-title">
              <Package size={16} />
              PARTS ({parts.length || selected.part_count || 0})
            </div>

            {partsLoading ? (
              <div className="empty-state">
                Loading parts...
              </div>
            ) : (
              parts.map((part) => (
                <div
                  className="part-card"
                  key={getId(part.id || part._id)}
                >
                  <strong>{part.part_name}</strong>

                  <span>{part.part_number}</span>

                  <small>
                    {part.material_grade} · {part.gross_weight} kg gross ·{" "}
                    {Number(part.annual_volume).toLocaleString("en-IN")} / year
                  </small>

                  <small>
                    Drawing: {part.drawing_number || "—"} · Rev{" "}
                    {part.drawing_revision || "—"}
                  </small>
                </div>
              ))
            )}

            {(error || success) && (
              <div className="status-message" role="status">
                {error && (
                  <p className="error-message">{error}</p>
                )}

                {success && (
                  <p className="success-message">{success}</p>
                )}
              </div>
            )}

            {selected.status === "estimating" && (
              <div className="status-actions">
                <button
                  className="primary-button"
                  onClick={() => updateStatus("sent")}
                  disabled={updating}
                >
                  {updating ? (
                    <Loader2 size={15} />
                  ) : (
                    <Send size={15} />
                  )}

                  Mark as sent
                </button>
              </div>
            )}

            {selected.status === "sent" && userRole === "sales_head" && (
              <div className="status-actions">
                <button
                  className="primary-button"
                  onClick={() => updateStatus("won")}
                  disabled={updating}
                >
                  <Trophy size={15} />
                  Mark as won
                </button>

                <div className="loss-reason">
                  <label htmlFor="loss-reason">
                    Reason for loss
                  </label>

                  <textarea
                    id="loss-reason"
                    value={lossReason}
                    onChange={(e) => setLossReason(e.target.value)}
                    placeholder="Enter why the customer did not proceed..."
                    rows={3}
                    disabled={updating}
                  />

                  <button
                    className="secondary-button"
                    onClick={() => updateStatus("lost")}
                    disabled={updating || !lossReason.trim()}
                  >
                    <XCircle size={15} />
                    Mark as lost
                  </button>
                </div>
              </div>
            )}

            {updating && (
              <p className="loading-text">
                Updating RFQ status...
              </p>
            )}
          </aside>
        </div>
      )}
    </>
  );
}