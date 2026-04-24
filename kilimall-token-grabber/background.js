console.log('[Kilimall Grabber] Service Worker 已启动，正在监听 *.kilimall.ke 流量...');

chrome.webRequest.onSendHeaders.addListener(
  (details) => {
    console.log('[Kilimall Grabber] Request detected:', details.url);

    if (!details.requestHeaders) {
      console.log('[Kilimall Grabber] No headers found');
      return;
    }

    let accessToken = '';
    let cookieHeader = '';

    for (const header of details.requestHeaders) {
      if (header.name.toLowerCase() === 'accesstoken') {
        accessToken = header.value || '';
      }
      if (header.name.toLowerCase() === 'cookie') {
        cookieHeader = header.value || '';
      }
    }

    console.log('[Kilimall Grabber] Found accessToken:', !!accessToken, 'cookie:', !!cookieHeader);

    if (!accessToken && !cookieHeader) return;

    let sellerSid = '';
    if (cookieHeader) {
      const match = cookieHeader.match(/seller-sid=([^;]+)/);
      if (match) sellerSid = match[1];
    }

    const currentToken = accessToken.trim();
    const currentCookie = sellerSid.trim();

    if (!currentToken || !currentCookie) return;

    chrome.storage.session.get(['lastToken', 'lastCookie'], (stored) => {
      if (currentToken === stored.lastToken && currentCookie === stored.lastCookie) return;

      chrome.storage.session.set({ lastToken: currentToken, lastCookie: currentCookie });

      fetch('http://127.0.0.1:8080/api/system/kilimall-cookie', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          token: currentToken,
          cookie: `seller-sid=${currentCookie}`
        })
      }).catch(err => console.error('Failed to report token:', err));
    });
  },
  { urls: ['*://*.kilimall.ke/*'] },
  ['requestHeaders']
);
