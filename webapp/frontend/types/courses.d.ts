export type Course = {
  id: string
  code: string
  type: string
  name: string
  description: string
  credit: number
  classroom: string
  capacity: number
  teacher: string
  keywords: string
  period: number
  dayOfWeek: string
  semester: string
  year: number
  requiredCourses: Array<string>
}

export type Announcement = {
  id: string
  courseName: string
  title: string
  message?: string
  createdAt: string
}

export type Document = {
  id: string
  classId: string
  name: string
}

export type Assignment = {
  id: string
  classId: string
  name: string
  description: string
}

export type ClassInfo = {
  id: string
  part: number
  title: string
  description: string
}

export type Classwork = ClassInfo & {
  documents: Array<Document>
  assignments: Array<Assignment>
}

export type Grade = {
  summary: {
    //受講した全スコアの総和にコース単位重みづけて総和( sum(total_score*course.credit)/100 )
    gpt: number
    credits: number //取得単位数or コース数
    //全学生のgpt統計値
    gptAvg: number
    gptStd: number //偏差値
    gptMax: number
    gptMin: number
  }

  courses: [
    {
      name: string
      code: string //UNIQUE
      totalScore: number //コース点数=sum(class.score)
      totalScoreAvg: number
      totalScoreStd: number //偏差値
      totalScoreMax: number
      totalScoreMin: number
      classScores: [
        {
          title: string
          part: number
          score: number //課題点数0~100
          submitters: number //課題提出者数
        }
      ]
    }
  ]
}
