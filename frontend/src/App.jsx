import { Routes, Route, useLocation, Navigate } from 'react-router-dom';
import Navbar from './components/Navbar';
import Config from './pages/Config';
import Screen from './pages/Screen';
import Login from './pages/Login';
import { useAuth } from './auth/AuthContext';
import { useBootstrap } from './hooks/useBootstrap';

function App() {
  const location = useLocation();
  const { user, loading, isAdmin } = useAuth();
  const isScreen = location.pathname.startsWith('/screen/');

  // Al abrir la app (ventana principal, con sesión), reiniciar los servicios
  // del centro. No se dispara en las ventanas secundarias del muro (/screen/),
  // que comparten este componente.
  useBootstrap(!!user && !isScreen);

  // Mientras se valida el token guardado, evitar parpadeo
  if (loading) {
    return <div className="app-loading">Cargando...</div>;
  }

  // Sin sesión: solo se muestra el login (cualquier ruta cae aquí)
  if (!user) {
    return <Login />;
  }

  return (
    <>
      {!isScreen && <Navbar />}
      <Routes>
        {/* Config solo para admin; el operador es redirigido al wall launcher */}
        <Route path="/" element={isAdmin ? <Config /> : <Navigate to="/screen-launcher" replace />} />
        <Route path="/config" element={isAdmin ? <Config /> : <Navigate to="/" replace />} />
        <Route path="/screen/:slot" element={<Screen />} />
        <Route path="/screen-launcher" element={<OperatorHome />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </>
  );
}

// Pantalla simple para el operador (sin acceso a Configuración): solo el
// botón "Abrir Pantallas" vive en la Navbar, así que aquí basta una guía.
function OperatorHome() {
  return (
    <div className="operator-home">
      <p>Usa <b>Abrir Pantallas</b> en la barra superior para levantar el muro de monitores.</p>
    </div>
  );
}

export default App;
