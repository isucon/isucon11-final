export type Course = {
  id: string
  code: string
  type: string
  name: string
  description: string
  credit: number
  teacher: string
  keywords: string
  period: number
  dayOfWeek: string
}

export type Announcement = {
  id: string
  courseName: string
  title: string
  message?: string
  createdAt: string
}

export type ClassInfo = {
  id: string
  part: number
  title: string
  description: string
  submissionClosedAt?: string
}
