import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";

import { useAuth } from "./AuthContext";

export interface RequireAuthProps {
  children: ReactNode;
  /** Where an unauthenticated visitor is sent. */
  redirectTo?: string;
}

/**
 * Route guard for the /werkstatt area.
 *
 * Without a token it redirects to the login page and remembers where the
 * visitor was heading, so a successful login can return there. Once the token
 * exists the guarded content renders.
 */
export default function RequireAuth({ children, redirectTo = "/werkstatt/login" }: RequireAuthProps) {
  const { token } = useAuth();
  const location = useLocation();

  if (!token) {
    return <Navigate to={redirectTo} replace state={{ from: location }} />;
  }

  return <>{children}</>;
}
