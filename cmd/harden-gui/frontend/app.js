// app.js — frontend de la GUI harden-win11.
// Vanilla JS, pas de framework. Utilise window.go.main.App.* injecté par Wails.

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);

let currentSections = [];
let rulesByID = {};
let engineInfo = null;
let isRunning = false;
let totalRulesInRun = 0;
let processedRules = 0;
const rowsByRuleID = {};

// ─────────────────────────────────────────────────────────────────
// Init
// ─────────────────────────────────────────────────────────────────

window.addEventListener('DOMContentLoaded', async () => {
    await refreshEngineInfo();
    await refreshSections();
    await refreshRuns();
    bindEvents();
    bindWailsEvents();
});

async function refreshEngineInfo() {
    try {
        engineInfo = await window.go.main.App.GetEngineInfo();
    } catch (err) {
        $('#engine-info').innerHTML = `<span style="color:#ff9099">init error: ${err}</span>`;
        return;
    }
    const adminLabel = engineInfo.isAdmin
        ? '<span style="color:#aaffbb">admin ✓</span>'
        : '<span style="color:#ff9099">non-admin (apply désactivé)</span>';
    $('#engine-info').innerHTML =
        `engine ${engineInfo.engineVersion} · manifest ${engineInfo.manifestVersion} · ${adminLabel}`;
    $('#log-path').textContent = engineInfo.logPath ? `log: ${engineInfo.logPath}` : '';
    $('#log-path').title = engineInfo.logPath || '';
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
    list.innerHTML = '<span class="muted small">chargement…</span>';
    try {
        currentSections = await window.go.main.App.GetSections();
    } catch (err) {
        list.innerHTML = `<span style="color:#ff9099">Erreur: ${err}</span>`;
        return;
    }
    rulesByID = {};
    for (const s of currentSections) {
        for (const r of (s.rules || [])) {
            rulesByID[r.id] = r;
        }
    }
    list.innerHTML = currentSections.map(s => `
        <label title="${escapeHtml(s.description || '')}">
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
    $('#btn-undo').addEventListener('click', () => alert("Undo via GUI : à wirer (utilise pour l'instant : harden-engine.exe undo)"));

    $('#modal-cancel').addEventListener('click', closeModal);
    $('#modal-confirm-btn').addEventListener('click', () => {
        closeModal();
        runEngine('apply');
    });
    $('#btn-cancel').addEventListener('click', () => {
        window.go.main.App.CancelRun();
        $('#loader-title').textContent = 'Annulation en cours…';
    });

    document.body.addEventListener('mouseover', onRowHover);
    document.body.addEventListener('mouseout', onRowOut);
    document.body.addEventListener('mousemove', onMouseMove);
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
    prepareTableForRun(sections);
    showLoader(mode);
    disableButtons(true);

    try {
        const summary = mode === 'apply'
            ? await window.go.main.App.Apply(sections)
            : await window.go.main.App.DryRun(sections);
        const cls = summary.cancelled ? 'aborted' : (summary.aborted ? 'aborted' : 'success');
        setStatus(cls, summarizeStatus(summary));
        await refreshRuns();
    } catch (err) {
        setStatus('error', `Erreur: ${err}`);
    } finally {
        isRunning = false;
        hideLoader();
        disableButtons(false);
    }
}

function promptAndApply() {
    const sections = selectedSections();
    if (sections.length === 0) {
        setStatus('error', 'Sélectionne au moins une section.');
        return;
    }
    $('#modal-sections').innerHTML = sections.map(id => `<li>${escapeHtml(id)}</li>`).join('');
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
// Loader
// ─────────────────────────────────────────────────────────────────

function showLoader(mode) {
    $('#loader-title').textContent = mode === 'apply' ? 'Application en cours…' : 'Analyse en cours…';
    $('#loader-progress').textContent = `0 / ${totalRulesInRun} règles`;
    $('#loader-current').textContent = '—';
    $('#loader').classList.remove('hidden');
}

function hideLoader() {
    $('#loader').classList.add('hidden');
}

function updateLoader(currentRule) {
    processedRules++;
    $('#loader-progress').textContent = `${processedRules} / ${totalRulesInRun} règles`;
    const rule = rulesByID[currentRule];
    $('#loader-current').textContent = rule ? rule.title : currentRule;
}

// ─────────────────────────────────────────────────────────────────
// Wails events
// ─────────────────────────────────────────────────────────────────

function bindWailsEvents() {
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
        if (payload && payload.ruleCount) {
            totalRulesInRun = payload.ruleCount;
            $('#loader-progress').textContent = `0 / ${totalRulesInRun} règles`;
        }
        setStatus('running', `Run ${payload.runId} (${payload.mode}, ${payload.sectionCount} section(s), ${payload.ruleCount} règles)`);
    });
    window.runtime.EventsOn('run_end', (summary) => {
        const cls = summary.cancelled || summary.aborted ? 'aborted' : 'success';
        setStatus(cls, summarizeStatus(summary));
    });
}

function handleEngineEvent(ev) {
    if (ev.type === 'action_result') {
        updateRuleRow(ev);
        updateLoader(ev.rule_id);
    }
}

// ─────────────────────────────────────────────────────────────────
// Tableau
// ─────────────────────────────────────────────────────────────────

function prepareTableForRun(sectionIDs) {
    const tbody = $('#results-body');
    tbody.innerHTML = '';
    Object.keys(rowsByRuleID).forEach(k => delete rowsByRuleID[k]);
    processedRules = 0;
    totalRulesInRun = 0;

    for (const s of currentSections) {
        if (!sectionIDs.includes(s.id)) continue;
        for (const r of (s.rules || [])) {
            const tr = renderRuleRow(r, 'pending', '');
            tbody.appendChild(tr);
            rowsByRuleID[r.id] = tr;
            totalRulesInRun++;
        }
    }
    if (totalRulesInRun === 0) {
        tbody.innerHTML = '<tr class="empty"><td colspan="4">Aucune règle dans la sélection.</td></tr>';
    }
}

function renderRuleRow(rule, status, detail) {
    const tr = document.createElement('tr');
    tr.className = 'row';
    tr.dataset.ruleId = rule.id;
    const severity = rule.severity || 'nice-to-have';
    tr.innerHTML = `
        <td><span class="severity ${severity}">${humanSeverity(severity)}</span></td>
        <td class="rule-name">
            ${escapeHtml(rule.title || rule.id)}
            <span class="rule-id-tech">${escapeHtml(rule.id)}</span>
        </td>
        <td><span class="status ${status}">${escapeHtml(humanStatus(status))}</span></td>
        <td class="detail">${detail || ''}</td>
    `;
    return tr;
}

function updateRuleRow(ev) {
    const ruleID = ev.rule_id;
    const rule = rulesByID[ruleID] || { id: ruleID, title: ruleID, severity: 'nice-to-have' };
    let tr = rowsByRuleID[ruleID];
    if (!tr) {
        tr = renderRuleRow(rule, ev.status, formatDetail(ev));
        $('#results-body').appendChild(tr);
        rowsByRuleID[ruleID] = tr;
        return;
    }
    const status = ev.status || 'unknown';
    const statusCell = tr.querySelector('.status');
    statusCell.className = `status ${status}`;
    statusCell.textContent = humanStatus(status);
    tr.querySelector('.detail').innerHTML = formatDetail(ev);
    tr.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

function formatDetail(ev) {
    if (ev.error) return `<span style="color:#ff9099">${escapeHtml(truncate(ev.error, 200))}</span>`;
    if (ev.status === 'rolled_back') return `<span style="color:#ffaa70">Action a planté → rollback exécuté</span>`;
    if (ev.current_state) return formatStateInline(ev.current_state);
    if (ev.before && ev.after) return `${formatStateInline(ev.before)} → ${formatStateInline(ev.after)}`;
    if (ev.before) return formatStateInline(ev.before);
    return '';
}

function formatStateInline(state) {
    if (state === null || state === undefined) return '<em>(absent)</em>';
    if (typeof state !== 'object') return `<code>${escapeHtml(String(state))}</code>`;
    const entries = Object.entries(state).slice(0, 3);
    return entries.map(([k, v]) => {
        const vs = (v === null || v === undefined) ? '∅' : (typeof v === 'object' ? JSON.stringify(v) : String(v));
        return `<code>${escapeHtml(k)}=${escapeHtml(truncate(vs, 40))}</code>`;
    }).join(' ');
}

function humanStatus(status) {
    return {
        'pending':     'En attente',
        'would_skip':  'OK (déjà conforme)',
        'would_apply': 'À appliquer',
        'would_fail':  'Échec test',
        'skipped':     'OK (déjà conforme)',
        'applied':     'Appliquée ✓',
        'failed':      'Échec',
        'rolled_back': 'Rollback exécuté',
    }[status] || status;
}

function humanSeverity(s) {
    return {
        'critical':     'Critique',
        'important':    'Important',
        'nice-to-have': 'Optionnel',
    }[s] || s;
}

function summarizeStatus(s) {
    const parts = [];
    if (s.skipped) parts.push(`${s.skipped} déjà OK`);
    if (s.applied) parts.push(`${s.applied} ${s.mode === 'apply' ? 'appliquées' : 'à appliquer'}`);
    if (s.failed) parts.push(`${s.failed} échec(s)`);
    if (s.rolledBack) parts.push(`${s.rolledBack} rollback`);
    let suffix = '';
    if (s.cancelled) suffix = ' [ANNULÉ]';
    else if (s.aborted) suffix = ' [ARRÊTÉ après rollback]';
    return `Run ${s.runId} (${s.mode}) · ${parts.join(' · ')}${suffix}`;
}

function setStatus(kind, message) {
    const el = $('#status-bar');
    el.className = kind;
    el.textContent = message;
}

// ─────────────────────────────────────────────────────────────────
// Tooltip au survol
// ─────────────────────────────────────────────────────────────────

function onRowHover(e) {
    const tr = e.target.closest('tr.row');
    if (!tr) return hideTooltip();
    const rule = rulesByID[tr.dataset.ruleId];
    if (!rule) return hideTooltip();
    showTooltip(rule);
}

function onRowOut(e) {
    if (!e.relatedTarget || !e.relatedTarget.closest('tr.row')) {
        hideTooltip();
    }
}

function onMouseMove(e) {
    const tt = $('#rule-tooltip');
    if (tt.classList.contains('hidden')) return;
    const margin = 16;
    let x = e.clientX + margin;
    let y = e.clientY + margin;
    const ttW = tt.offsetWidth;
    const ttH = tt.offsetHeight;
    if (x + ttW > window.innerWidth - 10) x = e.clientX - ttW - margin;
    if (y + ttH > window.innerHeight - 10) y = e.clientY - ttH - margin;
    tt.style.left = `${x}px`;
    tt.style.top = `${y}px`;
}

function showTooltip(rule) {
    const tt = $('#rule-tooltip');
    const irreversibleSection = rule.irreversible
        ? `<div class="tt-irreversible">⚠ Irréversible : ${escapeHtml(rule.irreversibleReason || 'Cette règle ne peut pas être annulée par undo.')}</div>`
        : '';
    const explanationSection = rule.explanation
        ? `<div class="tt-section"><div class="tt-label">Pourquoi</div>${escapeHtml(rule.explanation).replace(/\n/g, '<br>')}</div>`
        : '';
    const rebootSection = rule.requiresReboot
        ? '<div class="tt-section tt-label" style="color:#ffd770">Nécessite un redémarrage après application</div>'
        : '';
    tt.innerHTML = `
        <h4>${escapeHtml(rule.title)} <span class="severity ${rule.severity}" style="margin-left:6px;font-size:9px;vertical-align:middle">${escapeHtml(humanSeverity(rule.severity))}</span></h4>
        <div class="tt-desc">${escapeHtml(rule.description)}</div>
        ${explanationSection}
        <div class="tt-section">
            <div class="tt-label">Impact concret</div>
            <span class="tt-impact">${escapeHtml(rule.impact || '—')}</span>
        </div>
        ${rebootSection}
        ${irreversibleSection}
    `;
    tt.classList.remove('hidden');
}

function hideTooltip() {
    $('#rule-tooltip').classList.add('hidden');
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
