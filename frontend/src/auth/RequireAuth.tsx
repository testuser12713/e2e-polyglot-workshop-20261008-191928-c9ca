import type { ReactNode } from "react";

export interface RequireAuthProps {
  children: ReactNode;
  /** Where to send an unauthenticated visitor once the guard is real. */
  redirectTo?: string;
}

/**
 * Route guard for the /werkstatt area.
 *
 * The protected-routing ticket replaces the body with the real check
 * (redirect to /werkstatt/login when there is no token). Until then it renders
 * its children so the shell stays reachable and the skeleton can be exercised.
 */
export default function RequireAuth({ children }: RequireAuthProps) {
  return <>{children}</>;
}
