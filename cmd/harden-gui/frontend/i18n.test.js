// Tests Node.js isolés pour i18n.js. Pas de browser, pas de Wails — juste la
// logique pure. Lancer : node cmd/harden-gui/frontend/i18n.test.js
//
// On simule navigator + localStorage avec des stubs minimaux.

let _ls = {};
const localStorage = {
    getItem: k => _ls[k] === undefined ? null : _ls[k],
    setItem: (k, v) => { _ls[k] = String(v); },
    removeItem: k => { delete _ls[k]; },
};

let _navigatorLang = 'fr-FR';
const navigator = { get language() { return _navigatorLang; } };

global.localStorage = localStorage;
global.navigator = navigator;

// Charge i18n.js comme un module CommonJS hacké (le fichier est browser-style).
const fs = require('fs');
const path = require('path');
const code = fs.readFileSync(path.join(__dirname, 'i18n.js'), 'utf8');
// On élide la déclaration globale de `const I18N` pour pouvoir réinjecter
// avec eval. Plus simple : on wrap dans une factory qui retourne les fns.
function loadI18n() {
    const factory = new Function('navigator', 'localStorage',
        code + '\nreturn { t, setLang, getLang, I18N };');
    return factory(navigator, localStorage);
}

let pass = 0, fail = 0;
function test(name, fn) {
    try {
        fn();
        console.log('✓', name);
        pass++;
    } catch (e) {
        console.log('✗', name, '—', e.message);
        fail++;
    }
}
function assertEq(a, b, msg) {
    if (a !== b) throw new Error((msg || 'mismatch') + `: expected "${b}", got "${a}"`);
}

// 1. Default fr quand navigator.language = fr-FR
_ls = {}; _navigatorLang = 'fr-FR';
{
    const i = loadI18n();
    test('default lang = fr when navigator=fr-FR', () => {
        assertEq(i.getLang(), 'fr');
    });
    test('t() returns french text', () => {
        assertEq(i.t('verb.disable'), 'Désactiver');
    });
}

// 2. Default fr aussi quand navigator.language est vide
_ls = {}; _navigatorLang = '';
{
    const i = loadI18n();
    test('default lang = fr when navigator=empty', () => {
        assertEq(i.getLang(), 'fr');
    });
}

// 3. Default en quand navigator.language = en-US
_ls = {}; _navigatorLang = 'en-US';
{
    const i = loadI18n();
    test('default lang = en when navigator=en-US', () => {
        assertEq(i.getLang(), 'en');
    });
    test('t() returns english text', () => {
        assertEq(i.t('verb.disable'), 'Disable');
    });
}

// 4. setLang persiste dans localStorage et getLang reflète
_ls = {}; _navigatorLang = 'fr-FR';
{
    const i = loadI18n();
    test('setLang en updates getLang and localStorage', () => {
        i.setLang('en');
        assertEq(i.getLang(), 'en');
        assertEq(localStorage.getItem('harden-lang'), 'en');
    });
    test('after setLang en, t() returns english', () => {
        assertEq(i.t('verb.disable'), 'Disable');
    });
}

// 5. localStorage prime sur navigator.language
_ls = { 'harden-lang': 'en' }; _navigatorLang = 'fr-FR';
{
    const i = loadI18n();
    test('localStorage saved en overrides navigator fr-FR', () => {
        assertEq(i.getLang(), 'en');
    });
}

// 6. localStorage saved fr fait revenir en français
_ls = { 'harden-lang': 'fr' }; _navigatorLang = 'en-US';
{
    const i = loadI18n();
    test('localStorage saved fr overrides navigator en-US', () => {
        assertEq(i.getLang(), 'fr');
    });
    test('t verb.activate returns french', () => {
        assertEq(i.t('verb.activate'), 'Activer la protection');
    });
}

// 7. setLang invalide est ignoré
_ls = {}; _navigatorLang = 'fr-FR';
{
    const i = loadI18n();
    test('setLang("garbage") is ignored', () => {
        i.setLang('garbage');
        assertEq(i.getLang(), 'fr');
    });
}

// 8. t() avec params remplace les placeholders
_ls = {}; _navigatorLang = 'fr-FR';
{
    const i = loadI18n();
    test('t() with params replaces {n}', () => {
        assertEq(i.t('dashboard.improveOpt', { n: 5, s: 's' }),
                 'Ta machine est solide. 5 améliorations optionnelles possibles si tu veux pousser plus loin.');
    });
}

// 9. t() avec key inconnue retourne la key
_ls = {}; _navigatorLang = 'fr-FR';
{
    const i = loadI18n();
    test('t() unknown key returns the key itself', () => {
        assertEq(i.t('does.not.exist'), 'does.not.exist');
    });
}

// 10. t() en mode en avec key seulement présente en fr fallback to en (non, en fr→en : pas attendu)
_ls = { 'harden-lang': 'en' }; _navigatorLang = 'en-US';
{
    const i = loadI18n();
    test('all i18n keys exist in both fr and en', () => {
        const frKeys = Object.keys(i.I18N.fr);
        const enKeys = Object.keys(i.I18N.en);
        const missingInEn = frKeys.filter(k => !(k in i.I18N.en));
        const missingInFr = enKeys.filter(k => !(k in i.I18N.fr));
        if (missingInEn.length || missingInFr.length) {
            throw new Error(`missing in en: [${missingInEn}] missing in fr: [${missingInFr}]`);
        }
    });
}

console.log('\n----');
console.log(`Pass: ${pass}  Fail: ${fail}`);
process.exit(fail > 0 ? 1 : 0);
