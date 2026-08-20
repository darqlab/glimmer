import './style.css';

import { OpenFile, GetInitialFile } from '../wailsjs/go/main/App';
import { EventsOn, ClipboardSetText } from '../wailsjs/runtime/runtime';

const openBtn = document.getElementById('open-btn');
const themeBtn = document.getElementById('theme-btn');
const linesBtn = document.getElementById('lines-btn');
const copyBtn = document.getElementById('copy-btn');
const pathLabel = document.getElementById('path-label');
const content = document.getElementById('content');
const contentWrap = document.getElementById('content-wrap');
const emptyState = document.getElementById('empty-state');
const toast = document.getElementById('toast');

const THEME_ICONS = { light: '☀️', dark: '🌙' };

function systemPrefersDark() {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function applyTheme(theme) {
    if (theme) {
        document.documentElement.setAttribute('data-theme', theme);
    } else {
        document.documentElement.removeAttribute('data-theme');
    }
    const effective = theme || (systemPrefersDark() ? 'dark' : 'light');
    themeBtn.textContent = THEME_ICONS[effective] || '🌓';
}

function initTheme() {
    const saved = localStorage.getItem('glimmer:theme');
    applyTheme(saved || null);
}

themeBtn.addEventListener('click', () => {
    const current = document.documentElement.getAttribute('data-theme')
        || (systemPrefersDark() ? 'dark' : 'light');
    const next = current === 'dark' ? 'light' : 'dark';
    localStorage.setItem('glimmer:theme', next);
    applyTheme(next);
});

initTheme();

// Source line numbers: purely a data attribute on the content container —
// the gutter itself is drawn by CSS from each block's data-line. Off by
// default (it changes the page's look); persisted like the theme toggle.
function applyLines(on) {
    content.dataset.lines = on ? 'on' : '';
    linesBtn.setAttribute('aria-pressed', on ? 'true' : 'false');
}

function initLines() {
    const saved = localStorage.getItem('glimmer:lines');
    applyLines(saved === 'on');
}

linesBtn.addEventListener('click', () => {
    const next = content.dataset.lines !== 'on';
    localStorage.setItem('glimmer:lines', next ? 'on' : 'off');
    applyLines(next);
});

initLines();

// Copy-on-selection: glimmer is read-only, so selecting text has exactly
// one purpose — copying it. On by default; the toggle is the escape hatch.
let autoCopyEnabled = true;

function applyAutoCopy(on) {
    autoCopyEnabled = on;
    copyBtn.setAttribute('aria-pressed', on ? 'true' : 'false');
    copyBtn.style.opacity = on ? '1' : '0.5';
}

function initAutoCopy() {
    const saved = localStorage.getItem('glimmer:autocopy');
    applyAutoCopy(saved === null ? true : saved === 'on');
}

copyBtn.addEventListener('click', () => {
    const next = !autoCopyEnabled;
    localStorage.setItem('glimmer:autocopy', next ? 'on' : 'off');
    applyAutoCopy(next);
});

initAutoCopy();

let toastTimer = null;

function showToast(msg) {
    toast.textContent = msg;
    toast.classList.add('show');
    if (toastTimer) {
        clearTimeout(toastTimer);
    }
    toastTimer = setTimeout(() => {
        toast.classList.remove('show');
        toastTimer = null;
    }, 1600);
}

function copyText(text) {
    if (typeof ClipboardSetText === 'function') {
        return ClipboardSetText(text).then(() => undefined);
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
        return navigator.clipboard.writeText(text);
    }
    return Promise.reject(new Error('no clipboard API available'));
}

document.addEventListener('mouseup', () => {
    if (!autoCopyEnabled) {
        return;
    }
    // Deferred a tick: on mouseup the selection is not yet finalized.
    setTimeout(() => {
        const selection = window.getSelection();
        if (!selection) {
            return;
        }
        const text = selection.toString();
        if (!text.trim()) {
            return; // plain click, or whitespace-only selection
        }
        if (!contentWrap.contains(selection.anchorNode)) {
            return; // selection outside the reading area (toolbar, etc.)
        }
        copyText(text)
            .then(() => showToast('Copied to clipboard'))
            .catch(() => {});
    }, 0);
});

function showResult(result) {
    if (!result || !result.html) {
        return;
    }
    content.innerHTML = result.html;
    emptyState.style.display = 'none';
    pathLabel.textContent = result.path || '';
    pathLabel.title = result.path || '';
    document.title = result.name ? `glimmer — ${result.name}` : 'glimmer';
}

function showError(err) {
    console.error(err);
    const message = (err && err.message) ? err.message : String(err);
    content.innerHTML = `<p style="color:#c0392b"><strong>Error:</strong> ${escapeHtml(message)}</p>`;
    emptyState.style.display = 'none';
}

function escapeHtml(s) {
    const div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
}

openBtn.addEventListener('click', () => {
    OpenFile()
        .then((result) => {
            if (result && result.html) {
                showResult(result);
            }
        })
        .catch(showError);
});

GetInitialFile()
    .then((result) => {
        if (result && result.html) {
            showResult(result);
        }
    })
    .catch(showError);

// Auto-reload: the backend watches the open file and pushes a re-render on
// external change. Apply silently — no banner, no confirm click (DEC-A1) —
// and preserve the reader's scroll position across the swap. The offset is
// restored as an absolute value, not proportionally, because a proportional
// restore would move the reader's line whenever the document length changes,
// which is exactly what an edit does.
EventsOn('file:changed', (result) => {
    if (!result || !result.html) {
        return;
    }
    const prevScrollTop = contentWrap.scrollTop;
    showResult(result);
    const maxScrollTop = Math.max(0, contentWrap.scrollHeight - contentWrap.clientHeight);
    contentWrap.scrollTop = Math.min(prevScrollTop, maxScrollTop);
});
