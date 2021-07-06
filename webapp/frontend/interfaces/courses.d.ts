/* eslint-disable camelcase */
export interface Course {
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

export interface Announcement {
  id: string
  courseName: string
  title: string
  message?: string
  createdAt: string
}

export interface Document {
  id: string
  classId: string
  name: string
}

export interface Assignment {
  id: string
  classId: string
  name: string
  description: string
}

export interface ClassInfo {
  id: string
  part: number
  title: string
  description: string
}

export interface Classwork extends ClassInfo {
  documents: Array<Document>
  assignments: Array<Assignment>
}
