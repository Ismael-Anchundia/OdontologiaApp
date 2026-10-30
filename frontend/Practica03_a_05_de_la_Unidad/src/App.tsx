import { BrowserRouter, Routes, Route, Navigate, Outlet, useLocation } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './components/Dashboard';
import Login from './components/Login';
import ServiceWorkerUpdate from './components/ServiceWorkerUpdate';
import InstallPWAButton from './components/InstallPWAButton';
import NetworkStatusBanner from './components/NetworkStatusBanner';
import { AuthProvider, useAuth } from './context/AuthContext';
import { canAccessPage } from './services/permissions';

const ProtectedRoute = () => {
  const { isAuthenticated, user } = useAuth();
  const { pathname } = useLocation();
  if (!isAuthenticated) return <Navigate to="/login" replace/>;
  if (!canAccessPage(user?.dentalRole, pathname)) return <Navigate to="/" replace/>;
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
      <Route path="personal" element={<Dashboard/>}/>
      <Route path="*" element={<Navigate to="/" replace/>}/>
    </Route></Route>
  </Routes></BrowserRouter><ServiceWorkerUpdate/></AuthProvider>;
}

export default App;
