export function exportCsv(filename, rows, columns = null) {
  if (!rows.length) return
  const keys = columns?.length
    ? columns.filter((key) => Object.hasOwn(rows[0], key))
    : Object.keys(rows[0])
  const cell = (value) => {
    let text = String(value ?? '')
    if (/^[=+@\-\t\r]/.test(text)) text = `'${text}`
    return `"${text.replaceAll('"', '""')}"`
  }
  const blob = new Blob(
    [
      '\uFEFF',
      [
        keys.map(cell).join(','),
        ...rows.map((row) => keys.map((key) => cell(row[key])).join(',')),
      ].join('\r\n'),
    ],
    { type: 'text/csv;charset=utf-8;' },
  )
  const url = URL.createObjectURL(blob),
    anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
