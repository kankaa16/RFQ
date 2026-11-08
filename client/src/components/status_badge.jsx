import React from "react";

export default function StatusBadge({ status = "estimating" }) {
  const normalized = String(status).toLowerCase().replaceAll("_", " ");
  const kind = normalized === "won" || normalized === "approved" ? "green"
    : normalized === "lost" || normalized === "rejected" ? "red"
    : normalized === "sent" || normalized === "pending approval" ? "blue" : "neutral";
  return <span className={`status-badge ${kind}`}><span />{normalized}</span>;
}
