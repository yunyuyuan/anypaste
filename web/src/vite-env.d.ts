/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** App version baked in at build time (the release git tag, e.g. "v1.2.3"); undefined in dev. */
  readonly VITE_APP_VERSION?: string;
}
