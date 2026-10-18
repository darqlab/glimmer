import './style.css';

import { OpenFile, GetInitialFile, GetHighlightCSS } from '../wailsjs/go/main/App';
import { EventsOn, ClipboardSetText } from '../wailsjs/runtime/runtime';

const openBtn = document.getElementById('open-btn');
const themeBtn = document.getElementById('theme-btn');
const linesBtn = document.getElementById('lines-btn');
const copyBtn = document.getElementById('copy-btn');
const pathLabel = document.getElementById('path-label');
const pathText = pathLabel.querySelector('bdi');
const modeLabel = document.getElementById('mode-label');
const modeLabelText = document.getElementById('mode-label-text');
const notice = document.getElementById('notice');
const noticeText = document.getElementById('notice-text');
const noticeDismiss = document.getElementById('notice-dismiss');
const content = document.getElementById('content');
const contentWrap = document.getElementById('content-wrap');
const emptyState = document.getElementById('empty-state');
const emptyText = document.getElementById('empty-text');
const EMPTY_TEXT = emptyText.textContent;
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

// True once a document is on screen. A refusal then goes in the notice bar
// and leaves the document alone; with no document it fills the content area.
let hasDocument = false;

function hideNotice() {
    notice.hidden = true;
    notice.classList.remove('neutral');
    notice.setAttribute('role', 'alert');
}

function showNotice(message) {
    clearOpening();
    noticeText.textContent = '';
    const lead = /^(Can't show [^:]*:)\s*(.*)$/s.exec(message);
    if (lead) {
        const strong = document.createElement('strong');
        strong.textContent = lead[1];
        noticeText.append(strong, ' ' + lead[2]);
    } else {
        noticeText.textContent = message;
    }
    notice.classList.remove('neutral');
    notice.setAttribute('role', 'alert');
    notice.hidden = false;
    noticeDismiss.focus();
}

noticeDismiss.addEventListener('click', () => {
    hideNotice();
    openBtn.focus();
});

document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !notice.hidden && !notice.classList.contains('neutral')) {
        hideNotice();
        openBtn.focus();
    }
});

// Slow-load cue (DEC-S10): Go emits file:opening just before it reads the
// file; if no result has arrived 500 ms later, show a static "Opening <name>…".
let openingTimer = null;
let openingShown = false;

function startOpening(name) {
    clearOpening();
    openingTimer = setTimeout(() => {
        openingTimer = null;
        openingShown = true;
        const text = `Opening ${name}\u2026`;
        if (hasDocument) {
            noticeText.textContent = text;
            notice.classList.add('neutral');
            notice.setAttribute('role', 'status');
            notice.hidden = false;
        } else {
            emptyText.textContent = text;
        }
    }, 500);
}

function clearOpening() {
    if (openingTimer) {
        clearTimeout(openingTimer);
        openingTimer = null;
    }
    if (openingShown) {
        openingShown = false;
        emptyText.textContent = EMPTY_TEXT;
        if (notice.classList.contains('neutral')) {
            hideNotice();
        }
    }
}

EventsOn('file:opening', startOpening);

function modeTooltip(result) {
    if (result.mode === 'source') {
        return `${result.language} source, shown exactly as on disk`;
    }
    if (result.mode === 'text') {
        return 'Plain text, wrapped';
    }
    return 'Rendered markdown';
}

function applyResult(result) {
    clearOpening();
    hideNotice();
    const mode = result.mode || 'markdown';
    content.className = result.everyLine ? 'source-body' : 'markdown-body';
    content.dataset.mode = mode;
    contentWrap.dataset.mode = mode;
    content.innerHTML = result.html;
    emptyState.style.display = 'none';
    modeLabelText.textContent = result.language || '';
    modeLabel.title = modeTooltip(result);
    modeLabel.hidden = false;
    pathText.textContent = result.path || '';
    pathLabel.title = result.path || '';
    document.title = result.name ? `glimmer \u2014 ${result.name}` : 'glimmer';
    hasDocument = true;
}

function showResult(result) {
    if (!result || !result.html) {
        return;
    }
    if (openingShown) {
        // Let the cue paint before the (possibly large) insert blocks the thread.
        requestAnimationFrame(() => requestAnimationFrame(() => applyResult(result)));
        return;
    }
    applyResult(result);
}

function showError(err) {
    console.error(err);
    const message = (err && err.message) ? err.message : String(err);
    if (hasDocument) {
        showNotice(message);
        return;
    }
    clearOpening();
    content.className = 'message-body';
    content.dataset.mode = '';
    contentWrap.dataset.mode = '';
    content.textContent = '';
    const p = document.createElement('p');
    p.textContent = message;
    content.appendChild(p);
    emptyState.style.display = 'none';
    modeLabel.hidden = true;
}

// A load that produced no HTML (cancelled dialog, empty markdown file) must
// still end the opening cue.
function settle(promise) {
    promise
        .then((result) => {
            if (result && result.html) {
                showResult(result);
            } else {
                clearOpening();
            }
        })
        .catch(showError);
}

openBtn.addEventListener('click', () => {
    settle(OpenFile());
});

// The highlight stylesheet must be in the page before the first render, so
// the initial file is requested only after it has been injected.
GetHighlightCSS()
    .then((css) => {
        const style = document.createElement('style');
        style.id = 'highlight-css';
        style.textContent = css;
        document.head.appendChild(style);
    })
    .catch((err) => console.error(err))
    .then(() => settle(GetInitialFile()));

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
    const prevScrollLeft = contentWrap.scrollLeft;
    showResult(result);
    const maxScrollTop = Math.max(0, contentWrap.scrollHeight - contentWrap.clientHeight);
    const maxScrollLeft = Math.max(0, contentWrap.scrollWidth - contentWrap.clientWidth);
    contentWrap.scrollTop = Math.min(prevScrollTop, maxScrollTop);
    contentWrap.scrollLeft = Math.min(prevScrollLeft, maxScrollLeft);
});
