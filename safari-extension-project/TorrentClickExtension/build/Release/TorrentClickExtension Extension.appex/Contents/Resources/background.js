// TorrentClick Safari Extension - Background Service Worker

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'torrentclick-send-link',
    title: 'Send to TorrentClick (in-memory)',
    contexts: ['link']
  });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId === 'torrentclick-send-link' && info.linkUrl) {
    const torrentUrl = info.linkUrl;
    try {
      const response = await fetch(torrentUrl, {
        credentials: 'include',
        headers: { 'Accept': 'application/x-bittorrent, */*' }
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const buffer = await response.arrayBuffer();
      const bytes = new Uint8Array(buffer);
      let binary = '';
      const len = bytes.byteLength;
      const chunkSize = 8192;
      for (let i = 0; i < len; i += chunkSize) {
        const chunk = bytes.subarray(i, Math.min(i + chunkSize, len));
        binary += String.fromCharCode.apply(null, chunk);
      }
      const b64 = btoa(binary);

      let filename = 'download.torrent';
      try {
        const u = new URL(torrentUrl);
        filename = u.pathname.split('/').pop() || 'download.torrent';
      } catch (_) {}

      const schemeUrl = `torrentclick://add?name=${encodeURIComponent(filename)}&data=${encodeURIComponent(b64)}&url=${encodeURIComponent(torrentUrl)}`;

      if (tab && tab.id) {
        chrome.tabs.update(tab.id, { url: schemeUrl });
      }
    } catch (err) {
      console.error('[TorrentClick] Context menu dispatch failed:', err);
    }
  }
});
