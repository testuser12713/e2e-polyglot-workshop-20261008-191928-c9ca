import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { setAuthToken } from "../api/client";

export interface Employee {
  id: number;
  name: string;
  email: string;
}

export interface AuthContextValue {
  token: string | null;
  employee: Employee | null;
  /** Filled in by the staff-login ticket; inert until then. */
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const TOKEN_KEY = "werkstatt.token";
const EMPLOYEE_KEY = "werkstatt.employee";

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

function readEmployee(): Employee | null {
  try {
    const raw = localStorage.getItem(EMPLOYEE_KEY);
    return raw ? (JSON.parse(raw) as Employee) : null;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(readToken);
  const [employee, setEmployee] = useState<Employee | null>(readEmployee);

  useEffect(() => {
    setAuthToken(token);
    try {
      if (token) {
        localStorage.setItem(TOKEN_KEY, token);
      } else {
        localStorage.removeItem(TOKEN_KEY);
      }
    } catch {
      /* storage unavailable (private mode) — the in-memory token still works */
    }
  }, [token]);

  useEffect(() => {
    try {
      if (employee) {
        localStorage.setItem(EMPLOYEE_KEY, JSON.stringify(employee));
      } else {
        localStorage.removeItem(EMPLOYEE_KEY);
      }
    } catch {
      /* storage unavailable */
    }
  }, [employee]);

  const login = useCallback(async (_email: string, _password: string): Promise<void> => {
    // The staff-login ticket replaces this body with POST /api/auth/login.
    // Until then the context API is complete but deliberately inert.
    throw new Error("Die Anmeldung ist noch nicht verfügbar.");
  }, []);

  const logout = useCallback((): void => {
    setToken(null);
    setEmployee(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ token, employee, login, logout }),
    [token, employee, login, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
