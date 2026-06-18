const siteKey = import.meta.env.VITE_RECAPTCHA_SITE_KEY || '';
const scriptId = 'recaptcha-enterprise-script';

declare global {
  interface Window {
    grecaptcha?: {
      enterprise?: {
        execute(siteKey: string, options: { action: string }): Promise<string>;
        ready(callback: () => void): void;
      };
    };
  }
}

export function isRecaptchaEnabled() {
  return Boolean(siteKey);
}

export async function getRecaptchaToken(action: string): Promise<string | undefined> {
  if (!siteKey) {
    return undefined;
  }
  await loadRecaptchaScript();
  await waitForReady();
  return window.grecaptcha?.enterprise?.execute(siteKey, { action });
}

function loadRecaptchaScript() {
  if (document.getElementById(scriptId)) {
    return Promise.resolve();
  }
  return new Promise<void>((resolve, reject) => {
    const script = document.createElement('script');
    script.id = scriptId;
    script.src = `https://www.google.com/recaptcha/enterprise.js?render=${encodeURIComponent(siteKey)}`;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error('Could not load reCAPTCHA.'));
    document.head.appendChild(script);
  });
}

function waitForReady() {
  return new Promise<void>((resolve, reject) => {
    const enterprise = window.grecaptcha?.enterprise;
    if (!enterprise) {
      reject(new Error('reCAPTCHA is not available.'));
      return;
    }
    enterprise.ready(resolve);
  });
}
