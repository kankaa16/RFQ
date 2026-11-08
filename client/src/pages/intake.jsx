
import React from "react";
import {
  UploadCloud,
  FileText,
  Sparkles,
  RefreshCw,
  CheckCircle2,
  AlertCircle,
  ChevronDown,
  ChevronUp,
  ArrowRight,
  PencilLine
} from "lucide-react";
import { apiRequest } from "../services/api.js";

const ZERO_ID = "000000000000000000000000";

function unwrap(payload) {
  return payload?.data ?? payload ?? {};
}

function isValidId(value) {
  return (
    typeof value === "string" &&
    /^[a-fA-F0-9]{24}$/.test(value) &&
    value !== ZERO_ID
  );
}

// Only use extraction-specific identifiers here.
function getExtractionId(value) {
  if (!value || typeof value !== "object") return "";

  const candidates = [
    value.extraction_id,
    value.id,
    value._id
  ];

  return candidates.find(isValidId) || "";
}

// Only use document-specific identifiers here.
function getDocumentId(value) {
  if (!value || typeof value !== "object") return "";

  const candidates = [
    value.document_id,
    value.document?.id,
    value.document?._id,
    value.id,
    value._id
  ];

  return candidates.find(isValidId) || "";
}

function getFields(extraction) {
  const data = unwrap(extraction);

  return (
    data.extracted_fields ||
    data.fields ||
    data.extraction?.extracted_fields ||
    []
  );
}

function fieldName(field, index) {
  return (
    field.field_name ||
    field.name ||
    field.key ||
    `field_${index + 1}`
  );
}

function fieldValue(field) {
  const value =
    field.reviewed_value ??
    field.value ??
    field.extracted_value ??
    "";

  return typeof value === "string"
    ? value
    : JSON.stringify(value);
}

function prettyName(value) {
  return String(value)
    .replaceAll("_", " ")
    .replace(/\b\w/g, character => character.toUpperCase());
}

function isFieldAccepted(field) {
  return Boolean(
    field.is_accepted ||
    field.accepted ||
    field.review_status === "accepted" ||
    field.review_status === "edited"
  );
}

function getErrorMessage(error) {
  return error?.message || "Something went wrong. Please try again.";
}

export default function Intake({ onConverted }) {
  const [file, setFile] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState("");
  const [notice, setNotice] = React.useState("");

  const [documentId, setDocumentId] = React.useState("");
  const [extractionId, setExtractionId] = React.useState("");

  const [extraction, setExtraction] = React.useState(null);
  const [values, setValues] = React.useState({});
  const [accepted, setAccepted] = React.useState({});

  const [expanded, setExpanded] = React.useState(true);
  const [createdRfq, setCreatedRfq] = React.useState(null);

  const fields = getFields(extraction);

  const reviewedCount = fields.filter((field, index) => {
    const name = fieldName(field, index);
    return accepted[name] ?? isFieldAccepted(field);
  }).length;

  const allAccepted =
    fields.length > 0 && reviewedCount === fields.length;

  const extractionStatus = String(
    extraction?.status || ""
  ).toLowerCase();

  function reset() {
    setFile(null);
    setError("");
    setNotice("");

    setDocumentId("");
    setExtractionId("");

    setExtraction(null);
    setValues({});
    setAccepted({});

    setCreatedRfq(null);
  }

  async function uploadAndExtract(event) {
    event.preventDefault();

    if (!file) {
      setError("Please select an RFQ document first.");
      return;
    }

    setBusy(true);
    setError("");
    setNotice("");

    setExtraction(null);
    setCreatedRfq(null);
    setDocumentId("");
    setExtractionId("");
    setValues({});
    setAccepted({});

    try {
      const form = new FormData();

      form.append("file", file);
      form.append("category", "part_list");

      // Upload the document without an rfq_id.
      const uploaded = await apiRequest("/api/documents/upload", {
        method: "POST",
        body: form
      });

      const uploadData = unwrap(uploaded);

      const docId =
        getDocumentId(uploadData) ||
        getDocumentId(uploaded);

      if (!isValidId(docId)) {
        throw new Error(
          "Upload succeeded, but the backend did not return a valid document ID."
        );
      }

      setDocumentId(docId);

      // Extract document information using Gemini.
      const extracted = await apiRequest(
        `/api/ai-extractions/${docId}/extract`,
        { method: "POST" }
      );

      const extractionData = unwrap(extracted);

      const extractionObject =
        extractionData.extraction || extractionData;

      const extId =
        getExtractionId(extractionData) ||
        getExtractionId(extractionObject);

      if (!isValidId(extId)) {
        throw new Error(
          "Extraction completed, but the backend did not return a valid extraction ID."
        );
      }

      setExtractionId(extId);

      // Fetch the saved extraction using the document ID.
      const fetched = await apiRequest(
        `/api/ai-extractions/${docId}`
      );

      const fetchedData = unwrap(fetched);
      const savedExtraction =
        fetchedData.extraction || fetchedData;

      // Get only the extraction ID, never the document ID.
      const fetchedId =
        getExtractionId(fetchedData) ||
        getExtractionId(savedExtraction);

      if (!fetchedId) {
        throw new Error(
          "Could not identify the saved extraction record."
        );
      }

      if (fetchedId !== extId) {
        throw new Error(
          "The saved extraction ID does not match the extraction ID returned during upload."
        );
      }

      setExtraction(savedExtraction);

      const savedFields = getFields(savedExtraction);

      const initialValues = {};
      const initialAccepted = {};

      savedFields.forEach((field, index) => {
        const name = fieldName(field, index);

        initialValues[name] = fieldValue(field);
        initialAccepted[name] = isFieldAccepted(field);
      });

      setValues(initialValues);
      setAccepted(initialAccepted);

      setNotice(
        "AI extraction completed. Review every field before creating the RFQ."
      );
    } catch (e) {
      setError(getErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  function updateValue(name, value) {
    setValues(current => ({
      ...current,
      [name]: value
    }));

    setAccepted(current => ({
      ...current,
      [name]: false
    }));
  }

  function acceptField(name) {
    setAccepted(current => ({
      ...current,
      [name]: true
    }));
  }

  async function saveReview() {
    if (!isValidId(extractionId)) {
      throw new Error(
        "Invalid extraction ID. Please upload the document again."
      );
    }

    if (!fields.length) {
      throw new Error("No extracted fields were returned.");
    }

    // Avoid reviewing an extraction that is already reviewed.
    if (extractionStatus === "reviewed") {
      return;
    }

    const payloadFields = fields.map((field, index) => {
      const name = fieldName(field, index);
      const originalValue = fieldValue(field);
      const value = values[name] ?? originalValue;

      const isAccepted =
        accepted[name] ?? isFieldAccepted(field);

      const edited = value !== originalValue;

      return {
        field_name: name,
        action: !isAccepted
          ? "reject"
          : edited
            ? "edit"
            : "accept",
        value: edited ? value : ""
      };
    });

    if (payloadFields.some(field => field.action === "reject")) {
      throw new Error(
        "Please accept or edit every extracted field before saving."
      );
    }

    // This matches the backend FieldReviewRequest structure.
    await apiRequest(
      `/api/extractions/${extractionId}/review`,
      {
        method: "POST",
        body: JSON.stringify({
          fields: payloadFields
        })
      }
    );

    // Fetch the updated extraction after review.
    const refreshed = await apiRequest(
      `/api/ai-extractions/${documentId}`
    );

    const refreshedData = unwrap(refreshed);
    const refreshedExtraction =
      refreshedData.extraction || refreshedData;

    const refreshedId =
      getExtractionId(refreshedData) ||
      getExtractionId(refreshedExtraction);

    if (refreshedId !== extractionId) {
      throw new Error(
        "The refreshed extraction ID does not match."
      );
    }

    setExtraction(refreshedExtraction);

    const refreshedFields = getFields(refreshedExtraction);

    const refreshedValues = {};
    const refreshedAccepted = {};

    refreshedFields.forEach((field, index) => {
      const name = fieldName(field, index);

      refreshedValues[name] = fieldValue(field);
      refreshedAccepted[name] = isFieldAccepted(field);
    });

    setValues(refreshedValues);
    setAccepted(refreshedAccepted);
  }

  async function handleSaveReview() {
    setBusy(true);
    setError("");
    setNotice("");

    try {
      await saveReview();

      setNotice(
        "Review saved successfully. You can now create the RFQ."
      );
    } catch (e) {
      setError(getErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  async function convert() {
    setBusy(true);
    setError("");
    setNotice("");

    try {
      if (!allAccepted) {
        throw new Error(
          "Please accept every extracted field before creating the RFQ."
        );
      }

      // Save review only if it has not already been completed.
      await saveReview();

      const response = await apiRequest(
        `/api/extractions/${extractionId}/convert`,
        { method: "POST" }
      );

      const result = unwrap(response);

      const rfqId =
        result.rfq_id ||
        result.created_rfq_id ||
        result.rfq?.id ||
        result.rfq?._id ||
        "";

      const rfqNumber =
        result.rfq_number ||
        result.rfq?.rfq_number ||
        "New RFQ";

      setCreatedRfq({
        id: rfqId,
        number: rfqNumber
      });

      setNotice(
        "RFQ created successfully from the reviewed extraction."
      );

      onConverted?.();
    } catch (e) {
      setError(getErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div className="page-heading">
        <div>
          <span className="eyebrow">
            <i /> AI-ASSISTED INTAKE
          </span>

          <h1>RFQ intake</h1>

          <p>
            Upload customer documents, extract details with AI,
            and review them before creating an RFQ.
          </p>
        </div>

        <button
          className="secondary-button"
          onClick={reset}
          disabled={busy}
        >
          <RefreshCw size={15} />
          New intake
        </button>
      </div>

      <div className="intake-steps">
        <div className={`intake-step ${!extraction ? "current" : "done"}`}>
          <span>01</span>
          <div>
            <strong>Upload document</strong>
            <small>PDF, Excel, CSV</small>
          </div>
        </div>

        <div className={`intake-step ${extraction ? "current" : ""}`}>
          <span>02</span>
          <div>
            <strong>Review extraction</strong>
            <small>Verify AI-suggested fields</small>
          </div>
        </div>

        <div className={`intake-step ${createdRfq ? "done" : ""}`}>
          <span>03</span>
          <div>
            <strong>Create RFQ</strong>
            <small>Send reviewed data to register</small>
          </div>
        </div>
      </div>

      <section className="panel intake-upload-panel">
        <div className="panel-heading">
          <div>
            <h2>Upload RFQ documents</h2>
            <p>Choose the customer's RFQ document or part list.</p>
          </div>

          <UploadCloud size={19} />
        </div>

        <form onSubmit={uploadAndExtract}>
          <label className="upload-dropzone">
            <input
              type="file"
              accept=".pdf,.xlsx,.xls,.csv"
              onChange={event => {
                setFile(event.target.files?.[0] || null);
                setError("");
              }}
            />

            <span className="upload-symbol">
              <UploadCloud size={24} />
            </span>

            <strong>
              {file
                ? file.name
                : "Choose a file or drag it here"}
            </strong>

            <small>
              {file
                ? `${(file.size / 1024 / 1024).toFixed(2)} MB · Ready to upload`
                : "Supported: PDF, XLSX, XLS and CSV · Maximum 20 MB"}
            </small>

            <span className="secondary-button upload-browse">
              Browse files
            </span>
          </label>

          <div className="intake-actions">
            <span className="muted-help">
              <Sparkles size={14} />
              AI suggestions must be reviewed before RFQ creation.
            </span>

            <button
              className="primary-button"
              type="submit"
              disabled={!file || busy}
            >
              {busy
                ? "Processing document..."
                : <>Upload & extract <ArrowRight size={15} /></>}
            </button>
          </div>
        </form>
      </section>

      {error && (
        <div className="form-error intake-message">
          <AlertCircle size={15} />
          {error}
        </div>
      )}

      {notice && (
        <div className="intake-success">
          <CheckCircle2 size={16} />
          {notice}
        </div>
      )}

      {extraction && (
        <section className="panel extraction-panel">
          <div className="panel-heading extraction-heading">
            <div>
              <span className="eyebrow">
                <i /> EXTRACTION REVIEW
              </span>

              <h2>AI extracted fields</h2>

              <p>
                Check the source and value. Accept each field
                or edit it before conversion.
              </p>
            </div>

            <button
              className="text-button"
              onClick={() => setExpanded(value => !value)}
            >
              {expanded ? "Collapse" : "Expand"}
              {expanded
                ? <ChevronUp size={15} />
                : <ChevronDown size={15} />}
            </button>
          </div>

          <div className="extraction-meta">
            <span>
              <FileText size={14} />
              {extraction.document_name ||
                extraction.file_name ||
                file?.name ||
                "Uploaded document"}
            </span>

            <span>
              STATUS: {prettyName(extraction.status || "extracted")}
            </span>

            <span>
              {reviewedCount}/{fields.length} fields accepted
            </span>
          </div>

          {expanded && (
            <>
              {!fields.length ? (
                <div className="empty-state">
                  <strong>No extracted fields found.</strong>
                  <span>
                    Check the extraction response or try another document.
                  </span>
                </div>
              ) : (
                <div className="extracted-fields">
                  {fields.map((field, index) => {
                    const name = fieldName(field, index);
                    const original = fieldValue(field);
                    const value = values[name] ?? original;

                    const isAccepted =
                      accepted[name] ?? isFieldAccepted(field);

                    return (
                      <div
                        className="extracted-field"
                        key={`${name}-${index}`}
                      >
                        <div className="extracted-field-top">
                          <label>{prettyName(name)}</label>

                          <span className="field-confidence">
                            {field.confidence != null
                              ? `${Math.round(
                                  Number(field.confidence) *
                                  (Number(field.confidence) <= 1 ? 100 : 1)
                                )}% confidence`
                              : "AI suggestion"}
                          </span>
                        </div>

                        <div className="extracted-field-input">
                          <input
                            value={value}
                            onChange={event =>
                              updateValue(name, event.target.value)
                            }
                          />

                          <button
                            type="button"
                            className={
                              isAccepted
                                ? "accepted-button"
                                : "accept-button"
                            }
                            onClick={() => acceptField(name)}
                            disabled={extractionStatus === "reviewed"}
                          >
                            {isAccepted
                              ? <CheckCircle2 size={14} />
                              : <PencilLine size={14} />}

                            {isAccepted ? "Accepted" : "Accept"}
                          </button>
                        </div>

                        {(field.source_text || field.source_location) && (
                          <small className="field-source">
                            Source: {field.source_text || field.source_location}
                          </small>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}

              <div className="extraction-footer">
                <span>
                  {allAccepted
                    ? "All fields accepted in this review."
                    : "Accept each field after verifying it against the uploaded document."}
                </span>

                <button
                  className="primary-button"
                  onClick={handleSaveReview}
                  disabled={
                    busy ||
                    !allAccepted ||
                    extractionStatus === "reviewed"
                  }
                >
                  {extractionStatus === "reviewed"
                    ? "Review saved"
                    : busy
                      ? "Saving review..."
                      : "Save review"}
                </button>
              </div>
            </>
          )}

          {createdRfq && (
            <div className="created-rfq">
              <CheckCircle2 size={18} />

              <div>
                <strong>{createdRfq.number}</strong>
                <small>
                  Created from this extraction. Refresh the RFQ register to view it.
                </small>
              </div>
            </div>
          )}

          {!createdRfq && (
            <div className="convert-row">
              <div>
                <strong>Ready to create an RFQ?</strong>
                <small>
                  All fields must be accepted. The backend validates
                  the review before conversion.
                </small>
              </div>

              <button
                className="primary-button"
                onClick={convert}
                disabled={
                  busy ||
                  !allAccepted ||
                  extractionStatus !== "reviewed"
                }
              >
                {busy
                  ? "Creating RFQ..."
                  : <>Review & create RFQ <ArrowRight size={15} /></>}
              </button>
            </div>
          )}
        </section>
      )}
    </>
  );
}
