export type DentalRole = 'admin' | 'receptionist' | 'dentist' | 'assistant' | 'auditor';
export type Capability = 'patients' | 'appointments' | 'clinical' | 'inventory' | 'plans' | 'staff' | 'reports';

// Refleja RequireDentalRoles de routes.go para no mostrar módulos que la API rechazaría.
const capabilities: Record<Capability, DentalRole[]> = {
  patients: ['admin', 'receptionist', 'dentist', 'assistant'],
  appointments: ['admin', 'receptionist', 'dentist', 'assistant'],
  clinical: ['admin', 'dentist', 'assistant'],
  inventory: ['admin', 'assistant', 'auditor'],
  plans: ['admin', 'dentist', 'receptionist'],
  staff: ['admin'],
  reports: ['admin', 'auditor'],
};

const pageCapability: Record<string, Capability | null> = {
  '/': null,
  '/agenda': 'appointments',
  '/pacientes': 'patients',
  '/expediente': 'clinical',
  '/periodontograma': 'clinical',
  '/tratamientos': 'plans',
  '/inventario': 'inventory',
  '/reportes': 'reports',
  '/personal': 'staff',
};

export const roleLabels: Record<DentalRole, string> = {
  admin: 'Administrador', receptionist: 'Recepción', dentist: 'Odontólogo', assistant: 'Asistente', auditor: 'Auditor',
};

export const isDentalRole = (value: unknown): value is DentalRole => typeof value === 'string' && value in roleLabels;

export const hasCapability = (role: DentalRole | undefined, capability: Capability) => Boolean(role && capabilities[capability].includes(role));

export const canAccessPage = (role: DentalRole | undefined, path: string) => {
  const capability = pageCapability[path];
  if (capability === undefined) return true;
  return capability === null ? Boolean(role) : hasCapability(role, capability);
};
