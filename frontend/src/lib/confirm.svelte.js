// In-app confirmation dialog. Used instead of window.confirm(), which some
// in-app browsers (e.g. phone captive-portal sheets) block or hide.
export const dialog = $state({ open: false, message: '', confirmLabel: 'OK', danger: false });
let resolver = null;

/** Ask a yes/no question; resolves true if confirmed. */
export function ask(message, { confirmLabel = 'OK', danger = false } = {}) {
  resolver?.(false); // replace any dialog already open
  Object.assign(dialog, { open: true, message, confirmLabel, danger });
  return new Promise((resolve) => (resolver = resolve));
}

export function answer(ok) {
  dialog.open = false;
  resolver?.(ok);
  resolver = null;
}
