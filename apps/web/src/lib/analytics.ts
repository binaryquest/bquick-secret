const measurementId = 'G-QT7NS9WP6D';

declare global {
  interface Window {
    dataLayer?: unknown[];
    gtag?: (...args: unknown[]) => void;
  }
}

export function initAnalytics() {
  loadGoogleTag();

  window.dataLayer = window.dataLayer || [];
  window.gtag = function gtag(...args: unknown[]) {
    window.dataLayer?.push(args);
  };

  window.gtag('js', new Date());
  window.gtag('config', measurementId, {
    allow_ad_personalization_signals: false,
    allow_google_signals: false,
    client_storage: 'none',
    send_page_view: false
  });
  sendPageView();
}

function loadGoogleTag() {
  if (document.getElementById('google-tag-script')) {
    return;
  }

  const script = document.createElement('script');
  script.id = 'google-tag-script';
  script.async = true;
  script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(measurementId)}`;
  document.head.appendChild(script);
}

function sendPageView() {
  const pagePath = sanitizedPath(window.location.pathname);
  window.gtag?.('event', 'page_view', {
    page_location: `${window.location.origin}${pagePath}`,
    page_path: pagePath,
    page_title: document.title
  });
}

function sanitizedPath(pathname: string) {
  if (pathname.startsWith('/s/')) {
    return '/s/[secret]';
  }
  return pathname || '/';
}
