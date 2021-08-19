export type DayOfWeek =
  | 'monday'
  | 'tuesday'
  | 'wednesday'
  | 'thursday'
  | 'friday'

export type CourseType = 'liberal-arts' | 'major-subjects'

export type SearchCourseRequest = {
  keywords?: string
  type?: string
  credit?: number
  teacher?: string
  period?: number
  dayOfWeek?: string
  page?: number
}

export type Course = {
  id: string
  code: string
  type: CourseType
  name: string
  description: string
  credit: number
  period: number
  dayOfWeek: DayOfWeek
  teacher: string
  keywords: string
}

export type AddCourseRequest = Omit<Course, 'id' | 'teacher'>

export type Announcement = {
  id: string
  courseId: string
  courseName: string
  title: string
  unread: boolean
  createdAt: string
  message?: string
}

export type AnnouncementResponse = {
  id: string
  courseId: string
  courseName: string
  title: string
  unread: boolean
  createdAt: number
}

export type GetAnnouncementResponse = {
  unreadCount: number
  announcements: AnnouncementResponse[]
}

export type ClassInfo = {
  id: string
  part: number
  title: string
  description: string
  submissionClosed: boolean
  submitted: boolean
}

type SummaryGrade = {
  gpt: number
  credits: number
  gptAvg: number
  gptTScore: number
  gptMax: number
  gptMin: number
}

type ClassScore = {
  title: string
  part: number
  score: number
  submitters: number
}

type CourseGrade = {
  name: string
  code: string
  totalScore: number
  totalScoreAvg: number
  totalScoreTScore: number
  totalScoreMax: number
  totalScoreMin: number
  classScores: ClassScore[]
}

export type Grade = {
  summary: SummaryGrade
  courses: CourseGrade[]
}
