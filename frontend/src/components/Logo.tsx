export function AppIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 256 256" className={className} aria-hidden="true">
      <rect width="256" height="256" rx="56" fill="var(--brand-tile)" />
      <g
        transform="translate(10,-15)"
        fill="none"
        stroke="var(--brand-ink)"
        strokeWidth={30}
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="M158 76 H130 A28 28 0 0 0 102 104 V210" />
        <path d="M70 136 H166" />
      </g>
    </svg>
  )
}
