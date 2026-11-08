
import React from "react";
import {
  ArrowUpRight,
  ArrowRight,
  TrendingUp,
  RefreshCw,
  FileText
} from "lucide-react";
import StatusBadge from "../components/status_badge.jsx";
import { apiRequest } from "../services/api.js";

const money = (n) =>
  new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0
  }).format(Number(n || 0));

export default function SalesHeadDashboard({ rfqs = [], loading, onRefresh, setPage }) {
  const [quotes, setQuotes] = React.useState([]);
  const [quotesLoading, setQuotesLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  const loadSalesData = React.useCallback(async () => {
    setQuotesLoading(true);
    setError("");

    try {
      const result = await apiRequest("/api/quotes");
      setQuotes(result.data || []);
    } catch (e) {
      setError(e.message || "Could not load quotes.");
    } finally {
      setQuotesLoading(false);
    }
  }, []);

  React.useEffect(() => {
    loadSalesData();
  }, [loadSalesData]);

  const sentQuotes = quotes.filter((quote) => quote.status === "sent");
  const wonRFQs = rfqs.filter((rfq) => String(rfq.status).toLowerCase() === "won");
  const lostRFQs = rfqs.filter((rfq) => String(rfq.status).toLowerCase() === "lost");
  const activeRFQs = rfqs.filter(
    (rfq) => !["won", "lost"].includes(String(rfq.status).toLowerCase())
  );

  const recentQuotes = [...quotes]
    .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
    .slice(0, 5);

  const refreshAll = () => {
    onRefresh?.();
    loadSalesData();
  };

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow"><i /> SALES OVERVIEW</span>
          <h1>Sales dashboard.</h1>
          <p>Monitor quotation activity, customer opportunities and sales outcomes.</p>
        </div>

        <button className="secondary-button" onClick={refreshAll}>
          <RefreshCw size={15} /> Refresh data
        </button>
      </div>

      <section className="welcome-strip">
        <div>
          <span className="strip-label">SALES MANAGEMENT</span>
          <h2>Keep every opportunity moving.</h2>
          <p>Track quotations and monitor RFQ outcomes.</p>
        </div>
        <div className="strip-orbit orbit-a" />
        <div className="strip-orbit orbit-b" />
      </section>

      <div className="section-row">
        <div>
          <h2>Sales at a glance</h2>
          <p>Live figures from your RFQ and quotation records</p>
        </div>
        <span className="updated-label">
          <span />
          {loading || quotesLoading ? "Updating..." : "Connected to workspace"}
        </span>
      </div>

      <section className="stats-grid">
        <article className="stat-card">
          <div className="stat-top">
            <span>Active RFQs</span>
            <span className="stat-icon"><TrendingUp size={17} /></span>
          </div>
          <strong>{activeRFQs.length.toString().padStart(2, "0")}</strong>
          <small>Currently in progress</small>
        </article>

        <article className="stat-card">
          <div className="stat-top">
            <span>Quotes sent</span>
            <span className="stat-icon"><FileText size={17} /></span>
          </div>
          <strong>{sentQuotes.length.toString().padStart(2, "0")}</strong>
          <small>Submitted to customers</small>
        </article>

        <article className="stat-card">
          <div className="stat-top">
            <span>Won RFQs</span>
            <span className="stat-icon"><TrendingUp size={17} /></span>
          </div>
          <strong>{wonRFQs.length.toString().padStart(2, "0")}</strong>
          <small>Successfully converted</small>
        </article>

        <article className="stat-card">
          <div className="stat-top">
            <span>Lost RFQs</span>
            <span className="stat-icon"><FileText size={17} /></span>
          </div>
          <strong>{lostRFQs.length.toString().padStart(2, "0")}</strong>
          <small>Unsuccessful opportunities</small>
        </article>
      </section>

      {error && <div className="global-error">{error}</div>}

      <section className="content-grid">
        <div className="panel recent-panel">
          <div className="panel-heading">
            <div>
              <h2>Recent quotations</h2>
              <p>Latest quotations across the workspace</p>
            </div>
            <button className="text-button" onClick={() => setPage("quotes")}>
              View all <ArrowRight size={15} />
            </button>
          </div>

          {quotesLoading ? (
            <div className="empty-state">Loading quotations...</div>
          ) : recentQuotes.length ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>QUOTE</th>
                    <th>CUSTOMER</th>
                    <th>ANNUAL VALUE</th>
                    <th>STATUS</th>
                  </tr>
                </thead>
                <tbody>
                  {recentQuotes.map((quote) => (
                    <tr key={quote.id || quote._id}>
                      <td>
                        <strong>{quote.quote_number || "—"}</strong>
                        <small>{quote.rfq_number || "RFQ"}</small>
                      </td>
                      <td>{quote.customer_name || "—"}</td>
                      <td>{money(quote.annual_value)}</td>
                      <td><StatusBadge status={quote.status} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty-state">
              <FileText size={22} />
              <strong>No quotations found</strong>
              <span>Created quotations will appear here.</span>
            </div>
          )}
        </div>

        <aside className="panel side-panel">
          <div className="panel-heading">
            <div>
              <h2>Sales pipeline</h2>
              <p>Current RFQ distribution</p>
            </div>
          </div>

          <div className="pipeline-list">
            <div className="pipeline-row">
              <span>Active RFQs</span>
              <strong>{activeRFQs.length}</strong>
            </div>
            <div className="pipeline-row">
              <span>Won</span>
              <strong>{wonRFQs.length}</strong>
            </div>
            <div className="pipeline-row">
              <span>Lost</span>
              <strong>{lostRFQs.length}</strong>
            </div>
          </div>

          <div className="side-divider" />

          <button className="quick-action" onClick={() => setPage("quotes")}>
            <span className="quick-icon"><FileText size={17} /></span>
            <span>
              <strong>Browse quotations</strong>
              <small>View generated quotes</small>
            </span>
            <ArrowUpRight size={16} />
          </button>

          <button className="quick-action" onClick={() => setPage("rfqs")}>
            <span className="quick-icon"><TrendingUp size={17} /></span>
            <span>
              <strong>View RFQ pipeline</strong>
              <small>{rfqs.length} registered RFQs</small>
            </span>
            <ArrowUpRight size={16} />
          </button>
        </aside>
      </section>
    </>
  );
}
