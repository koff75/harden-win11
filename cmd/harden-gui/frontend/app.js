// app.js — frontend de la GUI harden-win11.
// Vanilla JS, pas de framework. Utilise le runtime Wails injecté
// (window.go.main.App.*) pour appeler le backend Go.

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);

let currentSections = [];   // SectionInfo[]
let engineInfo = null;
let isRunning = false;

// ─────────────────────────────────────────────────────────────────
// Init au chargement
// ─────────────────────────────────────────────────────────────────

window.addEventListener('DOMContentLoaded', async () => {
    await refreshEngineInfo();
    await refreshSections();
    await refreshRuns();
    bindEvents();
    bindWailsEvents();
});

// ─────────────────────────────────────────────────────────────────
// API → backend Go
// ─────────────────────────────────────────────────────────────────

async function refreshEngineInfo() {
    engineInfo = await window.go.main.App.GetEngineInfo();
    const adminLabel = engineInfo.isAdmin
        ? '<span style="color:#aaffbb">admin ✓</span>'
        : '<span style="color:#ff9099">non-admin (apply désactivé)</span>';
    $('#engine-info').innerHTML =
        `engine ${engineInfo.engineVersion} · manifest ${engineInfo.manifestVersion} · ${adminLabel}`;
    $('#modal-journal-dir').textContent = engineInfo.journalDir;

    if (!engineInfo.isAdmin) {
        $('#btn-apply').disabled = true;
        $('#btn-apply').title = 'Lance la GUI en admin pour activer';
        $('#btn-undo').disabled = true;
        $('#btn-undo').title = 'Lance la GUI en admin pour activer';
    }
}

async function refreshSections() {
    const list = $('#sections-list');
    try {
        currentSections = await window.go.main.App.GetSections();
    } catch (err) {
        list.innerHTML = `<span style="color:#ff9099">Erreur: ${err}</span>`;
        return;
    }
    list.innerHTML = currentSections.map(s => `
        <label>
            <input type="checkbox" name="section" value="${s.id}" checked>
            <span>${escapeHtml(s.title)}</span>
            <span class="rule-count">${s.ruleCount}</span>
        </label>
    `).join('');
}

async function refreshRuns() {
    const list = $('#runs-list');
    try {
        const runs = await window.go.main.App.ListRuns();
        if (runs.length === 0) {
            list.innerHTML = '<em>Aucun run dans le journal.</em>';
            return;
        }
        list.innerHTML = runs.slice(0, 8).map(r =>
            `<div class="run-item" title="${r}">${escapeHtml(r)}</div>`
        ).join('');
    } catch (err) {
        list.innerHTML = `<span style="color:#ff9099">${err}</span>`;
    }
}

// ─────────────────────────────────────────────────────────────────
// UI events
// ─────────────────────────────────────────────────────────────────

function bindEvents() {
    $('#btn-dryrun').addEventListener('click', () => runEngine('dryrun'));
    $('#btn-apply').addEventListener('click', () => promptAndApply());
    $('#btn-undo').addEventListener('click', () => runUndo());
    $('#modal-cancel').addEventListener('click', closeModal);
    $('#modal-confirm').addEventListener('click', () => {
        closeModal();
        runEngine('apply');
    });
}

function selectedSections() {
    return Array.from($$('input[name="section"]:checked')).map(cb => cb.value);
}

async function runEngine(mode) {
    if (isRunning) return;
    const sections = selectedSections();
    if (sections.length === 0) {
        setStatus('error', 'Sélectionne au moins une section.');
        return;
    }

    isRunning = true;
    clearResults();
    setStatus('running', `${mode}… ${sections.length} section(s)`);
    disableButtons(true);

    try {
        const summary = mode === 'apply'
            ? await window.go.main.App.Apply(sections)
            : await window.go.main.App.DryRun(sections);
        setStatus('success', summarizeStatus(summary));
        await refreshRuns();
    } catch (err) {
        setStatus('error', `Erreur: ${err}`);
    } finally {
        isRunning = false;
        disableButtons(false);
    }
}

async function runUndo() {
    if (isRunning) return;
    const runs = await window.go.main.App.ListRuns();
    if (runs.length === 0) {
        setStatus('error', 'Aucun run à undo.');
        return;
    }
    if (!confirm(`Undo le dernier run (${runs[0]}) ?\n\nLes .undo.ps1 seront rejoués dans l'ordre inverse.`)) {
        return;
    }
    setStatus('running', `undo en cours…`);
    // Note : pour l'instant on ne wire pas Undo() côté frontend complet (la
    // commande CLI le fait déjà). Future feature.
    setStatus('error', 'Undo via GUI : à wirer (pour l\'instant utilise harden-engine.exe undo)');
}

function promptAndApply() {
    const sections = selectedSections();
    if (sections.length === 0) {
        setStatus('error', 'Sélectionne au moins une section.');
        return;
    }
    const ul = $('#modal-sections');
    ul.innerHTML = sections.map(id => `<li>${escapeHtml(id)}</li>`).join('');
    $('#modal-confirm').classList.remove('hidden');
}

function closeModal() {
    $('#modal-confirm').classList.add('hidden');
}

function disableButtons(disabled) {
    $('#btn-dryrun').disabled = disabled;
    $('#btn-apply').disabled = disabled || (engineInfo && !engineInfo.isAdmin);
    $('#btn-undo').disabled = disabled || (engineInfo && !engineInfo.isAdmin);
}

// ─────────────────────────────────────────────────────────────────
// Wails events (live progress)
// ─────────────────────────────────────────────────────────────────

function bindWailsEvents() {
    // Le runtime Wails injecte window.runtime.EventsOn.
    if (!window.runtime || !window.runtime.EventsOn) {
        console.warn('Wails runtime not available');
        return;
    }
    window.runtime.EventsOn('engine_event', (line) => {
        try {
            const ev = JSON.parse(line);
            handleEngineEvent(ev);
        } catch (err) {
            console.warn('bad engine_event payload', line, err);
        }
    });
    window.runtime.EventsOn('run_start', (payload) => {
        clearResults();
        setStatus('running', `Run ${payload.runId} (${payload.mode}, ${payload.sectionCount} sections)`);
    });
    window.runtime.EventsOn('run_end', (summary) => {
        setStatus(summary.aborted ? 'aborted' : 'success', summarizeStatus(summary));
    });
}

function handleEngineEvent(ev) {
    if (ev.type === 'action_result') {
        appendResult(ev);
    }
    // section_start, section_end, rollback_result, etc. → on pourrait les
    // afficher mais on garde le tableau focus sur les action_result pour MVP.
}

// ─────────────────────────────────────────────────────────────────
// Rendu du tableau de résultats
// ─────────────────────────────────────────────────────────────────

function clearResults() {
    $('#results-body').innerHTML = '';
}

function appendResult(ev) {
    const tr = document.createElement('tr');
    const status = ev.status || 'unknown';
    const detail = formatDetail(ev);
    tr.innerHTML = `
        <td class="rule-id">${escapeHtml(ev.rule_id)}</td>
        <td><span class="status ${status}">${escapeHtml(status)}</span></td>
        <td class="detail">${detail}</td>
        <td class="duration">${ev.duration_ms ?? ''}</td>
    `;
    $('#results-body').appendChild(tr);
    tr.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

function formatDetail(ev) {
    if (ev.error) return `<span style="color:#ff9099">${escapeHtml(truncate(ev.error, 120))}</span>`;
    if (ev.current_state) return escapeHtml(JSON.stringify(ev.current_state));
    if (ev.before) return `before: ${escapeHtml(JSON.stringify(ev.before))}`;
    return '';
}

function summarizeStatus(s) {
    return `Run ${s.runId} (${s.mode}) — skipped:${s.skipped} applied:${s.applied} failed:${s.failed} rolled_back:${s.rolledBack}${s.aborted ? ' [ABORTED]' : ''}`;
}

function setStatus(kind, message) {
    const el = $('#status-bar');
    el.className = kind;
    el.textContent = message;
}

// ─────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────

function escapeHtml(str) {
    return String(str ?? '').replace(/[&<>"']/g, (c) => (
        { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]
    ));
}

function truncate(s, max) {
    s = String(s ?? '');
    return s.length > max ? s.slice(0, max) + '…' : s;
}
