// ==UserScript==
// @name         TorrentClick In-Memory Interceptor
// @namespace    https://github.com/torrentclick
// @version      1.0.0
// @description  Intercepts .torrent downloads in Safari/browsers and forwards them directly to TorrentClick in memory without downloading to disk.
// @author       TorrentClick
// @match        *://*/*
// @run-at       document-start
// @grant        none
// ==/UserScript==

(function () {
  'use strict';

  function isTorrentLink(anchor) {
    if (!anchor || !anchor.href) return false;
    const href = anchor.href.trim();
    if (!href || href.startsWith('javascript:') || href.startsWith('magnet:')) return false;

    try {
      const parsed = new URL(href, window.location.href);
      if (parsed.pathname.toLowerCase().endsWith('.torrent')) return true;
      if (parsed.search.toLowerCase().includes('.torrent')) return true;
      if (parsed.pathname.toLowerCase().includes('/torrent') && parsed.search.toLowerCase().includes('download')) return true;
      if (parsed.search.toLowerCase().includes('action=download')) return true;
    } catch (_) {
      if (href.toLowerCase().includes('.torrent')) return true;
    }

    const downloadAttr = anchor.getAttribute('download');
    if (downloadAttr && downloadAttr.toLowerCase().endsWith('.torrent')) {
      return true;
    }

    return false;
  }

  function bufferToBase64(buffer) {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    const len = bytes.byteLength;
    const chunkSize = 8192;
    for (let i = 0; i < len; i += chunkSize) {
      const chunk = bytes.subarray(i, Math.min(i + chunkSize, len));
      binary += String.fromCharCode.apply(null, chunk);
    }
    return btoa(binary);
  }

  function showToast(message, isSuccess, isError) {
    let toast = document.getElementById('torrentclick-toast');
    if (!toast) {
      toast = document.createElement('div');
      toast.id = 'torrentclick-toast';
      Object.assign(toast.style, {
        position: 'fixed',
        bottom: '24px',
        right: '24px',
        zIndex: '2147483647',
        backgroundColor: 'rgba(28, 28, 30, 0.95)',
        color: '#ffffff',
        backdropFilter: 'blur(10px)',
        webkitBackdropFilter: 'blur(10px)',
        padding: '12px 18px',
        borderRadius: '10px',
        fontFamily: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", Roboto, sans-serif',
        fontSize: '13px',
        fontWeight: '500',
        lineHeight: '1.4',
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.35), 0 0 0 1px rgba(255, 255, 255, 0.1)',
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        opacity: '0',
        transform: 'translateY(12px)',
        transition: 'all 0.25s cubic-bezier(0.16, 1, 0.3, 1)',
        pointerEvents: 'none',
      });
      document.body.appendChild(toast);
    }

    const icon = isError ? '⚠️' : isSuccess ? '✅' : '⚡';
    toast.innerHTML = `<span style="font-size:16px;">${icon}</span><span>${message}</span>`;
    toast.style.opacity = '1';
    toast.style.transform = 'translateY(0)';

    clearTimeout(toast._hideTimer);
    toast._hideTimer = setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateY(12px)';
    }, 2800);
  }

  document.addEventListener(
    'click',
    async (event) => {
      // Hold Option/Alt to bypass
      if (event.altKey) return;

      const anchor = event.target.closest('a');
      if (!anchor || !isTorrentLink(anchor)) return;

      event.preventDefault();
      event.stopPropagation();

      const torrentUrl = anchor.href;
      let filename = anchor.getAttribute('download') || '';
      if (!filename) {
        try {
          const u = new URL(torrentUrl);
          filename = u.pathname.split('/').pop() || 'download.torrent';
        } catch (_) {
          filename = 'download.torrent';
        }
      }
      if (!filename.toLowerCase().endsWith('.torrent')) {
        filename += '.torrent';
      }

      showToast(`Intercepting ${filename} in memory...`, false, false);

      try {
        const response = await fetch(torrentUrl, {
          credentials: 'include',
          headers: { 'Accept': 'application/x-bittorrent, */*' }
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const buffer = await response.arrayBuffer();
        if (!buffer || buffer.byteLength === 0) {
          throw new Error('Empty response body');
        }

        const base64Data = bufferToBase64(buffer);
        const targetSchemeUrl = `torrentclick://add?name=${encodeURIComponent(filename)}&data=${encodeURIComponent(base64Data)}&url=${encodeURIComponent(torrentUrl)}`;

        showToast(`Forwarded to TorrentClick!`, true, false);
        window.location.href = targetSchemeUrl;
      } catch (err) {
        console.warn('[TorrentClick] In-memory interception failed:', err);
        showToast(`Interception failed: ${err.message}`, false, true);
        setTimeout(() => {
          if (confirm(`TorrentClick in-memory interception encountered an error (${err.message}). Download normally?`)) {
            window.location.href = torrentUrl;
          }
        }, 300);
      }
    },
    true
  );

  console.log('[TorrentClick] Userscript in-memory interceptor loaded.');
})();
