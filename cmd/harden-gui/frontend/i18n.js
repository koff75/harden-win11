// i18n.js — Système de traduction simple FR/EN.
//
// Usage : t('label.today') retourne le texte selon currentLang.
// Détection au boot via navigator.language. Persistance via localStorage.

const I18N = {
    fr: {
        // Tooltip labels
        'tooltip.today':       "Aujourd'hui",
        'tooltip.after':       'Si tu actives',
        'tooltip.forwho':      'Pour qui',
        'tooltip.risk':        "Ce qui peut t'embêter",
        'tooltip.description': 'Description',
        'tooltip.currentstate':'État actuel',

        // Verbes contextuels
        'verb.uninstall':   'Désinstaller',
        'verb.activate':    'Activer la protection',
        'verb.block':       'Bloquer',
        'verb.disable':     'Désactiver',
        'verb.harden':      'Renforcer',
        'verb.harden_uac':  'Renforcer UAC',
        'verb.secure_acct': 'Sécuriser le compte',

        // Statuts user-facing
        'status.pending':         'En attente',
        'status.compliant':       'OK (déjà conforme)',
        'status.toapply':         'À appliquer',
        'status.tofail':          'Va échouer (en simulation)',
        'status.applied':         'Appliquée ✓',
        'status.failed':          'Échec',
        'status.rolledback':      'Modif annulée',
        'status.bloat.notinst':   'Pas installée',
        'status.bloat.toremove':  'À supprimer',
        'status.bloat.uninstalled':'Désinstallée ✓',

        // Action cell
        'cell.notyetchecked':   'Pas encore vérifiée',
        'cell.checkimpossible': 'Vérification impossible',
        'cell.actionfailed':    'Windows a refusé la modif — annulée par sécurité',
        'cell.compliant':       'Aucune action — déjà conforme',
        'cell.notinstalled':    'Pas installée — rien à faire',
        'cell.appliedsuccess':  'Appliquée avec succès',
        'cell.uninstalled':     'Désinstallée ✓',
        'cell.canbreakprefix':  '⚠ peut casser…',
        'cell.hoverhint':       'Survole pour voir le détail',

        // Dashboard
        'dashboard.allok':      'Tout est OK — aucune amélioration nécessaire',
        'dashboard.improveImp': 'Ta machine est bien protégée. {n} amélioration{s} importante{s}{opt} possible{ps}.',
        'dashboard.improveOpt': 'Ta machine est solide. {n} amélioration{s} optionnelle{s} possible{s} si tu veux pousser plus loin.',
        'dashboard.improveCrit':'Ta machine est OK sur {ok} points. {n} amélioration{s} possible{s} : {crit} critique{cs}{rest}.',

        // Header / boutons
        'btn.dryrun':  '🔍 Vérifier (sans rien changer)',
        'btn.apply':   '⚙️ Appliquer',
        'btn.cancel':  'Annuler',
        'btn.undo':    '↶ Annuler la dernière application',
        'btn.score':   '📊 Score',
        'btn.langtoggle': 'EN',

        // Misc
        'admin.notadmin':       "Tu n'es pas administrateur.",
        'admin.banner':         'Tu peux explorer et tester sans modifier, mais Appliquer et Annuler nécessitent les droits admin.',
        'admin.relaunch':       'Relancer en admin',

        // Tamper Protection preflight
        'tamper.title':         'Petite étape avant d\'améliorer ton antivirus.',
        'tamper.detail':        ' Windows bloque les modifications de Defender. Dans Sécurité Windows → « Protection contre les virus et menaces » → « Gérer les paramètres » → désactive « Protection contre les falsifications ».',
        'tamper.open':          'Ouvre-moi la page →',

        // Aside
        'aside.profile':         'Profil',
        'aside.sections':        'Sections',
        'aside.actions':         'Actions',
        'aside.history':         'Historique',
        'aside.tout':            'Tout',
        'aside.aucun':           'Aucun',
        'aside.advanced':        'Options avancées',
        'aside.audit_label':     'Mode test (vérifie seulement, sans modifier)',

        // Status bar
        'status.placeholder':    'Sélectionne des sections puis clique Vérifier ou Appliquer.',

        // Filters
        'filter.level':          'Niveau',
        'filter.state':          'État',
        'filter.critical':       'Critique',
        'filter.important':      'Important',
        'filter.nice':           'Optionnel',
        'filter.pending':        'En attente',
        'filter.compliant':      'Conforme',
        'filter.toapply':        'À renforcer',
        'filter.failed':         'Échec',
        'filter.reset':          'Reset',

        // Table
        'table.severity':        'Niveau',
        'table.rule':            'Règle',
        'table.state':           'État',
        'table.action':          'Action proposée',
        'table.empty':           "Aucun résultat encore. Clique « Vérifier » pour voir l'état actuel.",

        // Loader
        'loader.analyzing':      'Analyse en cours…',
        'loader.applying':       'Application en cours…',

        // Modal confirm
        'modal.confirm_title':   '⚠ Appliquer les modifs sur ton PC ?',
        'modal.confirm_text_pre':'Les actions vont',
        'modal.confirm_text_strong':'modifier le système',
        'modal.confirm_text_post':'Tout est enregistré dans un journal — tu peux annuler chaque modif après coup.',
        'modal.affected':        'Sections concernées :',
        'modal.confirm_btn':     'Confirmer et appliquer',

        // Watchlist banner
        'watchlist.title':       'Alertes détectées après une amélioration récente.',
        'watchlist.detail':      'Voir détail',

        // Drift banner (post-Windows Update)
        'drift.title':           'Windows Update a réinitialisé une partie de tes améliorations de sécurité.',
        'drift.reapply':         'Re-appliquer',

        // Erreurs et status messages génériques
        'error.generic':          'Erreur : {msg}',
        'error.selectSections':   'Sélectionne au moins une section.',
        'error.noRulesInSelection':'Aucune règle dans la sélection.',
        'error.adminRelaunch':    'Échec du relancement : {msg}',
        'error.undoFailed':       'Échec annulation : {msg}',
        'status.loading':         'Chargement…',
        'status.loadingRun':      'Chargement de la session {id}…',
        'status.profileChanged':  'Profil changé. Lance une vérification pour voir l\'état actuel.',
        'status.runStart':        'Session {id} ({mode}, {sec} section(s), {n} règle(s))',
        'status.undoing':         'Annulation de {id}…',
        'status.undoOk':          'Annulation OK pour {id}',
        'status.creatingRP':      'Création d\'un Restore Point Windows (peut prendre 30-60s)…',
        'status.rpCreated':       'Restore Point créé en {sec}s. Démarrage de l\'application…',
        'status.rpSkipped':       'Restore Point non créé {why}— l\'application continue (annulation via journal disponible).',
        'status.applyContinues':  'Cliquez « Appliquer » pour ré-appliquer les {n} élément(s) modifié(s).',

        // Loader counters
        'loader.progress':        '{done} / {total} règle(s)',

        // Dashboard
        'dashboard.detailCount':  '{ok}/{n} déjà OK',
        'dashboard.rollbackAlert':'{n} règle(s) a déclenché une annulation automatique. Examine les détails ci-dessous.',
        'dashboard.failedAlert':  '{n} règle(s) ont échoué — voir les détails ci-dessous.',

        // Suggestion banner
        'suggestion.label':       '💡 Suggéré :',
        'suggestion.autoSkipped': '{n} règle(s) pré-décochée(s) automatiquement :',

        // Table footer
        'table.count':            '{n} règle(s)',
        'table.countFiltered':    '{visible} / {total} règle(s) affichée(s)',

        // Watchlist detail
        'watchlist.summary':      '{n} event(s) sur {s} source(s) — clique pour voir le détail.',
        'watchlist.modalTitle':   'Anomalies Event Viewer post-application',
        'watchlist.modalIntro':   'Détectées par la surveillance 24h après tes dernières applications. Si une source est en pic, c\'est qu\'une règle Win11 Hardening casse peut-être quelque chose que tu utilises.',
        'watchlist.thRun':        'Session',
        'watchlist.thSource':     'Source',
        'watchlist.thEvents':     'Événements',
        'watchlist.thReason':     'Pourquoi c\'est suspect',
        'watchlist.modalHelp':    'Pour aller plus loin : <code>Get-WinEvent -LogName \'&lt;source&gt;\' -MaxEvents 50</code> pour voir les events bruts. Tu peux aussi lancer <code>harden-engine snapshot diff &lt;runID&gt;</code> pour voir ce qui a été modifié pendant la session.',

        // Score modal
        'score.title':            '📊 Score de maturité',
        'score.thComponent':      'Composant',
        'score.thPoints':         'Points',
        'score.thLevel':          'Niveau',
        'score.thDetail':         'Détail',
        'score.weighting':        'Pondération : critique 50, important 25, optionnel 10, Restore Point 8, surveillance 7. Total max : 100.',

        // Tooltips
        'tooltip.notdefined':     'non défini',
        'tooltip.rebootRequired': 'Redémarrage requis après application',
        'tooltip.irreversible':   'Irréversible',
        'tooltip.cantUndo':       'Cette règle ne peut pas être annulée.',
        'tooltip.includeRow':     'Décoche pour exclure cette règle',
        'tooltip.undoRun':        'Annuler cette session (revenir sur les règles appliquées)',
        'tooltip.confirmUndo':    'Annuler la session {id} ? Cela va revenir sur toutes les règles appliquées pendant cette session.',

        // Boutons admin (dynamiques côté JS)
        'admin.relaunching':      'Relancement…',
        'admin.applyDisabled':    'Lance la GUI en admin pour activer',

        // Drift summary
        'drift.summary':          '{n} élément(s) modifié(s) depuis la dernière mesure ({base}).',

        // HTML title attributes
        'title.dryrun':           'Lit l\'état système sans rien modifier (sans admin requis)',
        'title.apply':            'Applique réellement les changements (admin requis)',
        'title.undo':             'Annule les changements de la dernière session',
        'title.auditMode':        'Pour les protections Microsoft Defender qui peuvent générer des faux positifs (ASR, Network Protection) : active un mode test pendant 1-2 semaines pour voir ce qui serait bloqué dans Event Viewer (events 1121/1122) sans casser tes apps. Une fois sûr qu\'aucune app légitime n\'est impactée, décoche pour activer le blocage réel.',
        'title.score':            'Voir ton score de maturité hardening (note A/B/C/D)',
        'title.filterReset':      'Réinitialiser les filtres',
        'title.colInclude':       'Décoche pour exclure cette règle de la prochaine session',
        'title.includeAll':       'Tout cocher / décocher',
    },
    en: {
        'tooltip.today':       'Today',
        'tooltip.after':       'If you activate',
        'tooltip.forwho':      'For whom',
        'tooltip.risk':        'What might bother you',
        'tooltip.description': 'Description',
        'tooltip.currentstate':'Current state',

        'verb.uninstall':   'Uninstall',
        'verb.activate':    'Activate protection',
        'verb.block':       'Block',
        'verb.disable':     'Disable',
        'verb.harden':      'Harden',
        'verb.harden_uac':  'Harden UAC',
        'verb.secure_acct': 'Secure account',

        'status.pending':         'Pending',
        'status.compliant':       'OK (already compliant)',
        'status.toapply':         'To apply',
        'status.tofail':          'Would fail (simulation)',
        'status.applied':         'Applied ✓',
        'status.failed':          'Failed',
        'status.rolledback':      'Change reverted',
        'status.bloat.notinst':   'Not installed',
        'status.bloat.toremove':  'To remove',
        'status.bloat.uninstalled':'Uninstalled ✓',

        'cell.notyetchecked':   'Not yet checked',
        'cell.checkimpossible': 'Check impossible',
        'cell.actionfailed':    'Windows refused the change — reverted for safety',
        'cell.compliant':       'No action — already compliant',
        'cell.notinstalled':    'Not installed — nothing to do',
        'cell.appliedsuccess':  'Applied successfully',
        'cell.uninstalled':     'Uninstalled ✓',
        'cell.canbreakprefix':  '⚠ may break…',
        'cell.hoverhint':       'Hover for details',

        'dashboard.allok':      'All OK — no improvement needed',
        'dashboard.improveImp': 'Your machine is well protected. {n} important improvement{s}{opt} possible.',
        'dashboard.improveOpt': 'Your machine is solid. {n} optional improvement{s} possible if you want to push further.',
        'dashboard.improveCrit':'Your machine is OK on {ok} points. {n} improvement{s} possible: {crit} critical{rest}.',

        'btn.dryrun':  '🔍 Check (without changing anything)',
        'btn.apply':   '⚙️ Apply',
        'btn.cancel':  'Cancel',
        'btn.undo':    '↶ Undo last application',
        'btn.score':   '📊 Score',
        'btn.langtoggle': 'FR',

        'admin.notadmin':       "You are not administrator.",
        'admin.banner':         'You can explore and test without modifying, but Apply and Undo require admin rights.',
        'admin.relaunch':       'Relaunch as admin',

        // Tamper Protection preflight
        'tamper.title':         'One quick step before we improve your antivirus.',
        'tamper.detail':        ' Windows blocks changes to Defender. In Windows Security → "Virus & threat protection" → "Manage settings" → turn off "Tamper Protection".',
        'tamper.open':          'Open the page for me →',

        // Aside
        'aside.profile':         'Profile',
        'aside.sections':        'Sections',
        'aside.actions':         'Actions',
        'aside.history':         'History',
        'aside.tout':            'All',
        'aside.aucun':           'None',
        'aside.advanced':        'Advanced options',
        'aside.audit_label':     'Test mode (checks only, no modification)',

        // Status bar
        'status.placeholder':    'Select sections then click Check or Apply.',

        // Filters
        'filter.level':          'Level',
        'filter.state':          'State',
        'filter.critical':       'Critical',
        'filter.important':      'Important',
        'filter.nice':           'Optional',
        'filter.pending':        'Pending',
        'filter.compliant':      'Compliant',
        'filter.toapply':        'To harden',
        'filter.failed':         'Failed',
        'filter.reset':          'Reset',

        // Table
        'table.severity':        'Level',
        'table.rule':            'Rule',
        'table.state':           'State',
        'table.action':          'Proposed action',
        'table.empty':           'No results yet. Click "Check" to see the current state.',

        // Loader
        'loader.analyzing':      'Analyzing…',
        'loader.applying':       'Applying…',

        // Modal confirm
        'modal.confirm_title':   '⚠ Apply changes to your PC?',
        'modal.confirm_text_pre':'These actions will',
        'modal.confirm_text_strong':'modify the system',
        'modal.confirm_text_post':'Everything is recorded in a journal — you can undo each change afterwards.',
        'modal.affected':        'Affected sections:',
        'modal.confirm_btn':     'Confirm and apply',

        // Watchlist banner
        'watchlist.title':       'Alerts detected after a recent improvement.',
        'watchlist.detail':      'See details',

        // Drift banner (post-Windows Update)
        'drift.title':           'Windows Update reset some of your security improvements.',
        'drift.reapply':         'Re-apply',

        // Generic errors and status messages
        'error.generic':          'Error: {msg}',
        'error.selectSections':   'Select at least one section.',
        'error.noRulesInSelection':'No rules in selection.',
        'error.adminRelaunch':    'Relaunch failed: {msg}',
        'error.undoFailed':       'Undo failed: {msg}',
        'status.loading':         'Loading…',
        'status.loadingRun':      'Loading session {id}…',
        'status.profileChanged':  'Profile changed. Run a check to see the current state.',
        'status.runStart':        'Session {id} ({mode}, {sec} section(s), {n} rule(s))',
        'status.undoing':         'Undoing {id}…',
        'status.undoOk':          'Undo OK for {id}',
        'status.creatingRP':      'Creating a Windows Restore Point (may take 30-60s)…',
        'status.rpCreated':       'Restore Point created in {sec}s. Starting apply…',
        'status.rpSkipped':       'Restore Point not created {why}— apply continues (undo via journal still available).',
        'status.applyContinues':  'Click "Apply" to re-harden the {n} drifted item(s).',

        // Loader counters
        'loader.progress':        '{done} / {total} rule(s)',

        // Dashboard
        'dashboard.detailCount':  '{ok}/{n} already OK',
        'dashboard.rollbackAlert':'{n} rule(s) triggered an automatic revert. Check the details below.',
        'dashboard.failedAlert':  '{n} rule(s) failed — see details below.',

        // Suggestion banner
        'suggestion.label':       '💡 Suggested:',
        'suggestion.autoSkipped': '{n} rule(s) auto-unchecked:',

        // Table footer
        'table.count':            '{n} rule(s)',
        'table.countFiltered':    '{visible} / {total} rule(s) shown',

        // Watchlist detail
        'watchlist.summary':      '{n} event(s) across {s} source(s) — click to see details.',
        'watchlist.modalTitle':   'Event Viewer anomalies post-apply',
        'watchlist.modalIntro':   'Detected by the watchlist 24h after your recent applies. If a source spikes, a Win11 Hardening rule may be breaking something you use.',
        'watchlist.thRun':        'Session',
        'watchlist.thSource':     'Source',
        'watchlist.thEvents':     'Events',
        'watchlist.thReason':     'Why it looks suspicious',
        'watchlist.modalHelp':    'For deeper analysis: <code>Get-WinEvent -LogName \'&lt;source&gt;\' -MaxEvents 50</code> shows the raw events. You can also run <code>harden-engine snapshot diff &lt;runID&gt;</code> to see what was modified during the session.',

        // Score modal
        'score.title':            '📊 Maturity score',
        'score.thComponent':      'Component',
        'score.thPoints':         'Points',
        'score.thLevel':          'Level',
        'score.thDetail':         'Detail',
        'score.weighting':        'Weighting: critical 50, important 25, optional 10, Restore Point 8, watchlist 7. Max total: 100.',

        // Tooltips
        'tooltip.notdefined':     'not set',
        'tooltip.rebootRequired': 'Reboot required after applying',
        'tooltip.irreversible':   'Irreversible',
        'tooltip.cantUndo':       'This rule cannot be undone.',
        'tooltip.includeRow':     'Uncheck to exclude this rule',
        'tooltip.undoRun':        'Undo this session (revert applied rules)',
        'tooltip.confirmUndo':    'Undo session {id}? This will revert all rules applied during this session.',

        // Admin dynamic buttons
        'admin.relaunching':      'Relaunching…',
        'admin.applyDisabled':    'Relaunch the GUI as admin to enable',

        // Drift summary
        'drift.summary':          '{n} item(s) drifted since last baseline ({base}).',

        // HTML title attributes
        'title.dryrun':           'Reads system state without modifying anything (no admin needed)',
        'title.apply':            'Actually applies changes (admin required)',
        'title.undo':             'Undo changes from the last session',
        'title.auditMode':        'For Microsoft Defender protections that can produce false positives (ASR, Network Protection): turn on test mode for 1-2 weeks to see what would be blocked in Event Viewer (events 1121/1122) without breaking your apps. Once you\'re sure no legit app is impacted, uncheck to enable real blocking.',
        'title.score':            'View your hardening maturity score (grade A/B/C/D)',
        'title.filterReset':      'Reset filters',
        'title.colInclude':       'Uncheck to exclude this rule from the next session',
        'title.includeAll':       'Check / uncheck all',
    },
};

// Détection au boot. Default = FR (pas EN) car le projet est français à
// l'origine ; la version EN est une traduction. Si l'utilisateur a explicitement
// switché via le bouton, on respecte son choix (localStorage).
let currentLang = (function() {
    try {
        const saved = localStorage.getItem('harden-lang');
        if (saved === 'fr' || saved === 'en') return saved;
    } catch {}
    const nav = (navigator.language || '').toLowerCase();
    // Seul navigator.language commençant par 'en' bascule en EN.
    // Tout le reste (fr, par défaut, undefined) → FR.
    return nav.startsWith('en') ? 'en' : 'fr';
})();

function t(key, params) {
    const dict = I18N[currentLang] || I18N.en;
    let str = dict[key] !== undefined ? dict[key] : (I18N.en[key] !== undefined ? I18N.en[key] : key);
    if (params) {
        for (const [k, v] of Object.entries(params)) {
            str = str.replaceAll('{' + k + '}', String(v));
        }
    }
    return str;
}

function setLang(lang) {
    if (lang !== 'fr' && lang !== 'en') return;
    currentLang = lang;
    try { localStorage.setItem('harden-lang', lang); } catch {}
}

function getLang() { return currentLang; }
