export type HealthResponse = {
  ok: boolean;
  service: string;
};

export type ProjectDetection = {
  path: string;
  name: string;
  detected_mode: string;
  detected_signals: string[];
};

export type Run = {
  id: string;
  project_path: string;
  mode: string;
  provider: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type RunEvent = {
  time: string;
  type: string;
  message: string;
  data?: Record<string, unknown>;
};

const DEFAULT_API_BASE = 'http://localhost:4020';

export async function getHealth(apiBase = DEFAULT_API_BASE): Promise<HealthResponse> {
  const response = await fetch(`${apiBase}/api/health`);
  if (!response.ok) {
    throw new Error(`health request failed: ${response.status}`);
  }
  return response.json();
}

export async function detectProject(path: string, apiBase = DEFAULT_API_BASE): Promise<ProjectDetection> {
  const url = new URL(`${apiBase}/api/projects/detect`);
  url.searchParams.set('path', path);

  const response = await fetch(url);
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.error ?? `project detect failed: ${response.status}`);
  }
  return response.json();
}

export async function createRun(
  input: { project_path: string; mode: string; provider: string },
  apiBase = DEFAULT_API_BASE
): Promise<Run> {
  const response = await fetch(`${apiBase}/api/runs`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(input)
  });

  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.error ?? `run creation failed: ${response.status}`);
  }

  return response.json();
}

export async function listRuns(apiBase = DEFAULT_API_BASE): Promise<Run[]> {
  const response = await fetch(`${apiBase}/api/runs`);
  if (!response.ok) {
    throw new Error(`run list failed: ${response.status}`);
  }
  const payload = await response.json();
  return payload.runs ?? [];
}

export function subscribeToRunEvents(
  runId: string,
  handlers: {
    onEvent: (event: RunEvent) => void;
    onError?: (error: Event) => void;
  },
  apiBase = DEFAULT_API_BASE
): EventSource {
  const source = new EventSource(`${apiBase}/api/runs/${runId}/events`);
  source.addEventListener('run_event', (event) => {
    const payload = JSON.parse((event as MessageEvent).data) as RunEvent;
    handlers.onEvent(payload);
  });
  if (handlers.onError) {
    source.addEventListener('error', handlers.onError);
  }
  return source;
}
