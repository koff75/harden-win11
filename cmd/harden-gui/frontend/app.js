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
let currentProfile = 'personal';   // profil par défaut, override par l'utilisateur
let availableProfiles = [];
const rowsByRuleID = {};
const eventByRuleID = {};   // dernier event reçu par règle (pour le tooltip)
const excludedRules = new Set();   // rule_ids exclus du prochain run

// ─────────────────────────────────────────────────────────────────
// Init
// ─────────────────────────────────────────────────────────────────

window.addEventListener('DOMContentLoaded', async () => {
    await refreshEngineInfo();
    await refreshProfiles();
    await refreshSections();
    await refreshRuns();
    await refreshCoverage();
    bindEvents();
    bindWailsEvents();
});

let coverageReport = null;

async function refreshCoverage() {
    try {
        coverageReport = await window.go.main.App.GetCoverage();
    } catch (err) {
        return;
    }
    if (!coverageReport) return;
    const bar = $('#coverage-bar');
    if (!bar) return;
    const fmt = (st) => `${st.mapped}/${coverageReport.total_rules} (${Math.round(st.mapped/coverageReport.total_rules*100)}%)`;
    const cis = coverageReport.frameworks.cis;
    const anssi = coverageReport.frameworks.anssi;
    const ms = coverageReport.frameworks.ms_baseline;
    bar.querySelector('[data-fw="cis"]').textContent = `CIS ${fmt(cis)}`;
    bar.querySelector('[data-fw="anssi"]').textContent = `ANSSI ${fmt(anssi)}`;
    bar.querySelector('[data-fw="ms_baseline"]').textContent = `MS ${fmt(ms)}`;
    bar.classList.remove('hidden');
    bar.addEventListener('click', showCoverageModal);
}

function showCoverageModal() {
    if (!coverageReport) return;
    const fw = coverageReport.frameworks;
    const sources = coverageReport.sources || {};
    const fwRow = (label, st, src) => `
        <tr>
            <td><strong>${label}</strong><br><span class="muted small">${escapeHtml(src || '')}</span></td>
            <td>${st.mapped}/${coverageReport.total_rules}</td>
            <td>${st.unique_controls}</td>
            <td>${st.unmapped_rules.length}</td>
            <td><span class="muted small">${st.sample_controls.slice(0,3).map(escapeHtml).join('<br>')}</span></td>
        </tr>`;
    const html = `
        <div class="cov-modal" id="cov-modal-overlay">
            <div class="cov-modal-content">
                <span class="cov-close" id="cov-close">✕</span>
                <h3>Couverture vs référentiels publics</h3>
                <p class="muted small">Total règles harden-win11 : <strong>${coverageReport.total_rules}</strong> · Avec ≥1 mapping : <strong>${coverageReport.mapped_rules}</strong></p>
                <table>
                    <thead>
                        <tr><th>Référentiel</th><th>Couvertes</th><th>Contrôles uniques</th><th>Sans mapping</th><th>Exemples</th></tr>
                    </thead>
                    <tbody>
                        ${fwRow('CIS', fw.cis, sources.cis)}
                        ${fwRow('ANSSI', fw.anssi, sources.anssi)}
                        ${fwRow('MS Baseline', fw.ms_baseline, sources.ms_baseline)}
                    </tbody>
                </table>
                <p class="muted small" style="margin-top:16px"><em>${escapeHtml(coverageReport.disclaimer || '')}</em></p>
            </div>
        </div>`;
    document.body.insertAdjacentHTML('beforeend', html);
    const overlay = $('#cov-modal-overlay');
    const close = () => overlay.remove();
    $('#cov-close').addEventListener('click', close);
    overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
}

async function refreshProfiles() {
    try {
        availableProfiles = await window.go.main.App.GetProfiles();
    } catch (err) {
        $('#profile-list').innerHTML = `<span style="color:#ff9099">Erreur: ${err}</span>`;
        return;
    }

    // Détection auto du contexte → suggestion de profil par défaut.
    let suggestion = null;
    try {
        suggestion = await window.go.main.App.DetectContext();
        if (suggestion && suggestion.suggestedProfile) {
            currentProfile = suggestion.suggestedProfile;
        }
    } catch (err) {
        // pas grave : on garde le défaut 'personal'.
    }

    const suggestionBanner = suggestion
        ? `<div class="profile-suggestion" title="${escapeHtml(suggestion.reason)}">
              💡 Suggéré : <strong>${escapeHtml(profileTitleFromID(suggestion.suggestedProfile))}</strong>
              <div class="profile-suggestion-reason">${escapeHtml(suggestion.reason)}</div>
           </div>`
        : '';

    $('#profile-list').innerHTML = suggestionBanner + availableProfiles.map(p => `
        <label class="profile-item" title="${escapeHtml(p.description)}">
            <input type="radio" name="profile" value="${p.id}" ${p.id === currentProfile ? 'checked' : ''}>
            <span class="profile-title">${escapeHtml(p.title)}</span>
            <span class="profile-desc">${escapeHtml(p.description)}</span>
        </label>
    `).join('');
    $$('input[name="profile"]').forEach(rb => {
        rb.addEventListener('change', async () => {
            currentProfile = rb.value;
            await refreshSections();
            $('#results-body').innerHTML = '<tr class="empty"><td colspan="5">Profil changé. Lance un dry-run pour voir l\'état actuel.</td></tr>';
            $('#dashboard').classList.add('hidden');
            Object.keys(rowsByRuleID).forEach(k => delete rowsByRuleID[k]);
            Object.keys(eventByRuleID).forEach(k => delete eventByRuleID[k]);
        });
    });
}

function profileTitleFromID(id) {
    const found = availableProfiles.find(p => p.id === id);
    return found ? found.title : id;
}

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
        $('#admin-banner').classList.remove('hidden');
        $('#btn-relaunch-admin').addEventListener('click', async () => {
            const btn = $('#btn-relaunch-admin');
            btn.disabled = true;
            btn.textContent = 'Relancement…';
            try {
                await window.go.main.App.RelaunchAsAdmin();
            } catch (err) {
                btn.disabled = false;
                btn.textContent = 'Relancer en admin';
                alert('Échec du relancement : ' + err);
            }
        });
    }
}

async function refreshSections() {
    const list = $('#sections-list');
    list.innerHTML = '<span class="muted small">chargement…</span>';
    try {
        currentSections = await window.go.main.App.GetSections(currentProfile);
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
            `<div class="run-item" data-run-id="${escapeHtml(r)}" title="Cliquer pour charger ce run">${escapeHtml(r)}</div>`
        ).join('');
        list.querySelectorAll('.run-item').forEach(el => {
            el.addEventListener('click', () => loadHistoricalRun(el.dataset.runId));
        });
    } catch (err) {
        list.innerHTML = `<span style="color:#ff9099">${err}</span>`;
    }
}

async function loadHistoricalRun(runID) {
    if (isRunning) return;
    setStatus('running', `Chargement du run ${runID}…`);
    let events;
    try {
        events = await window.go.main.App.LoadRun(runID);
    } catch (err) {
        setStatus('error', `Erreur: ${err}`);
        return;
    }

    // Pré-remplir le tableau avec toutes les rules connues en pending,
    // puis appliquer les events du run pour faire passer chacune au statut
    // qui a été enregistré. Cohérent avec le rendering live.
    const allSectionIDs = currentSections.map(s => s.id);
    prepareTableForRun(allSectionIDs);
    let applied = 0;
    for (const ev of events) {
        updateRuleRow(ev);
        applied++;
    }
    const summary = computeSummary(events);
    summary.runId = runID;
    summary.mode = 'historique';
    setStatus('success', summarizeStatus(summary));
}

function computeSummary(events) {
    const s = { skipped: 0, applied: 0, failed: 0, rolledBack: 0, aborted: false, cancelled: false };
    for (const ev of events) {
        if (ev.status === 'skipped' || ev.status === 'would_skip') s.skipped++;
        else if (ev.status === 'applied' || ev.status === 'would_apply') s.applied++;
        else if (ev.status === 'failed' || ev.status === 'would_fail') s.failed++;
        else if (ev.status === 'rolled_back') s.rolledBack++;
    }
    return s;
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

    // Filtres
    $$('.filter-severity').forEach(cb => cb.addEventListener('change', applyFilters));
    $$('.filter-status').forEach(cb => cb.addEventListener('change', applyFilters));
    $('#filter-reset').addEventListener('click', () => {
        $$('.filter-severity').forEach(cb => cb.checked = true);
        $$('.filter-status').forEach(cb => {
            cb.checked = (cb.value !== 'conforme');
        });
        applyFilters();
    });

    // Boutons Tout / Aucun pour les sections.
    $('#sections-all').addEventListener('click', () => {
        $$('input[name="section"]').forEach(cb => cb.checked = true);
    });
    $('#sections-none').addEventListener('click', () => {
        $$('input[name="section"]').forEach(cb => cb.checked = false);
    });

    // Cocher / décocher individuellement une règle dans le tableau.
    document.body.addEventListener('change', (e) => {
        if (e.target.matches('input.include-rule')) {
            const ruleID = e.target.dataset.ruleId;
            if (e.target.checked) {
                excludedRules.delete(ruleID);
            } else {
                excludedRules.add(ruleID);
            }
            const tr = rowsByRuleID[ruleID];
            if (tr) tr.classList.toggle('excluded', !e.target.checked);
        }
        if (e.target === $('#include-all')) {
            const checked = e.target.checked;
            $$('input.include-rule').forEach(cb => {
                cb.checked = checked;
                cb.dispatchEvent(new Event('change', { bubbles: true }));
            });
        }
    });
}

// ─────────────────────────────────────────────────────────────────
// Filtres
// ─────────────────────────────────────────────────────────────────

// statusBucket : regroupe les statuses techniques en 4 catégories user-facing
// pour le filtre. Cohérent avec les checkboxes dans index.html.
function statusBucket(status) {
    if (status === 'pending') return 'pending';
    if (status === 'would_skip' || status === 'skipped') return 'conforme';
    if (status === 'would_apply' || status === 'applied') return 'to-apply';
    if (status === 'would_fail' || status === 'failed' || status === 'rolled_back') return 'failed';
    return '';
}

function applyFilters() {
    const sevAllowed = new Set(Array.from($$('.filter-severity:checked')).map(cb => cb.value));
    const stsAllowed = new Set(Array.from($$('.filter-status:checked')).map(cb => cb.value));

    const rows = $$('#results-body tr.row');
    let visible = 0;
    rows.forEach(tr => {
        const ruleID = tr.dataset.ruleId;
        const rule = rulesByID[ruleID] || {};
        const status = tr.dataset.status || 'pending';
        const bucket = statusBucket(status);

        let show = true;
        if (!sevAllowed.has(rule.severity || 'nice-to-have')) show = false;
        if (!stsAllowed.has(bucket)) show = false;
        tr.style.display = show ? '' : 'none';
        if (show) visible++;
    });
    const total = rows.length;
    $('#filter-count').textContent = visible === total
        ? `${total} règle(s)`
        : `${visible} / ${total} règle(s) affichée(s)`;
}

// ─────────────────────────────────────────────────────────────────
// Dashboard maturité
// ─────────────────────────────────────────────────────────────────

// renderDashboard : calcule + affiche le score de maturité et la
// répartition par niveau de sévérité, à partir des events reçus.
//
// Score = % de règles vérifiées qui sont conformes. On exclut :
//   - les 'pending' (pas encore évaluées)
//   - les 'failed/would_fail' (état inconnu — admin requis souvent)
// renderDashboard : 1 ligne synthétique qui guide vers l'action.
// Priorité : compter les règles à renforcer par sévérité, focus sur les
// critiques (les plus impactantes pour la sécurité).
function renderDashboard() {
    const dashboard = $('#dashboard');
    const rows = $$('#results-body tr.row');
    if (rows.length === 0) {
        dashboard.classList.add('hidden');
        return;
    }

    // Compter les règles à renforcer (would_apply / applied) par sévérité.
    let toApply = { critical: 0, important: 0, 'nice-to-have': 0 };
    let evaluatedAll = 0, conformeAll = 0;

    rows.forEach(tr => {
        const rule = rulesByID[tr.dataset.ruleId] || {};
        const sev = rule.severity || 'nice-to-have';
        const bucket = statusBucket(tr.dataset.status || 'pending');
        if (bucket === 'pending' || bucket === 'failed') return;
        evaluatedAll++;
        if (bucket === 'conforme') {
            conformeAll++;
        } else if (bucket === 'to-apply' && toApply[sev] !== undefined) {
            toApply[sev]++;
        }
    });

    if (evaluatedAll === 0) {
        dashboard.classList.add('hidden');
        return;
    }

    dashboard.classList.remove('hidden');
    dashboard.className = '';   // reset classes

    const total = toApply.critical + toApply.important + toApply['nice-to-have'];
    const detail = `${conformeAll}/${evaluatedAll} déjà OK`;

    if (total === 0) {
        dashboard.classList.add('level-ok');
        $('#dash-icon').textContent = '✓';
        $('#dash-headline').textContent = 'Système conforme — toutes les règles évaluées sont OK';
        $('#dash-detail').textContent = detail;
        return;
    }

    // Construit la headline en priorisant le critique.
    let headline = '';
    let level = 'level-light';
    let icon = '⚪';

    if (toApply.critical > 0) {
        headline = `${toApply.critical} règle${toApply.critical > 1 ? 's' : ''} critique${toApply.critical > 1 ? 's' : ''} à renforcer`;
        level = 'level-critical';
        icon = '🔴';
    } else if (toApply.important > 0) {
        headline = `${toApply.important} règle${toApply.important > 1 ? 's' : ''} importante${toApply.important > 1 ? 's' : ''} à renforcer`;
        level = 'level-medium';
        icon = '🟡';
    } else {
        headline = `${toApply['nice-to-have']} règle${toApply['nice-to-have'] > 1 ? 's' : ''} optionnelle${toApply['nice-to-have'] > 1 ? 's' : ''} à renforcer`;
        level = 'level-light';
        icon = '⚪';
    }

    // Si plusieurs niveaux ont des trucs à faire, ajouter un suffixe discret.
    const others = [];
    if (toApply.critical > 0 && (toApply.important > 0 || toApply['nice-to-have'] > 0)) {
        if (toApply.important > 0) others.push(`${toApply.important} importante${toApply.important > 1 ? 's' : ''}`);
        if (toApply['nice-to-have'] > 0) others.push(`${toApply['nice-to-have']} optionnelle${toApply['nice-to-have'] > 1 ? 's' : ''}`);
    } else if (toApply.important > 0 && toApply['nice-to-have'] > 0) {
        others.push(`${toApply['nice-to-have']} optionnelle${toApply['nice-to-have'] > 1 ? 's' : ''}`);
    }
    if (others.length > 0) {
        headline += ` (+ ${others.join(', ')})`;
    }

    dashboard.classList.add(level);
    $('#dash-icon').textContent = icon;
    $('#dash-headline').textContent = headline;
    $('#dash-detail').textContent = detail;
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
        const auditMode = $('#audit-mode').checked;
        const excluded = Array.from(excludedRules);
        const summary = mode === 'apply'
            ? await window.go.main.App.Apply(sections, currentProfile, auditMode, excluded)
            : await window.go.main.App.DryRun(sections, currentProfile, auditMode, excluded);
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
    Object.keys(eventByRuleID).forEach(k => delete eventByRuleID[k]);
    processedRules = 0;
    totalRulesInRun = 0;

    for (const s of currentSections) {
        if (!sectionIDs.includes(s.id)) continue;
        for (const r of (s.rules || [])) {
            const tr = renderRuleRow(r, 'pending', null);
            tbody.appendChild(tr);
            rowsByRuleID[r.id] = tr;
            totalRulesInRun++;
        }
    }
    applyFilters();
    if (totalRulesInRun === 0) {
        tbody.innerHTML = '<tr class="empty"><td colspan="5">Aucune règle dans la sélection.</td></tr>';
    }
    // Cacher le dashboard tant qu'on n'a pas évalué.
    $('#dashboard').classList.add('hidden');
}

function renderRuleRow(rule, status, ev) {
    const tr = document.createElement('tr');
    tr.className = 'row';
    tr.dataset.ruleId = rule.id;
    tr.dataset.status = status;
    const severity = rule.severity || 'nice-to-have';
    const excluded = excludedRules.has(rule.id);
    if (excluded) tr.classList.add('excluded');
    tr.innerHTML = `
        <td class="col-include"><input type="checkbox" class="include-rule" data-rule-id="${escapeHtml(rule.id)}" ${excluded ? '' : 'checked'} title="Décocher pour exclure cette règle"></td>
        <td><span class="severity ${severity}">${humanSeverity(severity)}</span></td>
        <td class="rule-name">
            ${escapeHtml(rule.title || rule.id)}
            <span class="rule-id-tech">${escapeHtml(rule.id)}</span>
        </td>
        <td><span class="status ${status}">${escapeHtml(humanStatus(status, rule.id))}</span></td>
        <td class="action-cell">${formatActionCell(rule, status, ev)}</td>
    `;
    return tr;
}

function updateRuleRow(ev) {
    const ruleID = ev.rule_id;
    eventByRuleID[ruleID] = ev;   // mémoriser pour le tooltip
    const rule = rulesByID[ruleID] || { id: ruleID, title: ruleID, severity: 'nice-to-have' };
    let tr = rowsByRuleID[ruleID];
    if (!tr) {
        tr = renderRuleRow(rule, ev.status, ev);
        $('#results-body').appendChild(tr);
        rowsByRuleID[ruleID] = tr;
        applyFilters();
        renderDashboard();
        return;
    }
    const status = ev.status || 'unknown';
    tr.dataset.status = status;
    const statusCell = tr.querySelector('.status');
    statusCell.className = `status ${status}`;
    statusCell.textContent = humanStatus(status, ruleID);
    tr.querySelector('.action-cell').innerHTML = formatActionCell(rule, status, ev);
    // ré-applique les filtres pour le cas où la rule devient (in)visible avec son nouveau status.
    applyFilters();
    renderDashboard();
    // Scroll seulement si la row est visible (sinon le filtre la cache et scroll est inutile).
    if (tr.style.display !== 'none') {
        tr.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
}

// formatActionCell — texte user-friendly pour la colonne "Action proposée".
//
// L'idée : pas de JSON brut. On dit clairement ce qui se passerait pour la
// règle dans son état actuel.
function formatActionCell(rule, status, ev) {
    if (status === 'pending') {
        return `<span class="action-icon pending">○</span><span class="action-text">Pas encore vérifiée</span>`;
    }
    if (status === 'would_fail' || status === 'failed') {
        const err = ev && ev.error ? truncate(ev.error, 200) : 'erreur inconnue';
        return `<span class="action-icon fail">✗</span><span class="action-text">Vérification impossible</span>
                <span class="action-state">${escapeHtml(err)}</span>`;
    }
    if (status === 'rolled_back') {
        return `<span class="action-icon fail">↶</span><span class="action-text">Action a échoué → rollback exécuté</span>`;
    }
    const isBloatware = rule.id && rule.id.startsWith('bloatware.');
    if (status === 'would_skip' || status === 'skipped') {
        const txt = isBloatware ? 'Pas installée — rien à faire' : 'Aucune action — déjà conforme';
        return `<span class="action-icon ok">✓</span><span class="action-text">${txt}</span>`;
    }
    if (status === 'would_apply') {
        const desc = rule.description || 'modification système';
        const stateBlurb = ev && ev.current_state ? humanStateBlurb(ev.current_state) : '';
        const breaksBadge = rule.breaks && rule.breaks.length > 0
            ? `<span class="breaks-badge" title="${escapeHtml(rule.breaks.join('  •  '))}">⚠ casse si…</span>`
            : '';
        const verb = isBloatware ? 'À supprimer' : 'À renforcer';
        return `<span class="action-icon warn">⚠</span><span class="action-text">${verb} : ${escapeHtml(desc)}</span> ${breaksBadge}
                ${stateBlurb ? `<span class="action-state">État actuel : ${stateBlurb}</span>` : ''}`;
    }
    if (status === 'applied') {
        const txt = isBloatware ? 'Désinstallée ✓' : 'Appliquée avec succès';
        return `<span class="action-icon ok">✓</span><span class="action-text">${txt}</span>`;
    }
    return '';
}

// humanStateBlurb : transforme un current_state {Foo: 1, Bar: "Disabled"}
// en un texte court "Foo=1, Bar=Disabled" (déjà plus lisible que JSON brut).
function humanStateBlurb(state) {
    if (state === null || state === undefined) return '<em>non défini</em>';
    if (typeof state !== 'object') return `<code>${escapeHtml(String(state))}</code>`;
    const entries = Object.entries(state).slice(0, 3);
    return entries.map(([k, v]) => {
        const vs = formatStateValue(v);
        return `<code>${escapeHtml(k)}=${escapeHtml(vs)}</code>`;
    }).join(' ');
}

function formatStateValue(v) {
    if (v === null || v === undefined) return 'absent';
    if (typeof v === 'boolean') return v ? 'oui' : 'non';
    if (typeof v === 'object') return truncate(JSON.stringify(v), 40);
    return truncate(String(v), 40);
}

function humanStatus(status, ruleID) {
    const isBloatware = ruleID && ruleID.startsWith('bloatware.');
    if (isBloatware) {
        if (status === 'would_skip' || status === 'skipped') return 'Pas installée';
        if (status === 'would_apply') return 'À supprimer';
        if (status === 'applied') return 'Désinstallée ✓';
    }
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
    const ruleID = tr.dataset.ruleId;
    const rule = rulesByID[ruleID];
    if (!rule) return hideTooltip();
    const status = tr.dataset.status || 'pending';
    const ev = eventByRuleID[ruleID];
    const currentState = ev && ev.current_state ? ev.current_state : null;
    showTooltip(rule, status, currentState);
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

function showTooltip(rule, status, currentState) {
    const tt = $('#rule-tooltip');
    const irreversibleSection = rule.irreversible
        ? `<div class="tt-irreversible">⚠ Irréversible : ${escapeHtml(rule.irreversibleReason || 'Cette règle ne peut pas être annulée par undo.')}</div>`
        : '';
    const explanationSection = rule.explanation
        ? `<div class="tt-section"><div class="tt-label">Pourquoi cette règle</div>${escapeHtml(rule.explanation).replace(/\n/g, '<br>')}</div>`
        : '';
    const rebootSection = rule.requiresReboot
        ? '<div class="tt-section" style="color:#ffd770;font-size:11px">⚙ Nécessite un redémarrage après application</div>'
        : '';

    // Comparatif avant/après si on a un current_state.
    let comparisonSection = '';
    if (status && status !== 'pending') {
        const stateBlurb = currentState ? humanStateBlurb(currentState) : '';
        const isCompliant = (status === 'would_skip' || status === 'skipped' || status === 'applied');

        if (isCompliant) {
            comparisonSection = `
                <div class="tt-row">
                    <span class="tt-key">Maintenant</span>
                    <span class="tt-val tt-current">${stateBlurb || '—'}</span>
                </div>
                <div class="tt-row">
                    <span class="tt-key">État cible</span>
                    <span class="tt-val tt-target">✓ Déjà atteint</span>
                </div>`;
        } else if (status === 'would_apply') {
            comparisonSection = `
                <div class="tt-row">
                    <span class="tt-key">Maintenant</span>
                    <span class="tt-val tt-current">${stateBlurb || '<em>non protégé</em>'}</span>
                </div>
                <div class="tt-row">
                    <span class="tt-key">Si appliquée</span>
                    <span class="tt-val tt-target">${escapeHtml(rule.description || 'Activation de la protection')}</span>
                </div>
                <div class="tt-row">
                    <span class="tt-key">Bénéfice</span>
                    <span class="tt-val tt-benefit">${escapeHtml(extractBenefit(rule.explanation) || rule.description || '—')}</span>
                </div>`;
        } else if (status === 'would_fail' || status === 'failed') {
            comparisonSection = `
                <div class="tt-row">
                    <span class="tt-key">Maintenant</span>
                    <span class="tt-val tt-current">Vérification impossible — souvent admin requis</span>
                </div>`;
        } else if (status === 'rolled_back') {
            comparisonSection = `
                <div class="tt-row">
                    <span class="tt-key">Action</span>
                    <span class="tt-val tt-side">A planté → rollback exécuté</span>
                </div>`;
        }
    }

    const breaksSection = rule.breaks && rule.breaks.length > 0
        ? `<div class="tt-section tt-breaks">
              <div class="tt-label tt-breaks-label">⚠ Casse si tu utilises</div>
              <ul class="tt-breaks-list">${rule.breaks.map(b => `<li>${escapeHtml(b)}</li>`).join('')}</ul>
           </div>`
        : '';

    tt.innerHTML = `
        <h4>${escapeHtml(rule.title)} <span class="severity ${rule.severity}" style="margin-left:6px;font-size:9px;vertical-align:middle">${escapeHtml(humanSeverity(rule.severity))}</span></h4>
        <div class="tt-desc">${escapeHtml(rule.description)}</div>
        ${comparisonSection}
        ${explanationSection}
        <div class="tt-section">
            <div class="tt-label">Impact concret si appliquée</div>
            <span class="tt-impact">${escapeHtml(rule.impact || '—')}</span>
        </div>
        ${breaksSection}
        ${rebootSection}
        ${irreversibleSection}
    `;
    tt.classList.remove('hidden');
}

// extractBenefit : sortir 1 phrase du explanation pour le panneau "Bénéfice".
// Heuristique simple : 1re phrase qui contient un verbe protecteur.
function extractBenefit(explanation) {
    if (!explanation) return null;
    const sentences = explanation.split(/(?<=[.!?])\s+/);
    const protective = /(bloque|emp[êe]che|prot[èe]ge|d[ée]tecte|active|interdit|refuse|isole|coupe|s[ée]curise|durcit)/i;
    for (const s of sentences) {
        if (protective.test(s)) return s.trim();
    }
    return sentences[0] ? sentences[0].trim() : null;
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
