import {
  CourseStatus,
  CourseType,
  DayOfWeek,
  RegistrationError,
} from '~/types/courses'

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
    default:
      const _n: never = dayOfWeek
      return ''
  }
}

export function formatStatus(status: CourseStatus): string {
  if (status === 'in-progress') {
    return '講義期間'
  } else if (status === 'registration') {
    return '履修登録期間'
  } else if (status === 'closed') {
    return '終了済み'
  } else {
    const _s: never = status
    return ''
  }
}

export function isRegistrationError(err: any): err is RegistrationError {
  return Object.keys(err).some((k) =>
    ['notRegistrableStatus', 'scheduleConflict', 'courseNotFound'].includes(k)
  )
}

export function formatRegistrationError(
  err: RegistrationError | undefined
): string {
  const message = []
  if (!err) {
    return ''
  }

  for (const key of Object.keys(err)) {
    if (key === 'notRegistrableStatus') {
      message.push('履修登録期間外')
    } else if (key === 'scheduleConflict') {
      message.push('時間割のコンフリクト')
    } else if (key === 'courseNotFound') {
      message.push('科目が存在しない')
    }
  }

  return message.join(', ')
}
