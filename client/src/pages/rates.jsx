import React from "react";
import { RefreshCw, Search } from "lucide-react";

export default function Rates({ rates, loading, onRefresh }) {
  const [query, setQuery] = React.useState("");
  const filtered = rates.filter(r => `${r.rate_code} ${r.rate_type} ${r.description}`.toLowerCase().includes(query.toLowerCase()));
  return <>
    <div className="page-heading"><div><span className="eyebrow"><i /> CONFIGURATION</span><h1>Rate masters</h1><p>Centralised rates used by the costing engine.</p></div><button className="secondary-button" onClick={onRefresh}><RefreshCw size={15} /> Refresh</button></div>
    <div className="panel"><div className="toolbar"><div className="search-box"><Search size={16} /><input value={query} onChange={e => setQuery(e.target.value)} placeholder="Search rate code or type..." /></div><span className="result-count">{filtered.length} rates</span></div>
      {loading ? <div className="empty-state">Loading rates...</div> : <div className="table-wrap"><table><thead><tr><th>RATE CODE</th><th>TYPE</th><th>VALUE</th><th>UNIT</th><th>VERSION</th><th>EFFECTIVE FROM</th><th>STATUS</th></tr></thead><tbody>{filtered.map(r => <tr key={r.id}><td><strong>{r.rate_code}</strong></td><td>{r.rate_type || "—"}</td><td>₹{Number(r.rate_value ?? r.rate ?? 0).toLocaleString("en-IN")}</td><td>{r.unit || "—"}</td><td>v{r.version || 1}</td><td>{r.effective_from ? new Date(r.effective_from).toLocaleDateString("en-IN") : "—"}</td><td><span className={`status-badge ${r.is_active ? "green" : "neutral"}`}><span />{r.is_active ? "active" : "inactive"}</span></td></tr>)}</tbody></table>{!filtered.length && <div className="empty-state">No rate masters found.</div>}</div>}
      <div className="table-footnote">Rate creation and editing are not enabled in this screen yet. Existing rate records are read-only.</div>
    </div>
  </>;
}
