import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Pause, Play } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export interface Banner {
  id: string
  eyebrow: string
  title: string
  description: string
  cta: { label: string; to: string }
  /** Tailwind background classes, e.g. a gradient. */
  className: string
}

/**
 * Auto-advancing banner carousel (presentation only, no data fetching).
 * - Advances every `intervalMs`. The active dot fills left to right and its
 *   animationend event advances the slide, so the fill and the switch can
 *   never drift apart. A manual change restarts the fill.
 * - Pauses while hovered or focused, and has a pause button (WCAG 2.2.2);
 *   pausing freezes the fill and resuming continues from there.
 * - Starts paused when the user prefers reduced motion.
 * - Inactive slides are `inert`, so keyboard focus never lands off-screen.
 */
export function BannerCarousel({ banners, intervalMs = 8_000 }: { banners: Banner[]; intervalMs?: number }) {
  const count = banners.length
  const [index, setIndex] = useState(0)
  const [paused, setPaused] = useState(() => prefersReducedMotion())
  const [hovered, setHovered] = useState(false)
  const [focused, setFocused] = useState(false)

  const running = count > 1 && !paused && !hovered && !focused

  if (count === 0) return null

  const go = (i: number) => setIndex((i + count) % count)

  return (
    <section
      aria-roledescription="carousel"
      aria-label="Promo"
      className="relative overflow-hidden rounded-2xl"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onFocus={() => setFocused(true)}
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget)) setFocused(false)
      }}
    >
      <div
        className="flex transition-transform duration-700 ease-out motion-reduce:transition-none"
        style={{ transform: `translateX(-${index * 100}%)` }}
      >
        {banners.map((b, i) => {
          const active = i === index
          const Heading = i === 0 ? 'h1' : 'h2'
          return (
            <div
              key={b.id}
              role="group"
              aria-roledescription="slide"
              aria-label={`${i + 1} dari ${count}`}
              aria-hidden={!active}
              inert={!active}
              className={cn('w-full shrink-0 px-6 py-10 pb-14 text-white sm:px-10 sm:py-14 sm:pb-16', b.className)}
            >
              <p className="text-sm font-medium text-white/80">{b.eyebrow}</p>
              <Heading className="mt-2 max-w-xl text-3xl font-bold leading-tight sm:text-4xl">{b.title}</Heading>
              <p className="mt-3 max-w-xl text-white/90">{b.description}</p>
              <Button
                size="lg"
                asChild
                className="mt-6 h-10 bg-white px-5 font-semibold text-foreground hover:bg-white/90 focus-visible:ring-white/60"
              >
                <Link to={b.cta.to}>{b.cta.label}</Link>
              </Button>
            </div>
          )
        })}
      </div>

      {count > 1 && (
        <>
          <ArrowButton side="left" label="Banner sebelumnya" onClick={() => go(index - 1)} />
          <ArrowButton side="right" label="Banner berikutnya" onClick={() => go(index + 1)} />

          <div className="absolute inset-x-0 bottom-4 flex items-center justify-center gap-2">
            {banners.map((b, i) => {
              const active = i === index
              return (
                <button
                  key={b.id}
                  type="button"
                  aria-label={`Tampilkan banner ${i + 1}`}
                  aria-current={active}
                  onClick={() => go(i)}
                  className={cn(
                    'relative h-2 overflow-hidden rounded-full bg-white/40 transition-[width] duration-300 outline-none hover:bg-white/60 focus-visible:ring-2 focus-visible:ring-white',
                    active ? 'w-7' : 'w-2',
                  )}
                >
                  {active && (
                    // The fill IS the timer: when it reaches the right edge the
                    // next slide shows. `key` restarts it on every slide change;
                    // pausing freezes it where it is (animation-play-state).
                    <span
                      key={index}
                      aria-hidden="true"
                      className="absolute inset-0 origin-left rounded-full bg-white"
                      style={{
                        animation: `banner-fill ${intervalMs}ms linear forwards`,
                        animationPlayState: running ? 'running' : 'paused',
                      }}
                      onAnimationEnd={() => go(index + 1)}
                    />
                  )}
                </button>
              )
            })}
            <button
              type="button"
              aria-label={paused ? 'Putar banner otomatis' : 'Jeda banner otomatis'}
              onClick={() => setPaused((p) => !p)}
              className="ml-2 rounded-full p-1 text-white/80 outline-none hover:text-white focus-visible:ring-2 focus-visible:ring-white"
            >
              {paused ? <Play className="size-3.5" aria-hidden="true" /> : <Pause className="size-3.5" aria-hidden="true" />}
            </button>
          </div>
        </>
      )}
    </section>
  )
}

function ArrowButton({ side, label, onClick }: { side: 'left' | 'right'; label: string; onClick: () => void }) {
  const Icon = side === 'left' ? ChevronLeft : ChevronRight
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className={cn(
        'absolute top-1/2 hidden -translate-y-1/2 rounded-full bg-black/20 p-2 text-white outline-none transition hover:bg-black/35 focus-visible:ring-2 focus-visible:ring-white sm:block',
        side === 'left' ? 'left-3' : 'right-3',
      )}
    >
      <Icon className="size-5" aria-hidden="true" />
    </button>
  )
}

function prefersReducedMotion() {
  return typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
}
