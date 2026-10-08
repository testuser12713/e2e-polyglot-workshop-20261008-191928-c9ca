import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { login as requestLogin } from "../api/auth";
import type { Employee } from "../api/auth";
import { setAuthToken } from "../api/client";

export type { Employee };

export interface AuthContextValue {
  token: string | null;
  employee: Employee | null;
  /** Signs in against POST /api/auth/login; rejects with the API's ApiError. */
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

  const login = useCallback(async (email: string, password: string): Promise<void> => {
    const result = await requestLogin(email, password);
    // Set the client token synchronously so a request fired by the page the
    // caller navigates to immediately after never races the state effect.
    setAuthToken(result.token);
    setToken(result.token);
    setEmployee(result.employee);
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
