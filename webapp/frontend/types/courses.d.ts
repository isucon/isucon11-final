export type DayOfWeek =
  | 'monday'
  | 'tuesday'
  | 'wednesday'
  | 'thursday'
  | 'friday'

export type CourseType = 'liberal-arts' | 'major-subjects'

export type CourseStatus = 'registration' | 'in-progress' | 'closed'

export type User = {
  code: string
  name: string
  isAdmin: boolean
}

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
  name: string
  credit: number
  description: string
  keywords: string
  status: CourseStatus
  type: CourseType
  code: string
  period: number
  dayOfWeek: DayOfWeek
  teacher: string
}

export type SyllabusCourse = Course & {
  code: string
  type: CourseType
  description: string
  credit: number
  keywords: string
  status: CourseStatus
}

export type AddCourseRequest = Omit<SyllabusCourse, 'id' | 'teacher' | 'status'>

export type SetCourseStatusRequest = {
  status: CourseStatus
}

export type Announcement = {
  id: string
  courseId: string
  courseName: string
  title: string
  unread: boolean
  message?: string
  hasError?: boolean
}

export type AddAnnouncementRequest = Omit<
  Announcement,
  'courseName' | 'unread' | 'hasError'
>

export type AnnouncementResponse = {
  id: string
  courseId: string
  courseName: string
  title: string
  unread: boolean
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

export type AddClassRequest = {
  part: number
  title: string
  description: string
}

type RegisterScoreRequestObject = {
  userCode: string
  score: number
}

export type RegisterScoreRequest = RegisterScoreRequestObject[]

type SummaryGrade = {
  gpa: number
  credits: number
  gpaAvg: number
  gpaTScore: number
  gpaMax: number
  gpaMin: number
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

export type RegistrationError = {
  notRegistrableStatus: string[]
  scheduleConflict: string[]
  courseNotFound: string[]
}
