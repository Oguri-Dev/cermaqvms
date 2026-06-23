import { useState } from 'react';
import { login, setToken } from '../api/grids';
import { useAuth } from '../auth/AuthContext';
import './Login.css';

export default function Login() {
  const { setUser } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(false);
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const res = await login(username.trim(), password, remember);
      setToken(res.token, remember);
      setUser(res.user);
    } catch (err) {
      setError(err.message || 'No se pudo iniciar sesión');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="login">
      <form className="login-card" onSubmit={submit}>
        <div className="login-brand">
          <img src="/logo.png" alt="Cermaq" className="login-logo" />
          <span className="login-title">OMNIFISH VMS</span>
        </div>

        <label className="login-field">
          <span>Usuario</span>
          <input value={username} onChange={e => setUsername(e.target.value)} autoFocus autoComplete="username" />
        </label>
        <label className="login-field">
          <span>Contraseña</span>
          <input type="password" value={password} onChange={e => setPassword(e.target.value)} autoComplete="current-password" />
        </label>

        <label className="login-remember">
          <input type="checkbox" checked={remember} onChange={e => setRemember(e.target.checked)} />
          <span>Recordar sesión en este equipo (para el muro de pantallas)</span>
        </label>

        {error && <div className="login-error">{error}</div>}

        <button type="submit" className="login-btn" disabled={busy || !username || !password}>
          {busy ? 'Ingresando...' : 'Ingresar'}
        </button>
      </form>
    </div>
  );
}
