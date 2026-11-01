import type { CalendarEvent } from '~/types'

const LOCALE = 'de-DE'

export function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

export function addDays(d: Date, n: number): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate() + n)
}

// isoDate formatiert ein Datum als YYYY-MM-DD in lokaler Zeit (nicht UTC).
export function isoDate(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function parseIsoDate(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y!, m! - 1, d!)
}

export function formatTime(d: Date | string): string {
  return new Date(d).toLocaleTimeString(LOCALE, { hour: '2-digit', minute: '2-digit' })
}

export function formatShortDate(d: Date | string): string {
  const date = typeof d === 'string' && d.length === 10 ? parseIsoDate(d) : new Date(d)
  return date.toLocaleDateString(LOCALE, { weekday: 'short', day: 'numeric', month: 'numeric' })
}

export function formatLongDate(d: Date): string {
  return d.toLocaleDateString(LOCALE, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

export function weekdayName(d: Date): string {
  return d.toLocaleDateString(LOCALE, { weekday: 'long' })
}

// daysUntil liefert die Anzahl Kalendertage von heute bis zum Datum (negativ = vorbei).
export function daysUntil(isoDay: string, today = new Date()): number {
  const ms = parseIsoDate(isoDay).getTime() - startOfDay(today).getTime()
  return Math.round(ms / 86_400_000)
}

export function relativeDays(isoDay: string): string {
  const n = daysUntil(isoDay)
  if (n === 0) return 'heute'
  if (n === 1) return 'morgen'
  if (n === -1) return 'gestern'
  if (n < 0) return `seit ${-n} Tagen`
  return `in ${n} Tagen`
}

// eventsOnDay liefert alle Termine, die den Tag beruehren (auch mehrtaegige).
export function eventsOnDay(events: CalendarEvent[], day: Date): CalendarEvent[] {
  const from = startOfDay(day).getTime()
  const to = addDays(day, 1).getTime()
  return events.filter(e => new Date(e.end).getTime() > from && new Date(e.start).getTime() < to
    // Termine mit Dauer 0 (z.B. Erinnerungen) trotzdem anzeigen
    || (e.start === e.end && new Date(e.start).getTime() >= from && new Date(e.start).getTime() < to))
}
