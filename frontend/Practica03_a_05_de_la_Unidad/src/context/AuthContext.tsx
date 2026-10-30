// src/context/AuthContext.tsx
import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';
import { clearToken, saveToken } from '../services/api';
import { isDentalRole, type DentalRole } from '../services/permissions';

// 1. Tipos de rol que maneja la aplicación
export type Rol = 'admin' | 'cliente';

// 2. Usuario autenticado: correo + rol (lo entrega la API en /api/login)
export interface Usuario {
  email: string;
  rol: Rol;
  referido_id?: number | null;
  dentalRole?: DentalRole;
}

interface AuthContextType {
  isAuthenticated: boolean;
  user: Usuario | null;
  login: (usuario: Usuario, token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth debe ser usado dentro de un AuthProvider");
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [user, setUser] = useState<Usuario | null>(() => {
    try {
      const stored = JSON.parse(localStorage.getItem('sigo_dental_user') ?? 'null') as Usuario | null;
      // Sesiones anteriores sin perfil clínico deben volver a iniciar sesión.
      return stored && isDentalRole(stored.dentalRole) ? stored : null;
    } catch { return null; }
  });
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(() => Boolean(localStorage.getItem('sigo_dental_token') && user));

  const login = (usuario: Usuario, token: string) => {
    saveToken(token);
    localStorage.setItem('sigo_dental_user', JSON.stringify(usuario));
    setIsAuthenticated(true);
    setUser(usuario);
  };

  const logout = useCallback(() => {
    clearToken();
    localStorage.removeItem('sigo_dental_user');
    setIsAuthenticated(false);
    setUser(null);
  }, []);

  useEffect(() => {
    if (!isAuthenticated) return;
    let timer = 0;
    let lastReset = 0;
    const resetIdleTimer = () => {
      const now = Date.now();
      if (now - lastReset < 30_000) return;
      lastReset = now;
      window.clearTimeout(timer);
      timer = window.setTimeout(logout, 15 * 60 * 1000);
    };
    const events = ['pointerdown', 'keydown', 'touchstart', 'mousemove'] as const;
    events.forEach(event => window.addEventListener(event, resetIdleTimer, { passive: true }));
    resetIdleTimer();
    return () => {
      window.clearTimeout(timer);
      events.forEach(event => window.removeEventListener(event, resetIdleTimer));
    };
  }, [isAuthenticated, logout]);

  useEffect(() => {
    const expire = () => {
      clearToken();
      localStorage.removeItem('sigo_dental_user');
      setIsAuthenticated(false);
      setUser(null);
    };
    window.addEventListener('session-expired', expire);
    return () => window.removeEventListener('session-expired', expire);
  }, []);

  return (
    <AuthContext.Provider value={{ isAuthenticated, user, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};
