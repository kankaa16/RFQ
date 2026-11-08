
import React from "react";
import {
  RefreshCw,
  Search,
  Plus,
  X,
  History,
  Clock3,
  ArrowUpRight,
  LoaderCircle
} from "lucide-react";
import { apiRequest } from "../services/api";

const initialForm = {
  rate_code: "",
  rate_name: "",
  rate_type: "machine",
  category: "",
  material_grade: "",
  machine_type: "",
  rate: "",
  unit: "",
  currency: "INR",
  effective_from: ""
};

const rateTypes = [
  "material",
  "machine",
  "labour",
  "overhead",
  "packing",
  "freight"
];

function formatDate(value) {
  if (!value || value.startsWith?.("0001-01-01")) return "—";

  return new Date(value).toLocaleDateString("en-IN", {
    day: "2-digit",
    month: "short",
    year: "numeric"
  });
}

function formatMoney(value) {
  return Number(value || 0).toLocaleString("en-IN", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  });
}

export default function Rates({ rates = [], loading, onRefresh }) {
  const [query, setQuery] = React.useState("");
  const [typeFilter, setTypeFilter] = React.useState("all");
  const [statusFilter, setStatusFilter] = React.useState("all");

  const [showForm, setShowForm] = React.useState(false);
  const [form, setForm] = React.useState(initialForm);
  const [formError, setFormError] = React.useState("");
  const [saving, setSaving] = React.useState(false);
  const [formMode, setFormMode] = React.useState("create");

  const [historyCode, setHistoryCode] = React.useState("");
  const [history, setHistory] = React.useState([]);
  const [historyLoading, setHistoryLoading] = React.useState(false);
  const [historyError, setHistoryError] = React.useState("");

  const safeRates = Array.isArray(rates) ? rates : [];

  const filtered = safeRates.filter((rate) => {
    const searchable = [
      rate.rate_code,
      rate.rate_name,
      rate.rate_type,
      rate.category,
      rate.machine_type,
      rate.material_grade
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();

    const matchesQuery = searchable.includes(query.toLowerCase());
    const matchesType =
      typeFilter === "all" || rate.rate_type === typeFilter;

    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "active" && rate.is_active) ||
      (statusFilter === "inactive" && !rate.is_active);

    return matchesQuery && matchesType && matchesStatus;
  });

  function updateForm(field, value) {
    setForm((current) => ({
      ...current,
      [field]: value
    }));
  }

  function openCreateForm() {
    setForm(initialForm);
    setFormMode("create");
    setFormError("");
    setShowForm(true);
  }

  function openVersionForm(rate) {
    setForm({
      rate_code: rate.rate_code || "",
      rate_name: rate.rate_name || "",
      rate_type: rate.rate_type || "machine",
      category: rate.category || "",
      material_grade: rate.material_grade || "",
      machine_type: rate.machine_type || "",
      rate: String(rate.rate ?? ""),
      unit: rate.unit || "",
      currency: rate.currency || "INR",
      effective_from: ""
    });

    setFormMode("version");
    setFormError("");
    setShowForm(true);
  }

  async function submitRate(event) {
    event.preventDefault();
    setFormError("");

    if (
      !form.rate_code.trim() ||
      !form.rate_name.trim() ||
      !form.category.trim() ||
      !form.unit.trim() ||
      !form.effective_from ||
      Number(form.rate) <= 0
    ) {
      setFormError("Please complete all required fields with a valid rate.");
      return;
    }

    if (
      form.rate_type === "machine" &&
      !form.machine_type.trim()
    ) {
      setFormError("Machine type is required for machine rates.");
      return;
    }

    if (
      form.rate_type === "material" &&
      !form.material_grade.trim()
    ) {
      setFormError("Material grade is required for material rates.");
      return;
    }

    setSaving(true);

    try {
      const payload = {
        rate_code: form.rate_code.trim(),
        rate_name: form.rate_name.trim(),
        rate_type: form.rate_type,
        category: form.category.trim(),
        material_grade: form.material_grade.trim(),
        machine_type: form.machine_type.trim(),
        rate: Number(form.rate),
        unit: form.unit.trim(),
        currency: form.currency,
        effective_from: new Date(
          `${form.effective_from}T00:00:00.000Z`
        ).toISOString()
      };

      await apiRequest("/api/rates", {
        method: "POST",
        body: JSON.stringify(payload)
      });

      setShowForm(false);
      setForm(initialForm);
      await onRefresh?.();
    } catch (error) {
      setFormError(error.message || "Could not save rate.");
    } finally {
      setSaving(false);
    }
  }

  async function openHistory(rateCode) {
    setHistoryCode(rateCode);
    setHistory([]);
    setHistoryError("");
    setHistoryLoading(true);

    try {
      const response = await apiRequest(
        `/api/rates/${encodeURIComponent(rateCode)}/history`
      );

      setHistory(response.data || []);
    } catch (error) {
      setHistoryError(error.message || "Could not load rate history.");
    } finally {
      setHistoryLoading(false);
    }
  }

  const activeCount = safeRates.filter((rate) => rate.is_active).length;

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow"><i /> CONFIGURATION</span>
          <h1>Rate masters</h1>
          <p>Manage manufacturing rates and their effective versions.</p>
        </div>

        <div className="rate-heading-actions">
          <button
            className="secondary-button"
            onClick={onRefresh}
            disabled={loading}
          >
            <RefreshCw size={15} />
            Refresh
          </button>

          <button className="primary-button" onClick={openCreateForm}>
            <Plus size={16} />
            Add rate
          </button>
        </div>
      </div>

      <div className="rate-summary-grid">
        <div className="rate-summary-card">
          <span>Total loaded rates</span>
          <strong>{safeRates.length}</strong>
          <small>Rate records currently available</small>
        </div>

        <div className="rate-summary-card">
          <span>Active rates</span>
          <strong>{activeCount}</strong>
          <small>Currently active versions</small>
        </div>

        <div className="rate-summary-card">
          <span>Rate categories</span>
          <strong>{new Set(safeRates.map((rate) => rate.rate_type)).size}</strong>
          <small>Across loaded rate records</small>
        </div>
      </div>

      <div className="panel rate-panel">
        <div className="rate-toolbar">
          <div className="search-box">
            <Search size={16} />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search code, name, machine or material..."
            />
          </div>

          <select
            value={typeFilter}
            onChange={(event) => setTypeFilter(event.target.value)}
            className="rate-filter"
          >
            <option value="all">All types</option>
            {rateTypes.map((type) => (
              <option value={type} key={type}>
                {type.charAt(0).toUpperCase() + type.slice(1)}
              </option>
            ))}
          </select>

          <select
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
            className="rate-filter"
          >
            <option value="all">All statuses</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>

          <span className="result-count">{filtered.length} rates</span>
        </div>

        {loading ? (
          <div className="empty-state">Loading rates...</div>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>RATE DETAILS</th>
                  <th>TYPE</th>
                  <th>VALUE</th>
                  <th>UNIT</th>
                  <th>VERSION</th>
                  <th>EFFECTIVE FROM</th>
                  <th>STATUS</th>
                  <th>ACTIONS</th>
                </tr>
              </thead>

              <tbody>
                {filtered.map((rate) => (
                  <tr key={rate.id}>
                    <td>
                      <div className="rate-code-cell">
                        <strong>{rate.rate_code}</strong>
                        <small>{rate.rate_name || rate.category || "—"}</small>
                      </div>
                    </td>

                    <td>
                      <span className="rate-type-label">
                        {rate.rate_type || "—"}
                      </span>
                    </td>

                    <td className="rate-value">
                      ₹{formatMoney(rate.rate)}
                    </td>

                    <td>{rate.unit || "—"}</td>
                    <td>v{rate.version || 1}</td>
                    <td>{formatDate(rate.effective_from)}</td>

                    <td>
                      <span
                        className={`status-badge ${
                          rate.is_active ? "green" : "neutral"
                        }`}
                      >
                        <span />
                        {rate.is_active ? "active" : "inactive"}
                      </span>
                    </td>

                    <td>
                      <div className="rate-row-actions">
                        <button
                          className="rate-icon-button"
                          title="View version history"
                          onClick={() => openHistory(rate.rate_code)}
                        >
                          <History size={15} />
                        </button>

                        <button
                          className="rate-text-button"
                          onClick={() => openVersionForm(rate)}
                        >
                          New version
                          <ArrowUpRight size={13} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {!filtered.length && (
              <div className="empty-state">
                No matching rate masters found.
              </div>
            )}
          </div>
        )}

        <div className="table-footnote">
          Creating a new version preserves previous rates and their effective
          dates. Existing versions are not overwritten.
        </div>
      </div>

      {showForm && (
        <div
          className="rate-modal-backdrop"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget && !saving) {
              setShowForm(false);
            }
          }}
        >
          <div className="rate-modal">
            <div className="rate-modal-header">
              <div>
                <span className="eyebrow">
                  <i /> RATE CONFIGURATION
                </span>
                <h2>
                  {formMode === "version" ? "Create new version" : "Add rate"}
                </h2>
                <p>
                  {formMode === "version"
                    ? "This creates a new version and closes the previous one."
                    : "Add a new manufacturing rate to the master."}
                </p>
              </div>

              <button
                className="rate-icon-button"
                onClick={() => setShowForm(false)}
                disabled={saving}
              >
                <X size={18} />
              </button>
            </div>

            <form onSubmit={submitRate}>
              <div className="rate-form-grid">
                <label>
                  Rate code *
                  <input
                    value={form.rate_code}
                    onChange={(event) => updateForm("rate_code", event.target.value)}
                    disabled={formMode === "version"}
                    placeholder="e.g. MACHINE-CNC-001"
                  />
                </label>

                <label>
                  Rate name *
                  <input
                    value={form.rate_name}
                    onChange={(event) => updateForm("rate_name", event.target.value)}
                    placeholder="e.g. CNC Lathe"
                  />
                </label>

                <label>
                  Rate type *
                  <select
                    value={form.rate_type}
                    onChange={(event) => updateForm("rate_type", event.target.value)}
                  >
                    {rateTypes.map((type) => (
                      <option value={type} key={type}>
                        {type.charAt(0).toUpperCase() + type.slice(1)}
                      </option>
                    ))}
                  </select>
                </label>

                <label>
                  Category *
                  <input
                    value={form.category}
                    onChange={(event) => updateForm("category", event.target.value)}
                    placeholder="e.g. machining"
                  />
                </label>

                {form.rate_type === "machine" && (
                  <label>
                    Machine type *
                    <input
                      value={form.machine_type}
                      onChange={(event) => updateForm("machine_type", event.target.value)}
                      placeholder="e.g. CNC Lathe"
                    />
                  </label>
                )}

                {form.rate_type === "material" && (
                  <label>
                    Material grade *
                    <input
                      value={form.material_grade}
                      onChange={(event) => updateForm("material_grade", event.target.value)}
                      placeholder="e.g. EN8"
                    />
                  </label>
                )}

                <label>
                  Rate value *
                  <input
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={form.rate}
                    onChange={(event) => updateForm("rate", event.target.value)}
                    placeholder="e.g. 950"
                  />
                </label>

                <label>
                  Unit *
                  <input
                    value={form.unit}
                    onChange={(event) => updateForm("unit", event.target.value)}
                    placeholder="e.g. INR/hour"
                  />
                </label>

                <label>
                  Currency
                  <select
                    value={form.currency}
                    onChange={(event) => updateForm("currency", event.target.value)}
                  >
                    <option value="INR">INR - Indian Rupee</option>
                  </select>
                </label>

                <label>
                  Effective from *
                  <input
                    type="date"
                    value={form.effective_from}
                    onChange={(event) => updateForm("effective_from", event.target.value)}
                  />
                </label>
              </div>

              {formError && (
                <div className="rate-form-error">{formError}</div>
              )}

              <div className="rate-modal-footer">
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => setShowForm(false)}
                  disabled={saving}
                >
                  Cancel
                </button>

                <button
                  type="submit"
                  className="primary-button"
                  disabled={saving}
                >
                  {saving ? (
                    <>
                      <LoaderCircle size={15} className="rate-spinner" />
                      Saving...
                    </>
                  ) : (
                    <>
                      <Plus size={15} />
                      {formMode === "version" ? "Create version" : "Save rate"}
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {historyCode && (
        <div className="rate-modal-backdrop" onMouseDown={(event) => {
          if (event.target === event.currentTarget) setHistoryCode("");
        }}>
          <div className="rate-modal rate-history-modal">
            <div className="rate-modal-header">
              <div>
                <span className="eyebrow">
                  <i /> VERSION HISTORY
                </span>
                <h2>{historyCode}</h2>
                <p>All saved versions of this rate.</p>
              </div>

              <button
                className="rate-icon-button"
                onClick={() => setHistoryCode("")}
              >
                <X size={18} />
              </button>
            </div>

            {historyLoading ? (
              <div className="empty-state">Loading rate history...</div>
            ) : historyError ? (
              <div className="rate-form-error">{historyError}</div>
            ) : history.length === 0 ? (
              <div className="empty-state">No version history found.</div>
            ) : (
              <div className="rate-history-list">
                {history.map((item) => (
                  <div className="rate-history-item" key={item.id}>
                    <div className="rate-history-main">
                      <div>
                        <strong>Version {item.version}</strong>
                        <span
                          className={`status-badge ${
                            item.is_active ? "green" : "neutral"
                          }`}
                        >
                          <span />
                          {item.is_active ? "active" : "inactive"}
                        </span>
                      </div>
                      <strong className="rate-history-value">
                        ₹{formatMoney(item.rate)}
                        <small> / {item.unit}</small>
                      </strong>
                    </div>

                    <div className="rate-history-dates">
                      <span>
                        <Clock3 size={13} />
                        From: {formatDate(item.effective_from)}
                      </span>
                      <span>
                        <Clock3 size={13} />
                        To: {formatDate(item.effective_to)}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <div className="rate-modal-footer">
              <button
                className="secondary-button"
                onClick={() => setHistoryCode("")}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
