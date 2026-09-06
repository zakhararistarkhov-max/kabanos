import { KabanosMark } from "@/components/Logo";

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-md">
        <div className="mb-6 flex flex-col items-center text-center">
          <KabanosMark size={88} />
          <h1 className="mt-3 text-3xl font-black tracking-tight text-brand">Kabanos</h1>
          <p className="mt-1 text-sm text-ink-500">Здоровье под контролем</p>
        </div>
        {children}
      </div>
    </div>
  );
}
