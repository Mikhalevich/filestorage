const sizeUnits = ['B', 'KB', 'MB', 'GB', 'TB']

export function formatSize(bytes: number): string {
  let value = bytes
  let unit = 0

  while (value >= 1024 && unit < sizeUnits.length - 1) {
    value /= 1024
    unit++
  }

  return unit === 0 ? `${value} ${sizeUnits[unit]}` : `${value.toFixed(1)} ${sizeUnits[unit]}`
}

const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' })

export function formatTime(rfc3339: string): string {
  return dateFormat.format(new Date(rfc3339))
}
