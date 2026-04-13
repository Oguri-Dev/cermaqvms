import { Routes, Route, useLocation } from 'react-router-dom';
import Navbar from './components/Navbar';
import Launch from './pages/Launch';
import Monitor from './pages/Monitor';
import Config from './pages/Config';
import Screen from './pages/Screen';

function App() {
  const location = useLocation();
  const showNavbar = location.pathname === '/monitor' || location.pathname === '/config';

  return (
    <>
      {showNavbar && <Navbar />}
      <Routes>
        <Route path="/" element={<Launch />} />
        <Route path="/monitor" element={<Monitor />} />
        <Route path="/config" element={<Config />} />
        <Route path="/screen/:slot" element={<Screen />} />
      </Routes>
    </>
  );
}

export default App;
