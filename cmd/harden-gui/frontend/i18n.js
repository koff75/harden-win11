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
        'status.tofail':          'Échec test',
        'status.applied':         'Appliquée ✓',
        'status.failed':          'Échec',
        'status.rolledback':      'Rollback exécuté',
        'status.bloat.notinst':   'Pas installée',
        'status.bloat.toremove':  'À supprimer',
        'status.bloat.uninstalled':'Désinstallée ✓',

        // Action cell
        'cell.notyetchecked':   'Pas encore vérifiée',
        'cell.checkimpossible': 'Vérification impossible',
        'cell.actionfailed':    'Action a échoué → rollback exécuté',
        'cell.compliant':       'Aucune action — déjà conforme',
        'cell.notinstalled':    'Pas installée — rien à faire',
        'cell.appliedsuccess':  'Appliquée avec succès',
        'cell.uninstalled':     'Désinstallée ✓',
        'cell.canbreakprefix':  '⚠ peut casser…',
        'cell.hoverhint':       'Survole pour voir le détail',

        // Dashboard
        'dashboard.allok':      'Système conforme — toutes les règles évaluées sont OK',
        'dashboard.improveImp': 'Ta machine est bien protégée. {n} amélioration{s} importante{s}{opt} possible{ps}.',
        'dashboard.improveOpt': 'Ta machine est solide. {n} amélioration{s} optionnelle{s} possible{s} si tu veux pousser plus loin.',
        'dashboard.improveCrit':'Ta machine est OK sur {ok} points. {n} amélioration{s} possible{s} : {crit} critique{cs}{rest}.',

        // Header / boutons
        'btn.dryrun':  'Vérifier',
        'btn.apply':   'Appliquer',
        'btn.cancel':  'Annuler',
        'btn.undo':    'Annuler le dernier run',
        'btn.score':   '📊 Score',
        'btn.langtoggle': 'EN',

        // Misc
        'admin.notadmin':       "Tu n'es pas administrateur.",
        'admin.banner':         'Tu peux explorer et faire des dry-runs, mais Appliquer et Annuler sont désactivés.',
        'admin.relaunch':       'Relancer en admin',
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
        'status.tofail':          'Test failed',
        'status.applied':         'Applied ✓',
        'status.failed':          'Failed',
        'status.rolledback':      'Rolled back',
        'status.bloat.notinst':   'Not installed',
        'status.bloat.toremove':  'To remove',
        'status.bloat.uninstalled':'Uninstalled ✓',

        'cell.notyetchecked':   'Not yet checked',
        'cell.checkimpossible': 'Check impossible',
        'cell.actionfailed':    'Action failed → rollback executed',
        'cell.compliant':       'No action — already compliant',
        'cell.notinstalled':    'Not installed — nothing to do',
        'cell.appliedsuccess':  'Applied successfully',
        'cell.uninstalled':     'Uninstalled ✓',
        'cell.canbreakprefix':  '⚠ may break…',
        'cell.hoverhint':       'Hover for details',

        'dashboard.allok':      'System compliant — all evaluated rules are OK',
        'dashboard.improveImp': 'Your machine is well protected. {n} important improvement{s}{opt} possible.',
        'dashboard.improveOpt': 'Your machine is solid. {n} optional improvement{s} possible if you want to push further.',
        'dashboard.improveCrit':'Your machine is OK on {ok} points. {n} improvement{s} possible: {crit} critical{rest}.',

        'btn.dryrun':  'Check',
        'btn.apply':   'Apply',
        'btn.cancel':  'Cancel',
        'btn.undo':    'Undo last run',
        'btn.score':   '📊 Score',
        'btn.langtoggle': 'FR',

        'admin.notadmin':       "You are not administrator.",
        'admin.banner':         'You can explore and run dry-runs, but Apply and Undo are disabled.',
        'admin.relaunch':       'Relaunch as admin',
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
