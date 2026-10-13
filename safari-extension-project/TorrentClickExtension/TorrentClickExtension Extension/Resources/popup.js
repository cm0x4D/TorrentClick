document.addEventListener('DOMContentLoaded', () => {
  const toggle = document.getElementById('autoInterceptToggle');
  const testBtn = document.getElementById('testBtn');

  // Load saved preference
  if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) {
    chrome.storage.local.get({ autoIntercept: true }, (items) => {
      toggle.checked = items.autoIntercept;
    });

    toggle.addEventListener('change', () => {
      chrome.storage.local.set({ autoIntercept: toggle.checked });
    });
  }

  // Test scheme button
  testBtn.addEventListener('click', () => {
    // Valid minimal bencode payload
    const testB64 = 'ZDg6YW5ub3VuY2UyNzpodHRwOi8vdHJhY2tlci5leGFtcGxlLmNvbS80OmluZm9kNjpsZW5ndGhpMTA0ODU3NmU0Om5hbWUxMjp0ZXN0ZmlsZS5pc28xMjpwaWVjZSBsZW5ndGhpMTYzODRlNjpwaWVjZXMyMDoxMjM0NTY3ODkwMTIzNDU2Nzg5MGVl';
    const testUrl = `torrentclick://add?name=SafariExtensionTest.torrent&data=${encodeURIComponent(testB64)}&url=safari-extension-test`;

    if (typeof chrome !== 'undefined' && chrome.tabs) {
      chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
        if (tabs && tabs[0]) {
          chrome.tabs.update(tabs[0].id, { url: testUrl });
        } else {
          window.location.href = testUrl;
        }
      });
    } else {
      window.location.href = testUrl;
    }
  });
});
