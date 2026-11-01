export interface Member {
  id: number
  name: string
  color: string
  emoji: string
  discordUserId: string
  icalUrl: string
  sort: number
}

export interface Todo {
  id: number
  title: string
  memberId: number | null
  due: string | null // YYYY-MM-DD
  done: boolean
  doneAt: string | null
  createdAt: string
  createdBy: string
}

export interface Goal {
  id: number
  memberId: number
  title: string
  description: string
  targetDate: string | null // YYYY-MM-DD
  progress: number
  done: boolean
  sort: number
  createdAt: string
}

export interface CalendarEvent {
  id: number
  uid: string
  title: string
  start: string
  end: string
  allDay: boolean
  memberId: number | null
  location: string
  notes: string
  source: 'local' | 'ical'
}

export interface Dashboard {
  now: string
  members: Member[]
  events: CalendarEvent[]
  todos: Todo[]
  goals: { member: Member, goal: Goal | null }[]
}

export interface FeedStatus {
  memberId: number
  events: number
  error: string
  fetchedAt: string
}

export interface WeatherSlot {
  hour: number
  temp: number
  precipProb: number
  icon: string
  text: string
}

export interface WeatherReport {
  enabled: boolean
  place: string
  updatedAt: string
  error?: string
  current: { temp: number, feels: number, wind: number, icon: string, text: string } | null
  day: {
    label: string
    date: string
    min: number
    max: number
    precipProb: number
    uv: number
    icon: string
    text: string
    slots: WeatherSlot[]
    advice: string[]
  } | null
}
