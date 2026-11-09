# RFQ-to-Quote Costing Web Application

**Technical Design Document (TDD)**  
**Project:** Auto-component manufacturer's RFQ-to-Quote Costing Web App  
**Document status:** Development snapshot  
**Last updated:** 8 November 2026

> **GitHub repository:** https://github.com/kankaa16/RFQ
> **Live application:** TODO 

---

## 1. Project overview

The application is designed for an auto-component manufacturer that receives requests for quotation (RFQs) through email, usually with drawing PDFs and Excel part lists.

The current manual process requires estimators to enter part information, copy rates from older systems, calculate material and machining costs, and prepare quotations in spreadsheets. This leads to long response times and makes it difficult to trace which rates were used for a quotation.

The application brings the main RFQ-to-quote workflow into one system:

- Register RFQs and track due dates, owners and status.
- Store part details and manufacturing routings.
- Maintain rate masters with effective dates and version history.
- Calculate costs at multiple volume scenarios using fixed formulas.
- Use AI to extract candidate information from RFQ documents.
- Require estimator review before extracted values are accepted.
- Create quotations and route high-value quotations for sales-head approval.
- Generate quote PDFs and search historical quote records.

The implementation focuses on the core build described in the problem statement. It is not intended to replace the customer's ERP.

## 2. Problem context and objectives

The brief describes the following operating assumptions:

| Item | Value |
|---|---:|
| RFQs received per month | 60 |
| Average parts per RFQ | 5 |
| Estimator time per part | 2.5 hours |
| Estimator hourly cost | ₹450 |
| Estimated annual manual effort | 9,000 hours |
| Estimated annual effort cost | ₹40.5 lakh |
| Current RFQ turnaround | 9 working days |
| Expected RFQ turnaround | 4 working days |
| Estimated annual contribution lost due to late replies | ₹86.4 lakh |
| Total estimated annual impact | ₹1.27 crore |

The intended product outcomes are:

1. Reduce estimator effort from 2.5 hours to approximately 1 hour per part.
2. Reduce RFQ turnaround from 9 working days to 4 working days.
3. Ensure every quote can be traced to the rate versions used during costing.
4. Reduce repeated manual entry by extracting RFQ and drawing information.
5. Make past quote information easier to find and reuse.

## 3. Users and roles

| Role | Main responsibilities |
|---|---|
| Estimator | Review RFQs, confirm extracted information, maintain routing details, calculate costs and prepare quotes. |
| Sales head | Review quotations requiring approval and update RFQ outcomes. |


## 4. Technology stack

| Layer | Technology | Purpose |
|---|---|---|
| Frontend | React, JavaScript, Vite, CSS | User interface and client-side navigation. |
| Backend | Go, Gin | REST API, authentication, business rules and services. |
| Database | MongoDB, MongoDB Go Driver v2 | Persistent application data. |
| Authentication | JWT | Authenticated API access and role-aware operations. |
| AI extraction | Google Gemini API | Extract candidate RFQ and drawing fields from uploaded documents. |
| Document handling | Go upload service and server-side file storage | Upload and store RFQ documents and their metadata. |
| PDF generation | Go PDF generation service | Produce quotation PDFs. |

### Why this architecture

- React and Vite provide a responsive single-page interface with a straightforward development workflow.
- Go and Gin provide a compact backend with explicit service and model boundaries.
- MongoDB supports the document-oriented RFQ, part, routing, extraction and quote records.
- Costing calculations remain in the backend so that the same validated formulas are used regardless of the frontend.
- AI is limited to extracting and suggesting information; it does not determine prices or perform financial calculations.

## 5. Repository structure

```text
rfq_costing/
├── client/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── layouts/
│   │   ├── services/
│   │   ├── store/
│   │   ├── utils/
│   │   ├── app.jsx
│   │   └── main.jsx
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── server/
│   ├── main.go
│   ├── config/
│   ├── database/
│   ├── middleware/
│   ├── models/
│   ├── routes/
│   ├── services/
│   ├── scripts/
│   ├── uploads/
│   ├── go.mod
│   └── .env
├── docs/
├── docker-compose.yml
├── .gitignore
└── README.md
```

The `.env` file is local configuration and must not be committed.

## 6. Architecture and request flow

```text
User
  |
  v
React + Vite frontend
  |
  | HTTP requests with JWT Bearer token
  v
Go + Gin API
  |
  +--> Authentication and role middleware
  |
  +--> RFQ, part and routing services
  |
  +--> Rate master and costing services
  |
  +--> Document and AI extraction services
  |
  +--> Quote, approval and PDF services
  |
  v
MongoDB

Document extraction service
  |
  +--> Uploaded RFQ PDF/document
  |
  +--> Gemini API
  |
  v
Candidate extracted fields
  |
  v
Estimator review and acceptance
```

The backend validates requests and applies business rules. The frontend does not independently calculate or decide final quote prices.

## 7. Data model

The main MongoDB collections and their responsibilities are summarized below. Exact field definitions are maintained in the Go model files.

| Collection / entity | Purpose |
|---|---|
| `users` | User identity, authentication data and role. |
| `customers` | Customer company name/code, contact information, terms, currency and active status. |
| `rfqs` | RFQ number, customer reference, due date, owner and workflow status. |
| `parts` | Part number/name, RFQ reference, drawing number/revision, material, weights, volumes and verification flags. |
| `routings` | Part-specific ordered operations, machine type, cycle/setup time, batch size and rate-master reference. |
| `rate_masters` | Rate code, name, type, category, value, unit, currency, version, effective dates and active status. |
| `costings` | Calculated costing scenarios, formula version and rate snapshots. |
| `quotes` | Quote version, RFQ/customer references, values, GST, annual value and status. |
| `approvals` | Quote approval request, reviewer, decision and review metadata. |
| `documents` | Uploaded document metadata and storage reference. |
| `ai_extractions` | Extracted values, source text/location, confidence and review/acceptance state. |
| `historical_quotes` | Imported and searchable historical quotation records. |

### Important relationships

- One RFQ can contain multiple parts.
- A part can have a routing containing multiple operations.
- Each routing operation can reference a machine rate master.
- A rate code can have multiple versions.
- A costing stores the selected rate snapshots so later rate changes do not silently rewrite the costing history.
- Quotes are associated with RFQ/customer costing data and may have approval records.

## 8. Main application workflows

### 8.1 Authentication

1. User submits login credentials.
2. Backend validates the credentials.
3. Backend returns a JWT and user profile.
4. Frontend sends the token in the `Authorization: Bearer <token>` header for protected API requests.
5. Backend middleware validates the token and applies role checks to protected operations.

### 8.2 RFQ register and status management

1. Estimator opens the RFQ register.
2. RFQ records show customer, due date, owner and status.
3. Estimator works on the RFQ and can mark it as sent.
4. Sales head can record the RFQ outcome as won or lost.
5. A lost RFQ requires a reason.

The implemented workflow uses estimating/sent/won/lost states. A complete plant-head review and MD deal-signing workflow is not included in the current core slice.

### 8.3 AI document intake

1. Estimator uploads an RFQ document.
2. Backend validates the file and stores document metadata.
3. The extraction service sends supported document content to Gemini.
4. Gemini returns structured candidate fields for the RFQ and its parts.
5. Extracted fields retain source information and review metadata, including confidence where available.
6. Estimator reviews, edits and accepts the extracted values.
7. The reviewed extraction can be converted into an RFQ and associated parts.

**Human review is mandatory.** AI output is a suggestion, not an authoritative business record. The application must not silently accept uncertain values.

AI is not used for costing arithmetic, margin calculations, quote approval or final price decisions.

### 8.4 Rate master management and versioning

1. An authorized user creates a rate with a rate code, type, category, value, unit and effective-from date.
2. A rate change creates a new version under the same rate code.
3. The previous version remains available in history and is closed/inactivated according to the service rules.
4. The new version becomes the current version according to its effective period.
5. Costing selects a rate that is valid for the costing date and stores a snapshot of the rate used.

The Rate Masters frontend supports adding rates, creating a new version and viewing version history. These UI actions were manually tested during development.

### 8.5 Part costing

1. Estimator selects a part.
2. Estimator enters or confirms material and costing assumptions.
3. Estimator adds or reviews routing operations.
4. Each routing operation must reference a valid machine rate master.
5. Backend checks the rate type, positive rate value and effective date.
6. Backend calculates the costing for the requested volume scenarios.
7. Costing output stores the formula version and rate snapshots.

The current costing service was updated to use the linked rate-master value rather than relying on a manually typed hour rate. The routing UI and saved routing data still require a complete end-to-end check to confirm every operation sends a valid `rate_master_id`.

### 8.6 Quote and approval

1. A quote is created from RFQ/part costing data.
2. The quote's annual value is checked.
3. Quotes above **₹25,00,000 annual value** require sales-head approval.
4. Sales head reviews the pending approval.
5. Quote PDF generation is available through a protected endpoint.

Quantity-extended totals, GST treatment and PDF totals require final validation before production use. PDF access must remain restricted until the quote reaches an allowed status.

## 9. Costing logic and rate traceability

Cost calculations are fixed backend formulas. Gemini is not involved in arithmetic.

The current costing service:

- Requires a `RateMasterID` for each routing operation.
- Fetches the corresponding rate master.
- Validates that the rate is a machine rate and is greater than zero.
- Checks that the costing date falls within the rate's effective period.
- Uses the master rate for machining and setup calculations.
- Stores rate snapshots for the costing.
- Stores a formula version for traceability.

The overall costing inputs include:

- Material rate and material weight.
- Scrap weight/rate or scrap assumptions.
- Routing cycle time and setup time.
- Machine-hour rate.
- Overhead.
- Packing.
- Freight.
- Margin.
- Requested volume scenarios.

**Implementation note:** The exact treatment of setup amortization, scrap recovery, overhead base, margin convention, quantity extensions and GST must be checked against the final tested code. The current quote service's unit-price subtotal and quantity/GST behavior were identified as needing validation.

## 10. API endpoints

The following endpoints were implemented or registered during development. Confirm the current route file before treating this list as a final API contract.

| Endpoint | Purpose |
|---|---|
| `POST /api/auth/login` | Authenticate user. |
| `GET /api/profile` | Get authenticated user profile. |
| `GET /api/rfqs` | List RFQs. |
| `GET /api/rfqs/:rfq_id/parts` | List parts for an RFQ. |
| `PATCH /api/rfqs/:rfq_id/status` | Update RFQ status. |
| `GET /api/rates` | List rate masters. |
| `POST /api/rates` | Create a rate or new version. |
| `GET /api/rates/:rate_code/history` | Retrieve rate version history. |
| `GET /api/parts/:part_id/routing` | Retrieve part routing. |
| `POST /api/parts/:part_id/routing` | Save part routing. |
| `POST /api/costings/calculate` | Calculate part costing. |
| `POST /api/documents/upload` | Upload RFQ document. |
| `POST /api/ai-extractions/:document_id/extract` | Run AI extraction. |
| `GET /api/ai-extractions/:document_id` | Retrieve extraction for review. |
| `POST /api/extractions/:extraction_id/review` | Review extracted fields. |
| `POST /api/extractions/:extraction_id/convert` | Convert reviewed extraction to RFQ/parts. |
| `POST /api/quotes` | Create quote. |
| `GET /api/quotes` | List quotes. |
| `POST /api/quotes/:id/review` | Review quote approval. |
| `GET /api/quotes/:id/pdf` | Generate/download quote PDF. |
| `GET /api/approvals` | List approvals for sales head. |
| `GET /api/historical-quotes` | Search historical quote records. |

## 11. User interface and design decisions

The client brief asks for a minimal interface with clear layouts and no unnecessary decoration.

The frontend uses:

- A restrained light theme and neutral surfaces.
- Clear typography and spacing.
- An RFQ register focused on status, due date, customer and ownership.
- A costing layout intended for dense but readable numeric information.
- A Rate Masters screen with add, version and history actions.
- An AI review flow where extracted values can be inspected and accepted.
- Role-aware dashboard behavior for estimator and sales-head users.

The target usage is desktop-first because the brief says estimators work primarily on desktop. Responsive styling exists, but phone and tablet breakpoints still require a complete QA pass.

## 12. Sample data

Seed scripts were created for fictional demonstration records, including customers, RFQs, parts, machine rates and routings.

The three sample RFQs from the brief are represented:

| RFQ | Customer | Part | Material | Gross weight | Annual volume | Cycle time | Hour rate |
|---|---|---|---|---:|---:|---:|---:|
| RFQ-2611-014 | Company B | Rear wheel spindle | EN8 | 0.61 kg | 180,000 | 3.2 min | ₹850 |
| RFQ-2611-019 | Company C | Hydraulic lift arm | 40Cr4 forging | 4.60 kg | 24,000 | 11.5 min | ₹1,450 |
| RFQ-2611-023 | Company D | Gear blank | 20MnCr5 | 1.48 kg | 60,000 | 4.8 min | ₹950 |

Additional fictional single-part and multi-part PDF samples were used to exercise document extraction, review and conversion.

## 13. Testing and validation performed

Manual checks during development included:

- Backend health endpoint.
- Login and authenticated profile.
- RFQ and part listing.
- Rate listing, rate creation, version creation and rate history.
- Routing retrieval.
- Costing calculation.
- Quote creation and approval review.
- Quote PDF endpoint.
- Document upload.
- AI extraction, field review and conversion.
- Approval listing.

These were manual integration checks. They must not be described as a complete automated test suite.

### Tests still required

- Unit tests for material, machining/setup, overhead, margin and multi-volume costing.
- Effective-date boundary tests for rates.
- Regression test proving old quote/costing snapshots do not change after a new rate version is created.
- Role tests for estimator and sales-head permissions.
- AI tests for missing fields, low confidence, malformed responses and multi-part documents.
- Duplicate customer/RFQ conversion tests.
- Quote threshold, quantity extension, GST and PDF total tests.
- Upload tests for unsupported file types, oversized files and unauthorized access.

## 14. Security and data handling

- Protected API routes use JWT authentication.
- Role checks are required for sales-head approval and RFQ outcome actions.
- Uploaded files are validated by the document service for supported formats and size.
- MongoDB URI, JWT secret and Gemini API key must be provided through environment configuration.
- `.env` files and credentials must never be committed.
- Customer drawings are NDA-sensitive. Production must use private storage, named-user access, HTTPS, least privilege, audit logging, backups and retention controls.
- Confirm the AI provider's data handling terms and account configuration satisfy the client's requirement that submitted data is not used for model training before processing confidential drawings.
- The current local upload directory must not be exposed as a public static directory in production.

## 15. Assumptions and limitations

The following assumptions and limitations are explicit:

1. The first release is a standalone application; ERP integration is not included.
2. Email-server integration, supplier portal, CAD/3D drawing analysis, messaging and payments are out of scope.
3. Sample customer and part records are fictional demonstration data.
4. Rates are maintained in the application until a separately scoped ERP integration exists.
5. The core implementation targets estimator and sales-head roles.
6. Plant-head routing review and MD deal sign-off are not claimed as complete workflows.
7. Historical quote search and backend import work exist, but the full Excel import UX, row-level validation and duplicate reporting need verification.
8. Customer auto-creation during AI conversion requires additional duplicate and transaction-safety testing.
9. Quote quantity extension, GST totals and PDF totals need final validation.
10. A live deployment and automated test suite have not been confirmed in this development snapshot.
---