import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { canAccessPage, roleLabels } from '../services/permissions';

interface SidebarProps { isCollapsed: boolean; isMobileOpen: boolean; closeMobileMenu: () => void }
const items = [
  ['/', 'Resumen', 'M3 3h8v8H3zM13 3h8v5h-8zM13 10h8v11h-8zM3 13h8v8H3z'],
  ['/agenda', 'Agenda', 'M8 2v4m8-4v4M3 10h18M5 4h14a2 2 0 012 2v13H3V6a2 2 0 012-2z'],
  ['/pacientes', 'Pacientes', 'M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2m16-11a4 4 0 11-8 0 4 4 0 018 0m2 4a4 4 0 014 4v3'],
  ['/expediente', 'Expediente clínico', 'M8 3H5a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2V8l-5-5H8zm7 0v5h5M12 11v7m-3.5-3.5h7'],
  ['/periodontograma', 'Periodontograma', 'M12 3v3m0 12v3m9-9h-3M6 12H3m15.36-6.36-2.12 2.12M7.76 16.24l-2.12 2.12m12.72 0-2.12-2.12M7.76 7.76 5.64 5.64M16 12a4 4 0 11-8 0 4 4 0 018 0z'],
  ['/tratamientos', 'Tratamientos', 'M5 3h14v18H5zM8 7h8m-8 4h8m-8 4h5'],
  ['/inventario', 'Inventario', 'M4 6h16v14H4zM8 6V4h8v2m-8 5h8m-8 4h8'],
  ['/reportes', 'Reportes', 'M4 19V5m0 14h17M8 16v-5m5 5V7m5 9v-4'],
  ['/personal', 'Personal y roles', 'M16 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2m16-11a4 4 0 11-8 0 4 4 0 018 0m1 7h5m-2.5-2.5v5'],
];

const Sidebar = ({ isCollapsed, isMobileOpen, closeMobileMenu }: SidebarProps) => {
  const { user } = useAuth();
  const { pathname } = useLocation();
  return <aside className={`fixed inset-y-0 left-0 z-50 flex transform flex-col bg-emerald-950 text-white transition-all duration-300 ${isMobileOpen?'translate-x-0':'-translate-x-full'} w-64 md:relative md:translate-x-0 ${isCollapsed?'md:w-20':'md:w-64'}`}>
    <div className={`border-b border-emerald-900 px-4 py-4 md:px-6 ${isCollapsed?'md:text-center':''}`}><div className="whitespace-nowrap text-xl font-bold"><span className="md:hidden">Dentalia</span><span className="hidden md:inline">{isCollapsed?'D':'Dentalia'}</span></div>{!isCollapsed&&<p className="mt-1 whitespace-nowrap text-xs font-medium tracking-wider text-emerald-200">SISTEMA ODONTOLÓGICO</p>}</div>
    <nav className="flex-1 space-y-2 overflow-y-auto p-3">{items.filter(([to])=>canAccessPage(user?.dentalRole,to)).map(([to,label,icon])=>{const active=to==='/'?pathname==='/':pathname.startsWith(to);return <Link key={to} to={to} onClick={closeMobileMenu} title={label} className={`flex items-center gap-3 rounded-lg p-3 transition ${active?'bg-emerald-700 text-white':'text-emerald-100 hover:bg-emerald-900'} ${isCollapsed?'md:justify-center':''}`}><svg className="h-5 w-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={icon}/></svg><span className={`whitespace-nowrap text-sm ${isCollapsed?'md:hidden':''}`}>{label}</span></Link>})}</nav>
    <div className="border-t border-emerald-900 p-3 text-xs text-emerald-200">{isCollapsed?<p className="text-center uppercase">{user?.dentalRole?roleLabels[user.dentalRole].slice(0,3):''}</p>:<p>Sesión: <span className="font-semibold uppercase">{user?.dentalRole?roleLabels[user.dentalRole]:''}</span></p>}</div>
  </aside>;
};
export default Sidebar;
