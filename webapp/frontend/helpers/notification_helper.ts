export function notify(title: string, body = '') {
  if (!('Notification' in window)) {
    console.log('This browser does not support notifications')
    return
  }
  if (Notification.permission !== 'granted') {
    Notification.requestPermission().then((permission) => {
      if (permission === 'granted') {
        showNotification(title, body)
      }
    })
    return
  }
  showNotification(title, body)
}

type NotificationOptions = {
  body?: string
  icon?: string
}

function showNotification(title: string, body: string) {
  const options: NotificationOptions = {
    icon: '/image/notification_logo_green.png',
  }
  if (body) {
    options.body = body
  }
  const n = new Notification(title, options)
  setTimeout(n.close.bind(n), 4000)
}
