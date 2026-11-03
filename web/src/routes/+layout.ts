// The production UI is a static single-page build embedded into the Go binary.
// A fallback page lets dynamic routes such as /transactions/[id] load on reload.
export const prerender = false;
export const ssr = false;
