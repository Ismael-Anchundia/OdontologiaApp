import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './components/Dashboard';
import Login from './components/Login';
import ServiceWorkerUpdate from './components/ServiceWorkerUpdate';
import InstallPWAButton from './components/InstallPWAButton';
import NetworkStatusBanner from './components/NetworkStatusBanner';
import { AuthProvider, useAuth } from './context/AuthContext';

const ProtectedRoute = () => {
  const { isAuthenticated, user } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace/>;
  if (user?.rol !== 'admin') return <Navigate to="/login" replace state={{ message: 'Tu cuenta todavía no tiene un perfil clínico autorizado.' }}/>;
  return <Outlet/>;
};

function App() {
  return <AuthProvider><NetworkStatusBanner/><InstallPWAButton/><BrowserRouter><Routes>
    <Route path="/login" element={<Login/>}/>
    <Route element={<ProtectedRoute/>}><Route path="/" element={<Layout/>}>
      <Route index element={<Dashboard/>}/>
      <Route path="agenda" element={<Dashboard/>}/><Route path="pacientes" element={<Dashboard/>}/>
      <Route path="expediente" element={<Dashboard/>}/><Route path="periodontograma" element={<Dashboard/>}/><Route path="tratamientos" element={<Dashboard/>}/>
      <Route path="inventario" element={<Dashboard/>}/><Route path="reportes" element={<Dashboard/>}/>
      <Route path="*" element={<Navigate to="/" replace/>}/>
    </Route></Route>
  </Routes></BrowserRouter><ServiceWorkerUpdate/></AuthProvider>;
}

export default App;
