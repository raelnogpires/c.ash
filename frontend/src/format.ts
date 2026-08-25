export const formatBRL = (cents: number) => new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(cents / 100)

export function parseBRL(value: string): number | null {
  const normalized = value.trim().replace(/\s/g, '').replace(/R\$/gi, '').replace(/\./g, '').replace(',', '.')
  if (!/^[-+]?\d+(\.\d{0,2})?$/.test(normalized)) return null
  const number = Number(normalized)
  if (!Number.isFinite(number)) return null
  return Math.round(number * 100)
}

export const formatDate = (date: string) => new Intl.DateTimeFormat('pt-BR', { timeZone: 'UTC', day: '2-digit', month: 'short', year: 'numeric' }).format(new Date(`${date}T00:00:00Z`))
export const today = () => {
  const date = new Date()
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
