
const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

export function getToken() {
  return localStorage.getItem("rfq_token") || "";
}

export function setToken(token) {
  localStorage.setItem("rfq_token", token);
}

export function clearToken() {
  localStorage.removeItem("rfq_token");
}

export async function apiRequest(path, options = {}) {
  const token = getToken();
  const headers = { ...(options.headers || {}) };

  if (!(options.body instanceof FormData) && options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers
  });

  const raw = await response.text();
  let payload = {};

  try {
    payload = raw ? JSON.parse(raw) : {};
  } catch {
    payload = { message: raw };
  }

  if (!response.ok || payload.success === false) {
    throw new Error(payload.message || `Request failed (${response.status})`);
  }

  return payload;
}

export const api = {
  login: (email, password) =>
    apiRequest("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password })
    }),

  profile: () => apiRequest("/api/profile"),

  health: () => apiRequest("/api/health"),

  rfqs: () => apiRequest("/api/rfqs"),

  parts: (rfqId) => apiRequest(`/api/rfqs/${rfqId}/parts`),

  rates: () => apiRequest("/api/rates"),

  routing: (partId) => apiRequest(`/api/parts/${partId}/routing`),

  calculateCosting: (data) =>
    apiRequest("/api/costings/calculate", {
      method: "POST",
      body: JSON.stringify(data)
    }),

  createQuote: (data) =>
    apiRequest("/api/quotes", {
      method: "POST",
      body: JSON.stringify(data)
    })
};
