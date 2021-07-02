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
  title: string
  message: string
  createdAt: string
}

export interface Document {
  id: string
  name: string
}

export interface Assignment {
  id: string
  name: string
  description: string
  deadline: string
  // not implemented
  // submitted: boolean
}

export interface Classwork {
  id: string
  title: string
  // not implemented
  // startedAt: string
  description: string
  documents: Array<Document>
  assignments: Array<Assignment>
}
