import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

interface NavbarProps { toggleSidebar: () => void }
const Navbar = ({ toggleSidebar }: NavbarProps) => {
  const { logout, user } = useAuth();
  const navigate = useNavigate();
  return <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white px-4 md:px-8">
    <div className="flex items-center gap-3"><button onClick={toggleSidebar} className="rounded-md p-2 text-slate-600 transition hover:bg-slate-100" aria-label="Alternar menú"><svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16"/></svg></button><h2 className="hidden text-sm font-semibold text-slate-700 sm:block">Clínica Dentalia <span className="font-normal text-slate-400">· Gestión clínica</span></h2></div>
    <div className="flex items-center gap-3"><span className="hidden text-xs text-slate-500 sm:block">{user?.email}</span><span className="grid h-9 w-9 place-items-center rounded-full bg-emerald-100 text-xs font-bold text-emerald-800">MC</span><button onClick={()=>{logout();navigate('/login')}} className="rounded-lg border border-slate-200 px-3 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-50">Salir</button></div>
  </header>;
};
export default Navbar;
