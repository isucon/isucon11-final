import { CourseType, DayOfWeek } from '~/types/courses'

export function formatType(type: CourseType): string {
  if (type === 'liberal-arts') {
    return '一般教養'
  } else if (type === 'major-subjects') {
    return '専門'
  } else {
    const _t: never = type
    return ''
  }
}

export function formatPeriod(dayOfWeek: DayOfWeek, period: number): string {
  switch (dayOfWeek) {
    case 'sunday':
      return `日${period}`
    case 'monday':
      return `月${period}`
    case 'tuesday':
      return `火${period}`
    case 'wednesday':
      return `水${period}`
    case 'thursday':
      return `木${period}`
    case 'friday':
      return `金${period}`
    case 'saturday':
      return `土${period}`
    default:
      const _n: never = dayOfWeek
      return ''
  }
}
