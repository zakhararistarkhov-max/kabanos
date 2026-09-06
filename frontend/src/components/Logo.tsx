import type { CSSProperties } from "react";

// Kabanos mascot: a cheerful boar (кабан) pressing a barbell overhead while
// running on a treadmill — front view. Self-contained colored illustration that
// reads on both light and dark backgrounds; scales from favicon to hero size.
export function KabanosMark({ size = 40, className }: { size?: number; className?: string }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      role="img"
      aria-label="Kabanos"
    >
      {/* ===== treadmill ===== */}
      <rect x="49" y="34" width="2.6" height="20" rx="1.3" fill="#475569" />
      <rect x="46.5" y="30" width="9" height="6.5" rx="1.6" fill="#334155" />
      <rect x="48" y="31.5" width="6" height="3.5" rx="1" fill="#38bdf8" opacity="0.85" />
      <path d="M8 60 L18 52 L58 52 L54 60 Z" fill="#334155" />
      <path d="M12 58.5 L20 53.2 L54.5 53.2 L51.5 58.5 Z" fill="#0f172a" />
      <path d="M22 56.4 h9" stroke="#38bdf8" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M34 56.4 h7" stroke="#38bdf8" strokeWidth="1.5" strokeLinecap="round" opacity="0.55" />
      <rect x="8" y="59" width="6" height="3.5" rx="1.5" fill="#475569" />
      <rect x="50" y="59" width="6" height="3.5" rx="1.5" fill="#475569" />

      {/* ===== barbell overhead ===== */}
      <rect x="14" y="8.4" width="36" height="3.2" rx="1.6" fill="#cbd5e1" />
      <rect x="18.5" y="7.2" width="2.4" height="5.6" rx="1" fill="#94a3b8" />
      <rect x="43.1" y="7.2" width="2.4" height="5.6" rx="1" fill="#94a3b8" />
      <rect x="8" y="4.5" width="6.5" height="11" rx="2.2" fill="#64748b" />
      <rect x="49.5" y="4.5" width="6.5" height="11" rx="2.2" fill="#64748b" />
      <rect x="9.6" y="6.6" width="3.3" height="6.8" rx="1.3" fill="#94a3b8" />
      <rect x="51.1" y="6.6" width="3.3" height="6.8" rx="1.3" fill="#94a3b8" />

      {/* ===== arms raised to the bar ===== */}
      <path d="M25 33 C21 27 20 18 20 12" stroke="#c58b76" strokeWidth="5.4" strokeLinecap="round" />
      <path d="M39 33 C43 27 44 18 44 12" stroke="#c58b76" strokeWidth="5.4" strokeLinecap="round" />
      <ellipse cx="20" cy="11.5" rx="3.4" ry="3" fill="#8a5a44" />
      <ellipse cx="44" cy="11.5" rx="3.4" ry="3" fill="#8a5a44" />

      {/* ===== torso ===== */}
      <path d="M24 32 Q32 29 40 32 L38 45 Q32 48 26 45 Z" fill="#c58b76" />
      <path d="M24 32 Q32 29 40 32 L38 45 Q32 48 26 45 Z" fill="none" stroke="#a86f5b" strokeWidth="1.2" />
      <path d="M32 33 V44" stroke="#a86f5b" strokeWidth="1.2" opacity="0.5" />

      {/* ===== legs (running stride) ===== */}
      <path d="M28 45 C26 49 25 52 26 55" stroke="#c58b76" strokeWidth="5" strokeLinecap="round" />
      <ellipse cx="26.5" cy="55.5" rx="3.4" ry="2.4" fill="#8a5a44" />
      <path d="M37 45 C40 47 40 50 37.5 52.5" stroke="#c58b76" strokeWidth="5" strokeLinecap="round" />
      <ellipse cx="37" cy="52.8" rx="3.4" ry="2.4" fill="#8a5a44" />

      {/* ===== head ===== */}
      <path d="M17 22 L14 12 L25 18 Z" fill="#8a5a44" />
      <path d="M47 22 L50 12 L39 18 Z" fill="#8a5a44" />
      <path d="M18.5 20.5 L17 14 L24 18 Z" fill="#c58b76" />
      <path d="M45.5 20.5 L47 14 L40 18 Z" fill="#c58b76" />
      <ellipse cx="32" cy="22" rx="13.5" ry="11.5" fill="#c58b76" />
      <ellipse cx="32" cy="22" rx="13.5" ry="11.5" fill="none" stroke="#a86f5b" strokeWidth="1.3" />
      {/* headband */}
      <path d="M19 16.5 Q32 11 45 16.5 L45 20.2 Q32 15 19 20.2 Z" fill="#38bdf8" />
      <circle cx="44" cy="18" r="2.6" fill="#0ea5e9" />
      <path d="M45 17 L51 14 L50 19 Z" fill="#38bdf8" />
      <path d="M45 19 L51 21 L48 22.4 Z" fill="#0ea5e9" />
      {/* happy eyes */}
      <path d="M24 21 q2.4 -3.4 4.8 0" stroke="#26313f" strokeWidth="2.2" strokeLinecap="round" />
      <path d="M35.2 21 q2.4 -3.4 4.8 0" stroke="#26313f" strokeWidth="2.2" strokeLinecap="round" />
      {/* cheeks */}
      <circle cx="22" cy="26" r="2.4" fill="#e79b86" opacity="0.7" />
      <circle cx="42" cy="26" r="2.4" fill="#e79b86" opacity="0.7" />
      {/* snout */}
      <ellipse cx="32" cy="27.5" rx="7.5" ry="5.4" fill="#a86f5b" />
      <ellipse cx="29.5" cy="27.5" rx="1.5" ry="2.1" fill="#5c3d2e" />
      <ellipse cx="34.5" cy="27.5" rx="1.5" ry="2.1" fill="#5c3d2e" />
      {/* tusks */}
      <path d="M27 30.5 q-2.2 0.6 -2 4 q2.3 -0.4 2.4 -3.4 Z" fill="#f8fafc" />
      <path d="M37 30.5 q2.2 0.6 2 4 q-2.3 -0.4 -2.4 -3.4 Z" fill="#f8fafc" />
    </svg>
  );
}

// KabanosLogo pairs the mark with the wordmark.
export function KabanosLogo({
  size = 34,
  wordSize = "1.125rem",
  className,
}: {
  size?: number;
  wordSize?: CSSProperties["fontSize"];
  className?: string;
}) {
  return (
    <span className={`inline-flex items-center gap-2 ${className ?? ""}`}>
      <KabanosMark size={size} />
      <span className="font-black tracking-tight text-brand" style={{ fontSize: wordSize }}>
        Kabanos
      </span>
    </span>
  );
}
