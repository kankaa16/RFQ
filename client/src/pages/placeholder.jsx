import React from "react";
import { ShieldCheck, FileText } from "lucide-react";

export default function Placeholder({ type }) {
  const approval = type === "approvals";
  return <><div className="page-heading"><div><span className="eyebrow"><i /> WORKFLOW</span><h1>{approval ? "Approvals" : "Quotations"}</h1><p>{approval ? "Review and decide on quotations awaiting sales approval." : "View and manage customer quotations."}</p></div></div><div className="panel feature-placeholder"><span className="placeholder-icon">{approval ? <ShieldCheck size={25} /> : <FileText size={25} />}</span><h2>{approval ? "Approval workspace" : "Quotation workspace"}</h2><p>{approval ? "The approval API is available in the backend. This screen needs the approval list endpoint and role-aware actions to be connected." : "Quote creation and PDF download exist in the backend. Multi-part quote line items and the quote listing endpoint need to be completed before this screen can be fully connected."}</p><span className="coming-label">INTEGRATION PENDING</span></div></>;
}
