// Auth boundary for the isolated component fixture. RPCs still use the real query hooks.
export function useAuth() {
  const params = new URLSearchParams(window.location.search)
  return { isAuthenticated: !params.has('guest'), isOfflineReadonly: false, isLoading: false }
}
