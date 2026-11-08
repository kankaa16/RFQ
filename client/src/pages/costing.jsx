
import React from "react";
import {
  Calculator,
  RefreshCw,
  Info,
  FileText,
  CheckCircle2
} from "lucide-react";
import { api } from "../services/api.js";

function getId(value) {
  if (!value) return "";
  if (typeof value === "string") return value;
  if (value.$oid) return value.$oid;
  if (value.hex) return value.hex;
  return "";
}

function formatCurrency(value) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 2
  }).format(Number(value || 0));
}

export default function Costing({ rfqs, onLoadParts, onCalculate }) {
  const [rfqId, setRfqId] = React.useState("");
  const [parts, setParts] = React.useState([]);
  const [partId, setPartId] = React.useState("");

  const [form, setForm] = React.useState({
    material_rate: 100,
    scrap_rate: 0,
    overhead_rate: 10,
    packing_cost: 2,
    freight_cost: 3,
    margin_rate: 15,
    volumes: "180000,90000,45000"
  });

  const [gstPercent, setGstPercent] = React.useState(18);
  const [result, setResult] = React.useState(null);
  const [quote, setQuote] = React.useState(null);

  const [loading, setLoading] = React.useState(false);
  const [quoteLoading, setQuoteLoading] = React.useState(false);
  const [error, setError] = React.useState("");
  const [success, setSuccess] = React.useState("");

  const selectedRFQ = rfqs.find(
    (r) => getId(r.id || r._id) === rfqId
  );

  const selectedPart = parts.find(
    (p) => getId(p.id || p._id) === partId
  );

  const costingData = result?.data || result;
  const scenarios = costingData?.scenarios || [];

  async function chooseRFQ(id) {
    setRfqId(id);
    setPartId("");
    setParts([]);
    setResult(null);
    setQuote(null);
    setError("");
    setSuccess("");

    if (!id) return;

    try {
      const loadedParts = await onLoadParts(id);
      setParts(loadedParts);
    } catch (e) {
      setError(e.message);
    }
  }

  async function calculate(e) {
    e.preventDefault();

    setLoading(true);
    setError("");
    setSuccess("");
    setResult(null);
    setQuote(null);

    try {
      const volumes = form.volumes
        .split(",")
        .map((x) => Number(x.trim()))
        .filter((x) => Number.isInteger(x) && x > 0);

      if (!volumes.length || volumes.length > 3) {
        throw new Error("Enter between 1 and 3 valid annual volumes.");
      }

      const payload = {
        part_id: partId,
        material_rate: Number(form.material_rate),
        scrap_rate: Number(form.scrap_rate),
        overhead_rate: Number(form.overhead_rate),
        packing_cost: Number(form.packing_cost),
        freight_cost: Number(form.freight_cost),
        margin_rate: Number(form.margin_rate),
        volumes
      };

      const response = await onCalculate(payload);
      setResult(response);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  async function createQuote() {
    if (!result || !selectedRFQ || !selectedPart) return;

    setQuoteLoading(true);
    setError("");
    setSuccess("");

    try {
      const costingId = getId(costingData.id);
      const currentRFQId = getId(selectedRFQ.id || selectedRFQ._id);

      const customerId = getId(
        selectedRFQ.customer_id ||
        selectedRFQ.customerId ||
        selectedRFQ.customer?.id ||
        selectedRFQ.customer?._id ||
        selectedRFQ.customer
      );

      if (!costingId) {
        throw new Error("Costing ID is missing from the calculation response.");
      }

      if (!customerId) {
        throw new Error(
          "Customer ID is missing from the selected RFQ. Check the RFQ API response."
        );
      }

      const response = await api.createQuote({
        rfq_id: currentRFQId,
        customer_id: customerId,
        costing_ids: [costingId],
        gst_percent: Number(gstPercent),
        payment_terms: "As mutually agreed",
        delivery_terms: "As per purchase order",
        validity_days: 30,
        delivery_location: "As specified by customer"
      });

      const createdQuote = response.data || response;

      setQuote(createdQuote);
      setSuccess(
        response.message || "Quote created successfully."
      );
    } catch (e) {
      setError(e.message);
    } finally {
      setQuoteLoading(false);
    }
  }

  const update = (e) => {
    setForm((current) => ({
      ...current,
      [e.target.name]: e.target.value
    }));
  };

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">
            <i /> COST ENGINE
          </span>
          <h1>Part costing</h1>
          <p>
            Calculate unit economics and generate customer quotations.
          </p>
        </div>
      </div>

      <div className="costing-grid">
        <form className="panel costing-form" onSubmit={calculate}>
          <div className="panel-heading">
            <div>
              <h2>Costing inputs</h2>
              <p>Choose an RFQ and part, then enter applicable rates.</p>
            </div>
            <Calculator size={19} />
          </div>

          <label>RFQ</label>
          <select
            value={rfqId}
            onChange={(e) => chooseRFQ(e.target.value)}
            required
          >
            <option value="">Select an RFQ</option>
            {rfqs.map((r) => {
              const id = getId(r.id || r._id);
              return (
                <option value={id} key={id}>
                  {r.rfq_number} · {r.customer_name || r.customer?.name || "Customer"}
                </option>
              );
            })}
          </select>

          <label>Part</label>
          <select
            value={partId}
            onChange={(e) => {
              setPartId(e.target.value);
              setResult(null);
              setQuote(null);
              setError("");
              setSuccess("");
            }}
            required
            disabled={!parts.length}
          >
            <option value="">Select a part</option>
            {parts.map((p) => {
              const id = getId(p.id || p._id);
              return (
                <option value={id} key={id}>
                  {p.part_number} · {p.part_name}
                </option>
              );
            })}
          </select>

          {selectedPart && (
            <div className="selected-part">
              <strong>{selectedPart.part_name}</strong>
              <span>
                {selectedPart.material_grade} ·{" "}
                {selectedPart.gross_weight} kg gross weight
              </span>
            </div>
          )}

          <div className="form-divider" />

          <div className="form-section-label">
            COST INPUTS <span>INR</span>
          </div>

          <div className="input-grid">
            <div>
              <label>Material rate / kg</label>
              <input
                name="material_rate"
                type="number"
                min="0"
                step="0.01"
                value={form.material_rate}
                onChange={update}
              />
            </div>

            <div>
              <label>Scrap rate / kg</label>
              <input
                name="scrap_rate"
                type="number"
                min="0"
                step="0.01"
                value={form.scrap_rate}
                onChange={update}
              />
            </div>

            <div>
              <label>Overhead rate (%)</label>
              <input
                name="overhead_rate"
                type="number"
                min="0"
                max="100"
                step="0.01"
                value={form.overhead_rate}
                onChange={update}
              />
            </div>

            <div>
              <label>Margin (%)</label>
              <input
                name="margin_rate"
                type="number"
                min="0"
                max="99.99"
                step="0.01"
                value={form.margin_rate}
                onChange={update}
              />
            </div>

            <div>
              <label>Packing / part</label>
              <input
                name="packing_cost"
                type="number"
                min="0"
                step="0.01"
                value={form.packing_cost}
                onChange={update}
              />
            </div>

            <div>
              <label>Freight / part</label>
              <input
                name="freight_cost"
                type="number"
                min="0"
                step="0.01"
                value={form.freight_cost}
                onChange={update}
              />
            </div>
          </div>

          <label>Annual volume scenarios</label>
          <input
            name="volumes"
            value={form.volumes}
            onChange={update}
            placeholder="180000, 90000, 45000"
          />

          <label>GST (%)</label>
          <input
            type="number"
            min="0"
            max="100"
            step="0.01"
            value={gstPercent}
            onChange={(e) => setGstPercent(e.target.value)}
          />

          <div className="inline-info">
            <Info size={15} />
            Confirm material, scrap and GST rates against approved masters
            and the applicable tax classification.
          </div>

          {error && <div className="form-error">{error}</div>}

          {success && (
            <div className="inline-info">
              <CheckCircle2 size={15} />
              {success}
            </div>
          )}

          <button
            className="primary-button full-button"
            disabled={loading || !partId}
            type="submit"
          >
            {loading ? "Calculating..." : (
              <>
                Calculate costing <Calculator size={16} />
              </>
            )}
          </button>
        </form>

        <div className="panel result-panel">
          <div className="panel-heading">
            <div>
              <h2>Calculation result</h2>
              <p>Unit cost and selling price by scenario</p>
            </div>
            <RefreshCw size={17} />
          </div>

          {!result ? (
            <div className="result-empty">
              <span className="result-symbol">
                <Calculator size={24} />
              </span>
              <strong>Your costing results will appear here.</strong>
              <span>
                Select a part and enter its cost inputs to get started.
              </span>
            </div>
          ) : (
            <div className="result-content">
              {scenarios.map((scenario, index) => (
                <div className="result-highlight" key={index}>
                  <span>
                    ANNUAL VOLUME:{" "}
                    {Number(scenario.annual_volume).toLocaleString("en-IN")}
                  </span>
                  <strong>
                    {formatCurrency(scenario.selling_price)}
                  </strong>
                  <small>
                    Unit cost: {formatCurrency(scenario.total_cost)}
                  </small>
                </div>
              ))}

              <div className="form-divider" />

              <div className="inline-info">
                <Info size={15} />
                Costing has been saved. Create a quotation to submit it
                for approval if the annual value exceeds ₹25 lakh.
              </div>

              {!quote ? (
                <button
                  className="primary-button full-button"
                  onClick={createQuote}
                  disabled={quoteLoading}
                  type="button"
                >
                  {quoteLoading ? "Creating quote..." : (
                    <>
                      <FileText size={16} />
                      Create Quote
                    </>
                  )}
                </button>
              ) : (
                <div className="selected-part">
                  <strong>
                    <CheckCircle2 size={15} /> Quote created
                  </strong>
                  <span>
                    Quote: {quote.quote_number || quote.id || "Created"}
                  </span>
                  <span>
                    Status: {(quote.status || "created").replaceAll("_", " ")}
                  </span>
                  {quote.approval_required && (
                    <span>Sent for Sales Head approval.</span>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </>
  );
}
