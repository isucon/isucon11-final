export type DayOfWeek =
  | 'sunday'
  | 'monday'
  | 'tuesday'
  | 'wednesday'
  | 'thursday'
  | 'friday'
  | 'saturday'

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
  submissionClosedAt?: string
}

export type Grade = {
  summary: {
    // 受講した全スコアの総和にコース単位重みづけて総和( sum(total_score*course.credit)/100 )
    gpt: number
    credits: number // 取得単位数or コース数
    // 全学生のgpt統計値
    gptAvg: number
    gptStd: number // 偏差値
    gptMax: number
    gptMin: number
  }

  courses: [
    {
      name: string
      code: string // UNIQUE
      totalScore: number // コース点数=sum(class.score)
      totalScoreAvg: number
      totalScoreStd: number // 偏差値
      totalScoreMax: number
      totalScoreMin: number
      classScores: [
        {
          title: string
          part: number
          score: number // 課題点数0~100
          submitters: number // 課題提出者数
        }
      ]
    }
  ]
}
