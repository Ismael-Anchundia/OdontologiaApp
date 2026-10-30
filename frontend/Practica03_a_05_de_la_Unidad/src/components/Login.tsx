// src/components/Login.tsx
import { useState, type FormEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth, type Rol } from "../context/AuthContext";
import { api, clearToken, saveToken } from "../services/api";
import { isDentalRole } from "../services/permissions";
import { useOnlineStatus } from "../hooks/useOnlineStatus";

const Login = () => {
  const [email, setEmail] = useState<string>("");
  const [password, setPassword] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(false);

  const navigate = useNavigate();
  const location = useLocation();
  const { login } = useAuth();
  const online = useOnlineStatus();
  const demoMode = import.meta.env.VITE_DEMO_MODE === 'true';

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (demoMode) {
      login({ email: 'demo@dentalia.local', rol: 'admin', dentalRole: 'admin' }, 'demo-session');
      navigate('/');
      return;
    }
    if (!online) {
      setError("Necesitas conexión a internet para iniciar sesión.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await api<{ email: string; rol: string; referido_id?: number; token: string }>("/api/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      }, { auth: false });
      const rol: Rol = data.rol === "admin" ? "admin" : "cliente";
      saveToken(data.token);
      let profile: { role?: string };
      try {
        profile = await api<{ role?: string }>("/api/dental/me");
      } catch (profileError) {
        clearToken();
        throw profileError;
      }
      if (!isDentalRole(profile.role)) {
        clearToken();
        throw new Error("La cuenta no tiene un perfil clínico activo.");
      }
      login({ email: data.email, rol, referido_id: data.referido_id, dentalRole: profile.role }, data.token);
      navigate("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo iniciar sesión.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4">
      <div className="max-w-md w-full bg-white rounded-xl shadow-md p-8 border border-slate-200">
        <div className="text-center mb-8">
          <h2 className="text-3xl font-bold text-emerald-800">Dentalia</h2>
          <p className="text-slate-500 mt-2">
            Gestión odontológica · Acceso del personal
          </p>
        </div>

        {(location.state as { message?: string } | null)?.message && <p role="status" className="mb-5 rounded-lg bg-green-50 p-3 text-sm text-green-700">{(location.state as { message?: string }).message}</p>}

        {error && (
          <div className="bg-red-50 text-red-600 p-3 rounded-lg text-sm mb-6 text-center border border-red-200">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Correo Electrónico
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
              placeholder="admin@upse.edu.ec"
              required={!demoMode}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              Contraseña
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 rounded-lg border border-slate-300 focus:ring-2 focus:ring-indigo-600 focus:border-indigo-600 outline-none transition"
              placeholder="••••••"
              required={!demoMode}
            />
          </div>

          <button
            type="submit"
            disabled={loading || !online}
            className="w-full bg-indigo-600 text-white font-bold py-3 rounded-lg hover:bg-indigo-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "Validando..." : demoMode ? "Entrar a la demostración" : online ? "Iniciar Sesión" : "Conéctate para iniciar sesión"}
          </button>
        </form>

        {demoMode ? <div className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-4 text-xs leading-5 text-amber-800">Modo demostración sin servidor: usa únicamente datos ficticios. No ingreses información de pacientes.</div> : import.meta.env.DEV && <div className="mt-6 p-4 bg-slate-50 rounded-lg border border-slate-200 text-xs text-slate-600 space-y-1">
          <p className="font-semibold text-slate-700">Cuenta inicial heredada del proyecto:</p>
          <p>Administrador: <span className="font-mono">admin@upse.edu.ec / 123456</span></p>
          <p className="mt-2 text-amber-700">Prototipo: sustituir estas credenciales antes de desplegarlo.</p>
        </div>}
      </div>
    </div>
  );
};

export default Login;
