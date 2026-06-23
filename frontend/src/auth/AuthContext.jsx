import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { getToken, clearToken, getMe, setUnauthorizedHandler } from '../api/grids';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);      // { username, role } | null
  const [loading, setLoading] = useState(true); // validando el token guardado

  const logout = useCallback(() => {
    clearToken();
    setUser(null);
  }, []);

  // Al cargar, validar el token persistido (si existe) contra /auth/me
  useEffect(() => {
    setUnauthorizedHandler(() => setUser(null));
    if (!getToken()) {
      setLoading(false);
      return;
    }
    getMe()
      .then(me => setUser(me))
      .catch(() => clearToken())
      .finally(() => setLoading(false));
  }, []);

  const value = {
    user,
    loading,
    isAdmin: user?.role === 'admin',
    setUser, // usado por la pantalla de login tras autenticar
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth debe usarse dentro de AuthProvider');
  return ctx;
}
