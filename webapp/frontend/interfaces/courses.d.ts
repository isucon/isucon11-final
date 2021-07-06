/* eslint-disable camelcase */
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
  day_of_week: string
  semester: string
  year: number
  required_courses: Array<string>
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
