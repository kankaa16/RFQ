
import { useCallback, useEffect, useState } from "react";
import { apiRequest } from "../services/api";

const formatCurrency = (value, currency = "INR") =>
  new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency,
    maximumFractionDigits: 2,
  }).format(Number(value || 0));

const formatDate = (value) => {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? "—"
    : date.toLocaleDateString("en-IN", {
        day: "2-digit",
        month: "short",
        year: "numeric",
      });
};

export default function HistoricalQuotes() {
  const [quotes, setQuotes] = useState([]);
  const [search, setSearch] = useState("");
  const [material, setMaterial] = useState("");
  const [process, setProcess] = useState("");
  const [customer, setCustomer] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const fetchQuotes = useCallback(async () => {
    setLoading(true);
    setError("");

    try {
      const params = new URLSearchParams();

      if (search.trim()) params.set("search", search.trim());
      if (material.trim()) params.set("material", material.trim());
      if (process.trim()) params.set("process", process.trim());
      if (customer.trim()) params.set("customer", customer.trim());

      const query = params.toString();
      const result = await apiRequest(
        `/api/historical-quotes${query ? `?${query}` : ""}`
      );

      setQuotes(result.data || []);
    } catch (err) {
      setError(err.message || "Could not load historical quotes.");
      setQuotes([]);
    } finally {
      setLoading(false);
    }
  }, [search, material, process, customer]);

  useEffect(() => {
    fetchQuotes();
  }, [fetchQuotes]);

  const clearFilters = () => {
    setSearch("");
    setMaterial("");
    setProcess("");
    setCustomer("");
  };

  const totalRecords = quotes.length;
  const averagePrice = totalRecords
    ? quotes.reduce((sum, quote) => sum + Number(quote.selling_price || 0), 0) /
      totalRecords
    : 0;

  return (
    <div className="page-container">
      <div className="page-header">
        <div>
          <p className="eyebrow">COST INTELLIGENCE</p>
          <h1>Historical quotes</h1>
          <p className="page-description">
            Search previous customer quotations and compare historical pricing.
          </p>
        </div>

        <button
          className="secondary-button"
          onClick={fetchQuotes}
          disabled={loading}
        >
          ↻ Refresh
        </button>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-label">Matching records</span>
          <strong>{totalRecords}</strong>
        </div>

        <div className="stat-card">
          <span className="stat-label">Average selling price</span>
          <strong>{formatCurrency(averagePrice)}</strong>
        </div>

        <div className="stat-card">
          <span className="stat-label">Latest quote date</span>
          <strong>
            {quotes.length
              ? formatDate(
                  quotes.reduce((latest, quote) =>
                    new Date(quote.quoted_at) > new Date(latest.quoted_at)
                      ? quote
                      : latest
                  ).quoted_at
                )
              : "—"}
          </strong>
        </div>
      </div>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h2>Search archive</h2>
            <p>Filter by part, material, process, or customer.</p>
          </div>

          <button className="text-button" onClick={clearFilters}>
            Clear filters
          </button>
        </div>

        <div className="filter-grid">
          <input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search part name, number, quote..."
          />

          <input
            value={material}
            onChange={(event) => setMaterial(event.target.value)}
            placeholder="Material grade"
          />

          <input
            value={process}
            onChange={(event) => setProcess(event.target.value)}
            placeholder="Manufacturing process"
          />

          <input
            value={customer}
            onChange={(event) => setCustomer(event.target.value)}
            placeholder="Customer name"
          />
        </div>

        <div className="filter-actions">
          <button
            className="primary-button"
            onClick={fetchQuotes}
            disabled={loading}
          >
            Apply filters
          </button>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h2>Quote archive</h2>
            <p>Showing up to 100 matching historical records.</p>
          </div>
          <span className="record-count">{totalRecords} records</span>
        </div>

        {error && <div className="error-message">{error}</div>}

        {loading ? (
          <div className="empty-state">Loading historical quotes...</div>
        ) : quotes.length === 0 ? (
          <div className="empty-state">
            <h3>No historical quotes found</h3>
            <p>Try changing your filters or import historical quote data.</p>
          </div>
        ) : (
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Quote reference</th>
                  <th>Customer</th>
                  <th>Part</th>
                  <th>Material</th>
                  <th>Process</th>
                  <th>Annual volume</th>
                  <th>Unit price</th>
                  <th>Quoted on</th>
                </tr>
              </thead>

              <tbody>
                {quotes.map((quote) => (
                  <tr key={quote.id || quote._id}>
                    <td>
                      <strong>{quote.original_quote_number || "—"}</strong>
                    </td>
                    <td>{quote.customer_name || "—"}</td>
                    <td>
                      <strong>{quote.part_name || "—"}</strong>
                      <small>{quote.part_number || ""}</small>
                    </td>
                    <td>{quote.material_grade || "—"}</td>
                    <td>{quote.manufacturing_process || "—"}</td>
                    <td>
                      {Number(quote.annual_volume || 0).toLocaleString("en-IN")}
                    </td>
                    <td>
                      <strong>
                        {formatCurrency(
                          quote.selling_price,
                          quote.currency || "INR"
                        )}
                      </strong>
                    </td>
                    <td>{formatDate(quote.quoted_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
