<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { createRun, detectProject, getHealth, listRuns, subscribeToRunEvents } from '$lib/api';
  import type { HealthResponse, ProjectDetection, Run, RunEvent } from '$lib/api';

  let health: HealthResponse | null = null;
  let healthError = '';
  let detectError = '';
  let submitError = '';
  let projectInfo: ProjectDetection | null = null;
  let runs: Run[] = [];
  let loadingRuns = true;
  let detecting = false;
  let creatingRun = false;
  let selectedRunId = '';
  let eventLog: RunEvent[] = [];

  let projectPath = '.';
  let mode = 'auto';
  let provider = 'auto';
  let eventSource: EventSource | null = null;

  const modeOptions = ['auto', 'browser', 'api', 'unit'];
  const providerOptions = ['auto', 'claude', 'claude-cli', 'codex-cli', 'gemini-cli', 'openai', 'openrouter', 'ollama'];

  function closeStream() {
    eventSource?.close();
    eventSource = null;
  }

  function selectRun(run: Run) {
    selectedRunId = run.id;
    eventLog = [];
    closeStream();
    eventSource = subscribeToRunEvents(run.id, {
      onEvent: (event) => {
        eventLog = [...eventLog, event];
        runs = runs.map((candidate) =>
          candidate.id === run.id
            ? {
                ...candidate,
                status: String(event.data?.status ?? candidate.status),
                updated_at: event.time
              }
            : candidate
        );
      },
      onError: () => {
        submitError = 'Lost connection to run event stream';
      }
    });
  }

  async function refreshRuns() {
    loadingRuns = true;
    try {
      runs = await listRuns();
    } catch (err) {
      submitError = err instanceof Error ? err.message : 'Failed to load runs';
    } finally {
      loadingRuns = false;
    }
  }

  async function inspectProject() {
    detectError = '';
    submitError = '';
    detecting = true;

    try {
      projectInfo = await detectProject(projectPath);
      if (mode === 'auto') {
        mode = projectInfo.detected_mode || 'auto';
      }
    } catch (err) {
      projectInfo = null;
      detectError = err instanceof Error ? err.message : 'Failed to inspect project';
    } finally {
      detecting = false;
    }
  }

  async function startRun() {
    submitError = '';
    creatingRun = true;

    try {
      const run = await createRun({
        project_path: projectPath,
        mode,
        provider
      });
      runs = [run, ...runs];
      selectRun(run);
    } catch (err) {
      submitError = err instanceof Error ? err.message : 'Failed to start run';
    } finally {
      creatingRun = false;
    }
  }

  onMount(async () => {
    try {
      health = await getHealth();
    } catch (err) {
      healthError = err instanceof Error ? err.message : 'Unknown error';
    }

    await refreshRuns();
  });

  onDestroy(() => {
    closeStream();
  });
</script>

<svelte:head>
  <title>suitest web</title>
  <meta name="description" content="Local web UI for suitest" />
</svelte:head>

<div class="page">
  <div class="hero">
    <p class="eyebrow">suitest</p>
    <h1>AI-powered testing, CLI or web</h1>
    <p class="lede">
      Local-first web mode for suitest. Run the same Go engine from a browser,
      with project setup, live lifecycle updates, and a better path toward full run control.
    </p>
  </div>

  <section class="grid">
    <div class="card stack">
      <div>
        <h2>Local API</h2>
        {#if health}
          <p><strong>{health.service}</strong> is ready.</p>
        {:else if healthError}
          <p class="error">{healthError}</p>
          <p>Start it with <code>make dev-backend</code> or <code>suitest web</code>.</p>
        {:else}
          <p>Checking local API…</p>
        {/if}
      </div>

      <div>
        <h2>Run setup</h2>
        <label>
          <span>Project path</span>
          <input bind:value={projectPath} placeholder=". or /path/to/project" />
        </label>

        <div class="actions-row">
          <button class="secondary" on:click={inspectProject} disabled={detecting}>
            {#if detecting}Inspecting…{:else}Inspect project{/if}
          </button>
        </div>

        {#if projectInfo}
          <div class="panel success">
            <p><strong>{projectInfo.name}</strong></p>
            <p>{projectInfo.path}</p>
            <p>Detected mode: <code>{projectInfo.detected_mode}</code></p>
            <p>Signals: {projectInfo.detected_signals.join(', ') || 'none'}</p>
          </div>
        {/if}

        {#if detectError}
          <p class="error">{detectError}</p>
        {/if}

        <div class="field-grid">
          <label>
            <span>Mode</span>
            <select bind:value={mode}>
              {#each modeOptions as option}
                <option value={option}>{option}</option>
              {/each}
            </select>
          </label>

          <label>
            <span>Provider</span>
            <select bind:value={provider}>
              {#each providerOptions as option}
                <option value={option}>{option}</option>
              {/each}
            </select>
          </label>
        </div>

        <div class="actions-row">
          <button on:click={startRun} disabled={creatingRun}>
            {#if creatingRun}Starting…{:else}Start run{/if}
          </button>
        </div>

        {#if submitError}
          <p class="error">{submitError}</p>
        {/if}
      </div>
    </div>

    <div class="stack">
      <div class="card">
        <div class="card-header">
          <h2>Recent runs</h2>
          <button class="secondary tiny" on:click={refreshRuns} disabled={loadingRuns}>Refresh</button>
        </div>

        {#if loadingRuns}
          <p>Loading runs…</p>
        {:else if runs.length === 0}
          <p>No runs yet. Start one from the setup form.</p>
        {:else}
          <div class="run-list">
            {#each runs as run}
              <button class:selected={selectedRunId === run.id} class="run-item" on:click={() => selectRun(run)}>
                <div>
                  <strong>{run.status}</strong>
                  <p>{run.project_path}</p>
                </div>
                <div class="run-meta">
                  <span>{run.mode}</span>
                  <span>{run.provider}</span>
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <div class="card">
        <h2>Live events</h2>
        {#if !selectedRunId}
          <p>Select a run to watch lifecycle events.</p>
        {:else if eventLog.length === 0}
          <p>Waiting for events from <code>{selectedRunId}</code>…</p>
        {:else}
          <div class="event-list">
            {#each eventLog as event}
              <div class="event-item">
                <div class="event-topline">
                  <strong>{event.type}</strong>
                  <span>{new Date(event.time).toLocaleTimeString()}</span>
                </div>
                <p>{event.message}</p>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </section>
</div>

<style>
  :global(body) {
    margin: 0;
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    background: #0b1020;
    color: #e8ecf3;
  }

  :global(button),
  :global(input),
  :global(select) {
    font: inherit;
  }

  .page {
    max-width: 1080px;
    margin: 0 auto;
    padding: 48px 20px 80px;
  }

  .hero {
    margin-bottom: 24px;
  }

  .eyebrow {
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: #7dd3fc;
    font-size: 0.8rem;
    margin-bottom: 8px;
  }

  h1 {
    margin: 0 0 12px;
    font-size: clamp(2rem, 4vw, 3.5rem);
    line-height: 1.05;
  }

  .lede {
    margin: 0;
    color: #b7c2d3;
    max-width: 62ch;
    line-height: 1.6;
  }

  .grid {
    display: grid;
    grid-template-columns: 1.1fr 0.9fr;
    gap: 18px;
  }

  .card {
    background: rgba(15, 23, 42, 0.92);
    border: 1px solid rgba(148, 163, 184, 0.18);
    border-radius: 16px;
    padding: 20px;
  }

  .stack {
    display: grid;
    gap: 18px;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }

  h2 {
    margin-top: 0;
    margin-bottom: 14px;
  }

  label {
    display: grid;
    gap: 8px;
    margin-bottom: 14px;
  }

  label span {
    color: #cbd5e1;
    font-size: 0.95rem;
  }

  input,
  select {
    background: rgba(15, 23, 42, 0.6);
    color: #f8fafc;
    border: 1px solid rgba(148, 163, 184, 0.28);
    border-radius: 10px;
    padding: 12px 14px;
  }

  .field-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }

  .actions-row {
    display: flex;
    gap: 12px;
    margin: 10px 0 0;
  }

  button {
    border: 0;
    border-radius: 10px;
    padding: 12px 16px;
    background: linear-gradient(135deg, #38bdf8, #0ea5e9);
    color: #082f49;
    font-weight: 700;
    cursor: pointer;
  }

  button.secondary {
    background: rgba(56, 189, 248, 0.14);
    color: #7dd3fc;
    border: 1px solid rgba(56, 189, 248, 0.3);
  }

  button.tiny {
    padding: 8px 10px;
    font-size: 0.9rem;
  }

  button:disabled {
    opacity: 0.65;
    cursor: wait;
  }

  .panel {
    border-radius: 12px;
    padding: 14px;
    margin-bottom: 14px;
    background: rgba(148, 163, 184, 0.08);
    border: 1px solid rgba(148, 163, 184, 0.16);
  }

  .panel.success {
    background: rgba(34, 197, 94, 0.08);
    border-color: rgba(74, 222, 128, 0.2);
  }

  .panel p {
    margin: 4px 0;
  }

  .error {
    color: #fda4af;
  }

  .run-list,
  .event-list {
    display: grid;
    gap: 12px;
  }

  .run-item,
  .event-item {
    width: 100%;
    text-align: left;
    display: flex;
    justify-content: space-between;
    gap: 14px;
    padding: 14px;
    border-radius: 12px;
    background: rgba(148, 163, 184, 0.08);
    border: 1px solid rgba(148, 163, 184, 0.14);
    color: inherit;
  }

  .run-item.selected {
    border-color: rgba(56, 189, 248, 0.7);
    box-shadow: 0 0 0 1px rgba(56, 189, 248, 0.35) inset;
  }

  .run-item p,
  .event-item p {
    margin: 6px 0 0;
    color: #b7c2d3;
    word-break: break-word;
  }

  .run-meta {
    display: grid;
    gap: 8px;
    justify-items: end;
    color: #7dd3fc;
    font-size: 0.95rem;
    text-transform: lowercase;
  }

  .event-item {
    display: block;
  }

  .event-topline {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    color: #7dd3fc;
    margin-bottom: 6px;
  }

  code {
    background: rgba(148, 163, 184, 0.18);
    border-radius: 6px;
    padding: 2px 6px;
  }

  @media (max-width: 860px) {
    .grid {
      grid-template-columns: 1fr;
    }

    .field-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
