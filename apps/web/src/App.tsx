import { CreatePage } from './pages/CreatePage';
import { HomePage } from './pages/HomePage';
import { HowItWorksPage } from './pages/HowItWorksPage';
import { PrivacyPage } from './pages/PrivacyPage';
import { SecretPage } from './pages/SecretPage';
import { TermsPage } from './pages/TermsPage';
import { Footer } from './components/Footer';

export function App() {
  const path = window.location.pathname;

  let page = <HomePage />;
  if (path === '/create') {
    page = <CreatePage />;
  } else if (path.startsWith('/s/')) {
    page = <SecretPage publicId={decodeURIComponent(path.split('/')[2] ?? '')} />;
  } else if (path === '/privacy') {
    page = <PrivacyPage />;
  } else if (path === '/how-it-works') {
    page = <HowItWorksPage />;
  } else if (path === '/terms') {
    page = <TermsPage />;
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <a className="brand" href="/">
          <img className="brand-logo" src="/bquick-secret-logo.svg" alt="" />
          <span>bQuick Secret</span>
        </a>
        <nav aria-label="Primary">
          <a href="/create">Create</a>
          <a href="/how-it-works">How it works</a>
          <a href="/privacy">Privacy</a>
          <a href="/terms">Terms</a>
          <a
            className="icon-link"
            href="https://github.com/binaryquest/bquick-secret"
            target="_blank"
            rel="noreferrer"
            aria-label="View source code on GitHub"
            title="View source code on GitHub"
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" focusable="false">
              <path d="M12 0C5.37 0 0 5.5 0 12.3c0 5.4 3.44 10 8.2 11.6.6.1.82-.26.82-.6v-2.1c-3.34.74-4.04-1.65-4.04-1.65-.55-1.42-1.34-1.8-1.34-1.8-1.09-.76.08-.74.08-.74 1.2.09 1.84 1.27 1.84 1.27 1.08 1.88 2.82 1.34 3.5 1.02.11-.8.42-1.34.76-1.65-2.66-.31-5.46-1.36-5.46-6.08 0-1.34.47-2.44 1.24-3.3-.12-.31-.54-1.56.12-3.25 0 0 1.01-.33 3.3 1.26.96-.27 1.98-.4 3-.41 1.02.01 2.04.14 3 .41 2.29-1.59 3.3-1.26 3.3-1.26.66 1.69.24 2.94.12 3.25.77.86 1.24 1.96 1.24 3.3 0 4.73-2.8 5.76-5.48 6.07.43.38.82 1.13.82 2.28v3.38c0 .34.21.71.83.59C20.56 22.3 24 17.7 24 12.3 24 5.5 18.63 0 12 0Z" />
            </svg>
          </a>
        </nav>
      </header>
      <main>{page}</main>
      <Footer />
    </div>
  );
}
