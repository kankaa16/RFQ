# RFQ Costing Workspace - AI Intake

## What changed
- Added an **AI intake** item to the estimator sidebar.
- Added document upload and AI extraction using the existing backend endpoints.
- Added extracted-field review, accept/edit actions, and RFQ conversion.
- Refreshes the RFQ register after successful conversion.
- Hides AI intake from the sales-head sidebar.

## Install / run
From the project root:

```bash
cd client
npm install
```

Create `client/.env`:

```env
VITE_API_URL=http://localhost:8080
```

Run the Go backend in a separate terminal, then:

```bash
npm run dev
```

Open `http://localhost:5173` and sign in with an estimator account. Choose **AI intake** in the left sidebar.

## Intake API calls used
- `POST /api/documents/upload` (multipart form field: `file`)
- `POST /api/ai-extractions/:document_id/extract`
- `GET /api/ai-extractions/:document_id`
- `POST /api/extractions/:extraction_id/review`
- `POST /api/extractions/:extraction_id/convert`

All requests use the existing bearer token from the frontend login.

## Important
The review screen sends a `fields` array containing `field_name`, `reviewed_value`, `is_accepted`, and `review_status`. This must match the review request struct in the Go backend. The backend's extraction/review route is the source of truth; if it expects a different JSON shape, update only the payload in `src/pages/intake.jsx` to match that struct.

AI output is a suggestion. Estimators must verify the extracted information against the customer document before accepting it and creating the RFQ.
