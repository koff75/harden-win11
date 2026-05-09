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
    await refreshWatchlistAlerts();
    bindEvents();
    bindWailsEvents();
});

async function refreshWatchlistAlerts() {
    try {
        const alerts = await window.go.main.App.GetWatchlistAlerts();
        if (!alerts || alerts.length === 0) return;
        const banner = $('#watchlist-banner');
        const totalEvents = alerts.reduce((sum, a) => sum + (a.countSeen || 0), 0);
        const sources = new Set(alerts.map(a => a.logName));
        $('#watchlist-summary').textContent =
            `${totalEvents} event(s) sur ${sources.size} source(s) — clique pour voir le détail.`;
        banner.classList.remove('hidden');
        $('#btn-watchlist-detail').addEventListener('click', () => showWatchlistModal(alerts));
        $('#btn-watchlist-dismiss').addEventListener('click', () => banner.classList.add('hidden'));
    } catch (err) {
        console.warn('GetWatchlistAlerts:', err);
    }
}

function showWatchlistModal(alerts) {
    const rows = alerts.map(a => `
        <tr>
            <td><span class="muted small">${escapeHtml(a.runId || '')}</span></td>
            <td><strong>${escapeHtml(a.logName)}</strong>${a.provider ? `<br><span class="muted small">${escapeHtml(a.provider)}</span>` : ''}</td>
            <td style="text-align:right">${a.countSeen}</td>
            <td>${escapeHtml(a.reason || '')}</td>
        </tr>`).join('');
    const html = `
        <div class="cov-modal" id="watchlist-modal-overlay">
            <div class="cov-modal-content" style="max-width:840px">
                <span class="cov-close" id="watchlist-modal-close">✕</span>
                <h3>Anomalies Event Viewer post-apply</h3>
                <p class="muted small">Détectées par la watchlist 24h après tes derniers apply. Si une source est en pic, c'est qu'une règle harden-win11 casse peut-être quelque chose que tu utilises.</p>
                <table>
                    <thead><tr><th>Run</th><th>Source</th><th>Events</th><th>Pourquoi c'est suspect</th></tr></thead>
                    <tbody>${rows}</tbody>
                </table>
                <p class="muted small" style="margin-top:12px">
                    Pour aller plus loin : <code>Get-WinEvent -LogName '&lt;source&gt;' -MaxEvents 50</code>
                    pour voir les events bruts. Tu peux aussi lancer <code>harden-engine snapshot diff &lt;runID&gt;</code>
                    pour voir ce qui a été modifié pendant le run.
                </p>
            </div>
        </div>`;
    document.body.insertAdjacentHTML('beforeend', html);
    const overlay = $('#watchlist-modal-overlay');
    const close = () => overlay.remove();
    $('#watchlist-modal-close').addEventListener('click', close);
    overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
}

// Coverage standards (CIS / ANSSI / MS) panel removed — moved to CLI only :
// `harden-engine.exe coverage` shows the same data. The GUI bar was visual
// noise that didn't help the end user decide what to do.

async function refreshProfiles() {
    try {
        availableProfiles = await window.go.main.App.GetProfiles();
    } catch (err) {
        $('#profile-list').innerHTML = `<span style="color:#ff9099">Erreur: ${err}</span>`;
        return;
    }

    // Détection auto du contexte → suggestion de profil + auto-exclusion
    // des rules qui casseraient quelque chose sur cette machine spécifique.
    let suggestion = null;
    try {
        suggestion = await window.go.main.App.DetectContext();
        if (suggestion && suggestion.suggestedProfile) {
            currentProfile = suggestion.suggestedProfile;
        }
        if (suggestion && Array.isArray(suggestion.autoSkipRules)) {
            for (const entry of suggestion.autoSkipRules) {
                excludedRules.add(entry.ruleId);
            }
        }
    } catch (err) {
        // pas grave : on garde le défaut 'personal'.
    }

    const autoSkipBlock = (suggestion && suggestion.autoSkipRules && suggestion.autoSkipRules.length)
        ? `<div class="auto-skip-block">
             ${suggestion.autoSkipRules.length} règle(s) pré-décochée(s) automatiquement :
             <ul>${suggestion.autoSkipRules.map(e =>
                `<li><code>${escapeHtml(e.ruleId)}</code> — ${escapeHtml(e.reason)}</li>`
             ).join('')}</ul>
           </div>`
        : '';
    const suggestionBanner = suggestion
        ? `<div class="profile-suggestion" title="${escapeHtml(suggestion.reason)}">
              💡 Suggéré : <strong>${escapeHtml(profileTitleFromID(suggestion.suggestedProfile))}</strong>
              <div class="profile-suggestion-reason">${escapeHtml(suggestion.reason)}</div>
              ${autoSkipBlock}
           </div>`
        : '';

    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    $('#profile-list').innerHTML = suggestionBanner + availableProfiles.map(p => {
        const title = lang === 'en' ? (p.titleEn || p.title) : p.title;
        const desc  = lang === 'en' ? (p.descriptionEn || p.description) : p.description;
        return `
        <label class="profile-item" title="${escapeHtml(desc)}">
            <input type="radio" name="profile" value="${p.id}" ${p.id === currentProfile ? 'checked' : ''}>
            <span class="profile-title">${escapeHtml(title)}</span>
            <span class="profile-desc">${escapeHtml(desc)}</span>
        </label>
    `;}).join('');
    $$('input[name="profile"]').forEach(rb => {
        rb.addEventListener('change', async () => {
            currentProfile = rb.value;
            await refreshSections();
            const lang = (typeof getLang === 'function') ? getLang() : 'fr';
            const msg = lang === 'en'
                ? 'Profile changed. Run a dry-run to see the current state.'
                : 'Profil changé. Lance un dry-run pour voir l\'état actuel.';
            $('#results-body').innerHTML = `<tr class="empty"><td colspan="5">${msg}</td></tr>`;
            $('#dashboard').classList.add('hidden');
            Object.keys(rowsByRuleID).forEach(k => delete rowsByRuleID[k]);
            Object.keys(eventByRuleID).forEach(k => delete eventByRuleID[k]);
        });
    });
}

function profileTitleFromID(id) {
    const found = availableProfiles.find(p => p.id === id);
    if (!found) return id;
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    return lang === 'en' ? (found.titleEn || found.title) : found.title;
}

async function refreshEngineInfo() {
    try {
        engineInfo = await window.go.main.App.GetEngineInfo();
    } catch (err) {
        $('#engine-info').innerHTML = `<span style="color:#ff9099">init error: ${err}</span>`;
        return;
    }
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    const adminLabel = engineInfo.isAdmin
        ? '<span style="color:#aaffbb">admin ✓</span>'
        : `<span style="color:#ff9099">${lang === 'en' ? 'non-admin (apply disabled)' : 'non-admin (apply désactivé)'}</span>`;
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
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    list.innerHTML = currentSections.map(s => {
        const title = lang === 'en' ? (s.titleEn || s.title) : s.title;
        const desc  = lang === 'en' ? (s.descriptionEn || s.description) : s.description;
        return `
        <label title="${escapeHtml(desc || '')}">
            <input type="checkbox" name="section" value="${s.id}" checked>
            <span>${escapeHtml(title)}</span>
            <span class="rule-count">${s.ruleCount}</span>
        </label>
    `;}).join('');
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
    $('#btn-undo').addEventListener('click', () => alert(getLang() === 'en'
        ? 'Undo via GUI: not wired yet (use harden-engine.exe undo for now)'
        : "Undo via GUI : à wirer (utilise pour l'instant : harden-engine.exe undo)"));

    // Lang toggle FR/EN. On évite window.location.reload() qui ne re-injecte
    // pas correctement les assets dans WebView2 (assets servis via wails://) ;
    // à la place, on re-render les parties dynamiques in-place et on update
    // les textes statiques du DOM via applyI18nStatic().
    const langBtn = $('#btn-lang-toggle');
    if (langBtn) {
        langBtn.textContent = getLang() === 'fr' ? 'EN' : 'FR';
        langBtn.addEventListener('click', async () => {
            const next = getLang() === 'fr' ? 'en' : 'fr';
            console.log('[i18n] switch from', getLang(), 'to', next);
            setLang(next);
            applyI18nStatic();
            // Re-render des contenus dynamiques (profils + sections re-fetchent
            // depuis Go pour utiliser title_en/description_en).
            await refreshProfiles();
            await refreshSections();
            rerenderAllRows();
            renderDashboard();
            langBtn.textContent = getLang() === 'fr' ? 'EN' : 'FR';
        });
    }
    applyI18nStatic();

    $('#modal-cancel').addEventListener('click', closeModal);
    $('#modal-confirm-btn').addEventListener('click', () => {
        closeModal();
        runEngine('apply');
    });
    $('#btn-cancel').addEventListener('click', () => {
        window.go.main.App.CancelRun();
        $('#loader-title').textContent = 'Annulation en cours…';
    });

    // Tooltip riche (sur cellules Niveau + Règle uniquement) qui suit la souris.
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

// applyI18nStatic : parcourt tous les éléments avec data-i18n="key" et
// remplace leur textContent par t(key). Couvre tous les libellés statiques
// du HTML — labels aside, boutons, filtres, headers de tableau, modal, etc.
// Pour les éléments avec contenu dynamique (admin-banner-text qui mixe
// strong + texte normal), on assemble explicitement.
function applyI18nStatic() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.dataset.i18n;
        if (!key) return;
        el.textContent = t(key);
    });
    // Cas spécial admin-banner : structure HTML mixte.
    const adminText = document.querySelector('#admin-banner .admin-banner-text');
    if (adminText) {
        adminText.innerHTML = `<strong>${escapeHtml(t('admin.notadmin'))}</strong> ${escapeHtml(t('admin.banner'))}`;
    }
    // Bouton lang : afficher la langue VERS laquelle on switche.
    const langBtn = document.querySelector('#btn-lang-toggle');
    if (langBtn) langBtn.textContent = getLang() === 'fr' ? 'EN' : 'FR';
}

// rerenderAllRows : ré-applique humanStatus / formatActionCell / tooltip à
// toutes les lignes existantes — utile après un switch de langue pour que
// les statuts ("Pas installée" / "Not installed", etc.) et le tooltip
// passent dans la nouvelle langue.
function rerenderAllRows() {
    const rows = document.querySelectorAll('#results-body tr.row');
    rows.forEach(tr => {
        const ruleID = tr.dataset.ruleId;
        const rule = rulesByID[ruleID];
        const ev = eventByRuleID[ruleID];
        const status = tr.dataset.status || 'pending';
        if (!rule) return;
        const statusCell = tr.querySelector('.status');
        if (statusCell) statusCell.textContent = humanStatus(status, ruleID);
        const actionCell = tr.querySelector('.action-cell');
        if (actionCell) actionCell.innerHTML = formatActionCell(rule, status, ev);
    });
}

async function showMaturityModal() {
    const rows = $$('#results-body tr.row');
    const counts = {
        critical: { total: 0, compliant: 0 },
        important: { total: 0, compliant: 0 },
        'nice-to-have': { total: 0, compliant: 0 },
        undetermined: 0,
    };
    rows.forEach(tr => {
        const rule = rulesByID[tr.dataset.ruleId] || {};
        const sev = rule.severity || 'nice-to-have';
        const bucket = statusBucket(tr.dataset.status || 'pending');
        if (bucket === 'pending') return;
        if (bucket === 'failed') { counts.undetermined++; return; }
        if (!counts[sev]) return;
        counts[sev].total++;
        if (bucket === 'conforme') counts[sev].compliant++;
    });
    let report;
    try {
        report = await window.go.main.App.ComputeMaturity({
            criticalTotal: counts.critical.total,
            criticalCompliant: counts.critical.compliant,
            importantTotal: counts.important.total,
            importantCompliant: counts.important.compliant,
            niceTotal: counts['nice-to-have'].total,
            niceCompliant: counts['nice-to-have'].compliant,
            undeterminedCount: counts.undetermined,
        });
    } catch (err) {
        alert('Impossible de calculer le score : ' + err);
        return;
    }
    const compsHtml = report.components.map(c => {
        const pct = c.weight > 0 ? Math.round(c.earned / c.weight * 100) : 0;
        const cls = c.status === 'ok' ? 'comp-ok' : c.status === 'partial' ? 'comp-partial' : 'comp-missing';
        return `
            <tr class="${cls}">
                <td>${escapeHtml(c.name)}</td>
                <td style="text-align:right">${Math.round(c.earned)} / ${c.weight}</td>
                <td><div class="comp-bar"><div class="comp-bar-fill" style="width:${pct}%"></div></div></td>
                <td><span class="muted small">${escapeHtml(c.detail || '')}</span></td>
            </tr>`;
    }).join('');
    const actionsHtml = (report.next_actions || []).map(a => `<li>${escapeHtml(a)}</li>`).join('');
    const html = `
        <div class="cov-modal" id="maturity-modal-overlay">
            <div class="cov-modal-content" style="max-width:780px">
                <span class="cov-close" id="maturity-modal-close">✕</span>
                <h3>📊 Score de maturité</h3>
                <div class="maturity-grade-block grade-${report.grade}">
                    <div class="maturity-grade">${report.grade}</div>
                    <div class="maturity-score">${report.score}<span class="muted small"> / 100</span></div>
                    <div class="maturity-headline">${escapeHtml(report.headline)}</div>
                </div>
                <table class="maturity-table">
                    <thead><tr><th>Composant</th><th>Points</th><th>Niveau</th><th>Détail</th></tr></thead>
                    <tbody>${compsHtml}</tbody>
                </table>
                ${actionsHtml ? `<h4 style="margin-top:18px">Pour gagner du score</h4><ol>${actionsHtml}</ol>` : ''}
                <p class="muted small" style="margin-top:14px"><em>Pondération : critique 50, important 25, optionnel 10, Restore Point 8, watchlist 7. Total max : 100.</em></p>
            </div>
        </div>`;
    document.body.insertAdjacentHTML('beforeend', html);
    const overlay = $('#maturity-modal-overlay');
    const close = () => overlay.remove();
    $('#maturity-modal-close').addEventListener('click', close);
    overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
}

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
    $('#btn-maturity').classList.remove('hidden');
    if (!$('#btn-maturity').dataset.bound) {
        $('#btn-maturity').addEventListener('click', showMaturityModal);
        $('#btn-maturity').dataset.bound = '1';
    }

    const total = toApply.critical + toApply.important + toApply['nice-to-have'];
    const detail = `${conformeAll}/${evaluatedAll} déjà OK`;

    if (total === 0) {
        dashboard.classList.add('level-ok');
        $('#dash-icon').textContent = '✓';
        $('#dash-headline').textContent = t('dashboard.allok');
        $('#dash-detail').textContent = detail;
        return;
    }

    // Headline vulgarisée : on met d'abord le positif (X points OK) puis
    // les améliorations possibles. Plus engageant qu'un nb brut de "règles
    // à renforcer".
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    const en = lang === 'en';
    let headline = '';
    let level = 'level-light';
    let icon = '✓';
    const totalNonCompliant = toApply.critical + toApply.important + toApply['nice-to-have'];

    const plural = (n, fr_s, fr_ms, en_s, en_ms) => en ? (n > 1 ? en_ms : en_s) : (n > 1 ? fr_ms : fr_s);

    if (toApply.critical > 0) {
        const detailExtras = [];
        if (toApply.important > 0) {
            detailExtras.push(en
                ? `${toApply.important} important`
                : `${toApply.important} importante${toApply.important > 1 ? 's' : ''}`);
        }
        if (toApply['nice-to-have'] > 0) {
            detailExtras.push(en
                ? `${toApply['nice-to-have']} optional`
                : `${toApply['nice-to-have']} optionnelle${toApply['nice-to-have'] > 1 ? 's' : ''}`);
        }
        const join = en ? ', ' : ' et ';
        const extras = detailExtras.length > 0 ? `, ${detailExtras.join(join)}` : '';
        if (en) {
            headline = `Your machine is OK on ${conformeAll} points. ${totalNonCompliant} improvement${totalNonCompliant > 1 ? 's' : ''} possible: ${toApply.critical} critical${extras}.`;
        } else {
            headline = `Ta machine est OK sur ${conformeAll} points. ${totalNonCompliant} amélioration${totalNonCompliant > 1 ? 's' : ''} possible${totalNonCompliant > 1 ? 's' : ''} : ${toApply.critical} critique${toApply.critical > 1 ? 's' : ''}${extras}.`;
        }
        level = 'level-critical';
        icon = '🔴';
    } else if (toApply.important > 0) {
        const optExtra = toApply['nice-to-have'] > 0
            ? (en
                ? ` and ${toApply['nice-to-have']} optional`
                : ` et ${toApply['nice-to-have']} optionnelle${toApply['nice-to-have'] > 1 ? 's' : ''}`)
            : '';
        if (en) {
            headline = `Your machine is well protected. ${toApply.important} important improvement${toApply.important > 1 ? 's' : ''}${optExtra} possible.`;
        } else {
            headline = `Ta machine est bien protégée. ${toApply.important} amélioration${toApply.important > 1 ? 's' : ''} importante${toApply.important > 1 ? 's' : ''}${optExtra} possible${(toApply.important + toApply['nice-to-have']) > 1 ? 's' : ''}.`;
        }
        level = 'level-medium';
        icon = '🟡';
    } else {
        if (en) {
            headline = `Your machine is solid. ${toApply['nice-to-have']} optional improvement${toApply['nice-to-have'] > 1 ? 's' : ''} possible if you want to push further.`;
        } else {
            headline = `Ta machine est solide. ${toApply['nice-to-have']} amélioration${toApply['nice-to-have'] > 1 ? 's' : ''} optionnelle${toApply['nice-to-have'] > 1 ? 's' : ''} possible si tu veux pousser plus loin.`;
        }
        level = 'level-light';
        icon = '⚪';
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
        setStatus('error', getLang() === 'en' ? 'Select at least one section.' : 'Sélectionne au moins une section.');
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
        setStatus('error', getLang() === 'en' ? 'Select at least one section.' : 'Sélectionne au moins une section.');
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
        // Remonte en haut pour voir le dashboard + premières règles dès que le run termine.
        window.scrollTo({ top: 0, behavior: 'smooth' });
    });
    window.runtime.EventsOn('restore_point_started', () => {
        const en = getLang() === 'en';
        setStatus('running', en
            ? 'Creating a Windows Restore Point (may take 30-60s)…'
            : 'Création d\'un Restore Point Windows (peut prendre 30-60s)…');
    });
    window.runtime.EventsOn('restore_point_done', (payload) => {
        const en = getLang() === 'en';
        if (payload && payload.created) {
            const sec = Math.round((payload.durationMs || 0)/1000);
            setStatus('running', en
                ? `Restore Point created in ${sec}s. Starting apply…`
                : `Restore Point créé en ${sec}s. Démarrage de l'apply…`);
        } else {
            const why = payload && payload.reason ? `(${payload.reason})` : '';
            setStatus('running', en
                ? `Restore Point not created ${why} — apply continues (rollback via journal still available).`
                : `Restore Point non créé ${why} — l'apply continue (rollback via journal disponible).`);
        }
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
        const lang = (typeof getLang === 'function') ? getLang() : 'fr';
        tbody.innerHTML = `<tr class="empty"><td colspan="5">${lang === 'en' ? 'No rules in selection.' : 'Aucune règle dans la sélection.'}</td></tr>`;
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
    // Pas de title natif HTML : le tooltip riche custom (#rule-tooltip)
    // est attaché globalement via mouseover et limité aux cellules Niveau
    // (col 2) et Règle (col 3) par isHoverableCell().
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
    const actionCell = tr.querySelector('.action-cell');
    actionCell.innerHTML = formatActionCell(rule, status, ev);
    // Le tooltip riche custom se rafraîchit automatiquement au mouseover
    // (il lit rulesByID + eventByRuleID en live), pas besoin de refresh ici.
    // ré-applique les filtres pour le cas où la rule devient (in)visible avec son nouveau status.
    applyFilters();
    renderDashboard();
    // Pas de scrollIntoView ici : le user veut rester en haut de la page,
    // le scroll-to-top est fait dans run_end (cf. bindWailsEvents).
}

// formatActionCell — texte user-friendly pour la colonne "Action proposée".
//
// L'idée : pas de JSON brut. On dit clairement ce qui se passerait pour la
// règle dans son état actuel.
// (buildUserTooltipText removed — replaced by the rich HTML tooltip
// rendered through showTooltip into <div id="rule-tooltip">.)

// contextualVerb : verbe lisible selon le type de rule. Délègue au système
// i18n pour la traduction.
function contextualVerb(rule) {
    const id = rule.id || '';
    if (id.startsWith('bloatware.')) return t('verb.uninstall');
    if (id.startsWith('asr.') || id.startsWith('defender.')) return t('verb.activate');
    if (id.startsWith('firewall.')) return t('verb.block');
    if (id === 'system_settings.rdp_disable' ||
        id === 'system_settings.hibernate_off' ||
        id === 'system_settings.fast_startup_off' ||
        id === 'privacy.recall_off' ||
        id === 'privacy.cortana_off' ||
        (id.startsWith('privacy.') && id.endsWith('_off')) ||
        id === 'network.llmnr_disable' ||
        id === 'network.mdns_disable' ||
        id === 'network.netbios_off' ||
        id === 'network.wpad_disable' ||
        id === 'network.smbv1_disable' ||
        id === 'network.smb_guest_auth_off') return t('verb.disable');
    if (id.startsWith('network.')) return t('verb.harden');
    if (id.startsWith('system_settings.uac_')) return t('verb.harden_uac');
    if (id.startsWith('accounts.')) return t('verb.secure_acct');
    return t('verb.harden');
}

function formatActionCell(rule, status, ev) {
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    const en = lang === 'en';

    if (status === 'pending') {
        return `<span class="action-icon pending">○</span><span class="action-text">${escapeHtml(t('cell.notyetchecked'))}</span>`;
    }
    if (status === 'would_fail' || status === 'failed') {
        const err = ev && ev.error ? truncate(ev.error, 200) : (en ? 'unknown error' : 'erreur inconnue');
        return `<span class="action-icon fail">✗</span><span class="action-text">${escapeHtml(t('cell.checkimpossible'))}</span>
                <span class="action-state">${escapeHtml(err)}</span>`;
    }
    if (status === 'rolled_back') {
        return `<span class="action-icon fail">↶</span><span class="action-text">${escapeHtml(t('cell.actionfailed'))}</span>`;
    }
    const isBloatware = rule.id && rule.id.startsWith('bloatware.');
    if (status === 'would_skip' || status === 'skipped') {
        const txt = isBloatware ? t('cell.notinstalled') : t('cell.compliant');
        return `<span class="action-icon ok">✓</span><span class="action-text">${escapeHtml(txt)}</span>`;
    }
    if (status === 'would_apply') {
        // Cellule courte : verbe contextuel + 1 ligne synthétique.
        const verb = contextualVerb(rule);
        const userAfter = en ? (rule.userAfterEn || rule.userAfter) : rule.userAfter;
        const synth = userAfter || rule.title || (en ? 'system change' : 'modification système');
        return `<span class="action-icon warn">⚠</span><span class="action-text"><strong>${escapeHtml(verb)}</strong> · ${escapeHtml(synth)}</span>`;
    }
    if (status === 'applied') {
        const txt = isBloatware ? t('cell.uninstalled') : t('cell.appliedsuccess');
        return `<span class="action-icon ok">✓</span><span class="action-text">${escapeHtml(txt)}</span>`;
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
        if (status === 'would_skip' || status === 'skipped') return t('status.bloat.notinst');
        if (status === 'would_apply') return t('status.bloat.toremove');
        if (status === 'applied') return t('status.bloat.uninstalled');
    }
    const map = {
        'pending':     t('status.pending'),
        'would_skip':  t('status.compliant'),
        'would_apply': t('status.toapply'),
        'would_fail':  t('status.tofail'),
        'skipped':     t('status.compliant'),
        'applied':     t('status.applied'),
        'failed':      t('status.failed'),
        'rolled_back': t('status.rolledback'),
    };
    return map[status] || status;
}

function humanSeverity(s) {
    return {
        'critical':     t('filter.critical'),
        'important':    t('filter.important'),
        'nice-to-have': t('filter.nice'),
    }[s] || s;
}

function summarizeStatus(s) {
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';
    const en = lang === 'en';
    const parts = [];
    if (s.skipped) parts.push(`${s.skipped} ${en ? 'already OK' : 'déjà OK'}`);
    if (s.applied) {
        const verb = s.mode === 'apply'
            ? (en ? 'applied' : 'appliquées')
            : (en ? 'to apply' : 'à appliquer');
        parts.push(`${s.applied} ${verb}`);
    }
    if (s.failed) parts.push(`${s.failed} ${en ? 'failure(s)' : 'échec(s)'}`);
    if (s.rolledBack) parts.push(`${s.rolledBack} rollback`);
    let suffix = '';
    if (s.cancelled) suffix = en ? ' [CANCELLED]' : ' [ANNULÉ]';
    else if (s.aborted) suffix = en ? ' [STOPPED after rollback]' : ' [ARRÊTÉ après rollback]';
    return `Run ${s.runId} (${s.mode}) · ${parts.join(' · ')}${suffix}`;
}

function setStatus(kind, message) {
    const el = $('#status-bar');
    el.className = kind;
    el.textContent = message;
}

// ─────────────────────────────────────────────────────────────────
// Tooltip riche au survol (suit la souris). Limité aux cellules
// "Niveau" (colonne 2) et "Règle" (colonne 3).
// ─────────────────────────────────────────────────────────────────

// hoverEnabled : true seulement si e.target est dans une des 2 colonnes
// autorisées. Le user veut explicitement pas de tooltip sur Status/Action.
function isHoverableCell(target) {
    if (!target || !target.closest) return false;
    const td = target.closest('td');
    if (!td) return false;
    const tr = td.closest('tr.row');
    if (!tr) return false;
    // Index de la cellule dans la row (0-based).
    const idx = Array.prototype.indexOf.call(tr.children, td);
    // Colonnes : 0 = include checkbox, 1 = severity (Niveau), 2 = rule-name (Règle),
    // 3 = status (État), 4 = action (Action proposée).
    return idx === 1 || idx === 2;
}

function onRowHover(e) {
    if (!isHoverableCell(e.target)) {
        hideTooltip();
        return;
    }
    const tr = e.target.closest('tr.row');
    const ruleID = tr.dataset.ruleId;
    const rule = rulesByID[ruleID];
    if (!rule) return hideTooltip();
    const status = tr.dataset.status || 'pending';
    const ev = eventByRuleID[ruleID];
    const currentState = ev && ev.current_state ? ev.current_state : null;
    showTooltip(rule, status, currentState);
}

function onRowOut(e) {
    // Cache si on quitte vers un élément qui n'est pas dans une cellule hoverable.
    if (!isHoverableCell(e.relatedTarget)) {
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
    const lang = (typeof getLang === 'function') ? getLang() : 'fr';

    // Champs user-friendly localisés (avec fallback FR si EN absent).
    const today  = lang === 'en' ? (rule.userTodayEn  || rule.userToday)  : rule.userToday;
    const after  = lang === 'en' ? (rule.userAfterEn  || rule.userAfter)  : rule.userAfter;
    const forWho = lang === 'en' ? (rule.userForWhoEn || rule.userForWho) : rule.userForWho;
    const risk   = lang === 'en' ? (rule.userRiskEn   || rule.userRisk)   : rule.userRisk;

    // Bloc principal user-friendly (Aujourd'hui / Si tu actives / Pour qui / Risque).
    let userFriendlySection = '';
    if (today && after) {
        userFriendlySection = `
            <div class="tt-row">
                <span class="tt-key">${escapeHtml(t('tooltip.today'))}</span>
                <span class="tt-val tt-current">${escapeHtml(today)}</span>
            </div>
            <div class="tt-row">
                <span class="tt-key">${escapeHtml(t('tooltip.after'))}</span>
                <span class="tt-val tt-target">${escapeHtml(after)}</span>
            </div>
            ${forWho ? `<div class="tt-row">
                <span class="tt-key">${escapeHtml(t('tooltip.forwho'))}</span>
                <span class="tt-val tt-benefit">${escapeHtml(forWho)}</span>
            </div>` : ''}
            ${risk ? `<div class="tt-row">
                <span class="tt-key">${escapeHtml(t('tooltip.risk'))}</span>
                <span class="tt-val tt-side">${escapeHtml(risk)}</span>
            </div>` : ''}
        `;
    }

    // État actuel observé (après dryrun).
    let currentSection = '';
    if (currentState && status && status !== 'pending') {
        const stateBlurb = humanStateBlurb(currentState);
        if (stateBlurb) {
            currentSection = `<div class="tt-row">
                <span class="tt-key">${escapeHtml(t('tooltip.currentstate'))}</span>
                <span class="tt-val tt-current">${stateBlurb}</span>
            </div>`;
        }
    }

    const breaksSection = rule.breaks && rule.breaks.length > 0
        ? `<div class="tt-section tt-breaks">
              <div class="tt-label tt-breaks-label">${lang === 'en' ? '⚠ Breaks if you use' : '⚠ Casse si tu utilises'}</div>
              <ul class="tt-breaks-list">${rule.breaks.map(b => `<li>${escapeHtml(b)}</li>`).join('')}</ul>
           </div>`
        : '';

    const rebootSection = rule.requiresReboot
        ? `<div class="tt-section" style="color:#ffd770;font-size:11px">⚙ ${lang === 'en' ? 'Reboot required after applying' : 'Redémarrage requis après application'}</div>`
        : '';

    const irreversibleSection = rule.irreversible
        ? `<div class="tt-irreversible">⚠ ${lang === 'en' ? 'Irreversible' : 'Irréversible'} : ${escapeHtml(rule.irreversibleReason || (lang === 'en' ? "This rule can't be reverted via undo." : "Cette règle ne peut pas être annulée par undo."))}</div>`
        : '';

    tt.innerHTML = `
        <h4>${escapeHtml(rule.title)} <span class="severity ${rule.severity}" style="margin-left:6px;font-size:9px;vertical-align:middle">${escapeHtml(humanSeverity(rule.severity))}</span></h4>
        ${userFriendlySection}
        ${currentSection}
        ${breaksSection}
        ${rebootSection}
        ${irreversibleSection}
    `;
    tt.classList.remove('hidden');
}

function hideTooltip() {
    const tt = $('#rule-tooltip');
    if (tt) tt.classList.add('hidden');
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
