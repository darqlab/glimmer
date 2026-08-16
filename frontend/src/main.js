import './style.css';

import { OpenFile, GetInitialFile } from '../wailsjs/go/main/App';

const openBtn = document.getElementById('open-btn');
const themeBtn = document.getElementById('theme-btn');
const pathLabel = document.getElementById('path-label');
const content = document.getElementById('content');
const emptyState = document.getElementById('empty-state');

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
