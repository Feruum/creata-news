const AGENT_URL = import.meta.env.VITE_AGENT_URL || "http://localhost:8000";

async function agentFetch(
  path: string,
  options?: RequestInit,
): Promise<Response> {
  const res = await fetch(`${AGENT_URL}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(
      body.error || body.detail || `Request failed: ${res.status}`,
    );
  }
  return res;
}

export interface DocumentSource {
  source: string;
  count: number;
}

export const agentApi = {
  async listDocuments(): Promise<DocumentSource[]> {
    const res = await agentFetch("/admin/documents");
    const data = await res.json();
    return data.sources || data || [];
  },

  async uploadDocument(
    file: File,
  ): Promise<{ status: string; source: string; chunks: number }> {
    const form = new FormData();
    form.append("file", file);
    const res = await agentFetch("/admin/ingest", {
      method: "POST",
      body: form,
    });
    return res.json();
  },

  async deleteDocument(source: string): Promise<void> {
    await agentFetch(`/admin/documents?source=${encodeURIComponent(source)}`, {
      method: "DELETE",
    });
  },
};
