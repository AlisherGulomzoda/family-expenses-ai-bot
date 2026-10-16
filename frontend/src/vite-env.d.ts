/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** "true" serves the API from MSW mocks in the browser (no backend needed). */
  readonly VITE_API_MOCKS?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
