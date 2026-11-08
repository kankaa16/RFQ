
import React from "react";
import {
  Calculator,
  RefreshCw,
  Info,
  FileText,
  CheckCircle2,
  Plus,
  Trash2,
  Settings2
} from "lucide-react";
import { api, apiRequest } from "../services/api.js";

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

function newOperation() {
  return {
    name: "",
    machine_type: "",
    cycle_time_minutes: "",
    setup_time_minutes: 0,
    batch_size: 1000,
    hour_rate: "",
    rate_master_id: ""
  };
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

  const [routingOperations, setRoutingOperations] = React.useState([]);
  const [routingVerified, setRoutingVerified] = React.useState(false);
  const [routingLoading, setRoutingLoading] = React.useState(false);
  const [routingSaving, setRoutingSaving] = React.useState(false);

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

  async function loadRouting(id) {
    setRoutingLoading(true);
    setRoutingVerified(false);
    setRoutingOperations([]);

    try {
      const response = await apiRequest(
        `/api/parts/${id}/routing`
      );

      const routing =
        response.data?.routing ||
        response.data ||
        response.routing ||
        response;

      const operations = routing.operations || [];

      setRoutingOperations(
        operations.length
          ? operations.map((operation) => ({
              name: operation.name || "",
              machine_type: operation.machine_type || "",
              cycle_time_minutes: operation.cycle_time_minutes ?? "",
              setup_time_minutes: operation.setup_time_minutes ?? 0,
              batch_size: operation.batch_size ?? 1000,
              hour_rate: operation.hour_rate ?? "",
              rate_master_id: getId(operation.rate_master_id)
            }))
          : [newOperation()]
      );

      setRoutingVerified(Boolean(routing.is_verified));
    } catch (e) {
      if (
        e.message?.toLowerCase().includes("not found") ||
        e.message?.toLowerCase().includes("no documents")
      ) {
        setRoutingOperations([newOperation()]);
        setRoutingVerified(false);
      } else {
        setError(`Could not load routing: ${e.message}`);
        setRoutingOperations([newOperation()]);
      }
    } finally {
      setRoutingLoading(false);
    }
  }

  async function chooseRFQ(id) {
    setRfqId(id);
    setPartId("");
    setParts([]);
    setRoutingOperations([]);
    setRoutingVerified(false);
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

  async function choosePart(id) {
    setPartId(id);
    setResult(null);
    setQuote(null);
    setError("");
    setSuccess("");
    setRoutingVerified(false);
    setRoutingOperations([]);

    if (id) {
      await loadRouting(id);
    }
  }

  function updateOperation(index, field, value) {
    setRoutingOperations((current) =>
      current.map((operation, i) =>
        i === index
          ? { ...operation, [field]: value }
          : operation
      )
    );

    setRoutingVerified(false);
    setResult(null);
    setQuote(null);
  }

  function addOperation() {
    setRoutingOperations((current) => [
      ...current,
      newOperation()
    ]);
    setRoutingVerified(false);
    setResult(null);
  }

  function removeOperation(index) {
    setRoutingOperations((current) =>
      current.filter((_, i) => i !== index)
    );
    setRoutingVerified(false);
    setResult(null);
  }

  function validateRouting() {
    if (!routingOperations.length) {
      throw new Error("Add at least one manufacturing operation.");
    }

    for (let i = 0; i < routingOperations.length; i++) {
      const operation = routingOperations[i];

      if (!operation.name.trim()) {
        throw new Error(`Enter the name for operation ${i + 1}.`);
      }

      if (!operation.machine_type.trim()) {
        throw new Error(`Enter the machine type for operation ${i + 1}.`);
      }

      if (
        Number(operation.cycle_time_minutes) <= 0 ||
        !Number.isFinite(Number(operation.cycle_time_minutes))
      ) {
        throw new Error(`Enter a valid cycle time for operation ${i + 1}.`);
      }

      if (
        Number(operation.setup_time_minutes) < 0 ||
        !Number.isFinite(Number(operation.setup_time_minutes))
      ) {
        throw new Error(`Enter a valid setup time for operation ${i + 1}.`);
      }

      if (
        Number(operation.batch_size) <= 0 ||
        !Number.isInteger(Number(operation.batch_size))
      ) {
        throw new Error(`Enter a valid batch size for operation ${i + 1}.`);
      }

      if (
        Number(operation.hour_rate) <= 0 ||
        !Number.isFinite(Number(operation.hour_rate))
      ) {
        throw new Error(`Enter a valid hourly rate for operation ${i + 1}.`);
      }
    }
  }

  async function saveRouting() {
    if (!partId) {
      throw new Error("Select a part first.");
    }

    if (routingVerified) return;

    validateRouting();

    setRoutingSaving(true);

    try {
      const operations = routingOperations.map((operation, index) => ({
        sequence: index + 1,
        name: operation.name.trim(),
        machine_type: operation.machine_type.trim(),
        cycle_time_minutes: Number(operation.cycle_time_minutes),
        setup_time_minutes: Number(operation.setup_time_minutes),
        batch_size: Number(operation.batch_size),
        hour_rate: Number(operation.hour_rate),
        ...(operation.rate_master_id
          ? { rate_master_id: operation.rate_master_id }
          : {})
      }));

      const response = await apiRequest(
        `/api/parts/${partId}/routing`,
        {
          method: "POST",
          body: JSON.stringify({
            part_id: partId,
            source: "manual",
            operations
          })
        }
      );

      const savedRouting =
        response.data?.routing ||
        response.data ||
        response;

      if (
        savedRouting.is_verified === false ||
        response.success === false
      ) {
        throw new Error("Backend did not verify the saved routing.");
      }

      setRoutingVerified(true);
      setSuccess("Manufacturing routing saved and verified.");
    } finally {
      setRoutingSaving(false);
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

      if (!partId) {
        throw new Error("Select a part first.");
      }

      // save and verify routing before calculating
      if (!routingVerified) {
        await saveRouting();
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
      setSuccess("Costing calculated successfully.");
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
      setSuccess(response.message || "Quote created successfully.");
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
            onChange={(e) => choosePart(e.target.value)}
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

          <div className="panel-heading">
            <div>
              <h2>Manufacturing routing</h2>
              <p>
                Define the operations required to manufacture this part.
              </p>
            </div>
            <Settings2 size={19} />
          </div>

          {routingLoading ? (
            <div className="inline-info">
              Loading routing...
            </div>
          ) : (
            <>
              {routingVerified && (
                <div className="inline-info">
                  <CheckCircle2 size={15} />
                  Verified routing is available for this part.
                </div>
              )}

              {!routingVerified && (
                <div className="inline-info">
                  <Info size={15} />
                  Enter the actual manufacturing operations and machine
                  rates. Routing must be verified before costing.
                </div>
              )}

              {routingOperations.map((operation, index) => (
                <div
                  className="selected-part"
                  key={index}
                  style={{ marginBottom: 14 }}
                >
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      marginBottom: 12
                    }}
                  >
                    <strong>Operation {index + 1}</strong>

                    {!routingVerified && (
                      <button
                        type="button"
                        className="secondary-button"
                        onClick={() => removeOperation(index)}
                        disabled={routingOperations.length === 1}
                        title="Remove operation"
                      >
                        <Trash2 size={15} />
                      </button>
                    )}
                  </div>

                  <div className="input-grid">
                    <div>
                      <label>Operation name</label>
                      <input
                        value={operation.name}
                        disabled={routingVerified}
                        onChange={(e) =>
                          updateOperation(index, "name", e.target.value)
                        }
                        placeholder="CNC Turning"
                      />
                    </div>

                    <div>
                      <label>Machine type</label>
                      <input
                        value={operation.machine_type}
                        disabled={routingVerified}
                        onChange={(e) =>
                          updateOperation(index, "machine_type", e.target.value)
                        }
                        placeholder="CNC Lathe"
                      />
                    </div>

                    <div>
                      <label>Cycle time (minutes)</label>
                      <input
                        type="number"
                        min="0.01"
                        step="0.01"
                        value={operation.cycle_time_minutes}
                        disabled={routingVerified}
                        onChange={(e) =>
                          updateOperation(
                            index,
                            "cycle_time_minutes",
                            e.target.value
                          )
                        }
                      />
                    </div>

                    <div>
                      <label>Setup time (minutes)</label>
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={operation.setup_time_minutes}
                        disabled={routingVerified}
                        onChange={(e) =>
                          updateOperation(
                            index,
                            "setup_time_minutes",
                            e.target.value
                          )
                        }
                      />
                    </div>

                    <div>
                      <label>Batch size</label>
                      <input
                        type="number"
                        min="1"
                        step="1"
                        value={operation.batch_size}
                        disabled={routingVerified}
                        onChange={(e) =>
                          updateOperation(index, "batch_size", e.target.value)
                        }
                      />
                    </div>

                    <div>
                      <label>Machine rate (₹ / hour)</label>
                      <input
                        type="number"
                        min="0.01"
                        step="0.01"
                        value={operation.hour_rate}
                        disabled={routingVerified}
                        onChange={(e) =>
                          updateOperation(index, "hour_rate", e.target.value)
                        }
                      />
                    </div>
                  </div>
                </div>
              ))}

              {!routingVerified && (
                <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
                  <button
                    type="button"
                    className="secondary-button"
                    onClick={addOperation}
                    disabled={!partId || routingLoading || routingSaving}
                  >
                    <Plus size={15} /> Add operation
                  </button>

                  <button
                    type="button"
                    className="primary-button"
                    onClick={async () => {
                      setError("");
                      setSuccess("");
                      try {
                        await saveRouting();
                      } catch (e) {
                        setError(e.message);
                      }
                    }}
                    disabled={
                      !partId ||
                      routingSaving ||
                      routingLoading
                    }
                  >
                    {routingSaving ? "Saving..." : "Save & verify routing"}
                  </button>
                </div>
              )}
            </>
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
            disabled={
              loading ||
              routingLoading ||
              routingSaving ||
              !partId
            }
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
                Select a part, configure its routing and enter cost inputs
                to get started.
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
