export interface ShopSettings {
  shop_name: string
  currency: 'LKR' | 'USD'
  tax_rate: string
  service_charge: string
  receipt_header: string
  receipt_footer: string
  webhook_url: string
  backup_frequency: string
  theme: string
  language: string
  whatsapp_diagnosing: string
  whatsapp_repairing: string
  whatsapp_ready: string
  whatsapp_completed: string
  whatsapp_warranty_checking: string
  whatsapp_warranty_ready: string
  whatsapp_warranty_returned: string
}

export const defaultShopSettings: ShopSettings = {
  shop_name: 'Universal Repair POS',
  currency: 'LKR',
  tax_rate: '0',
  service_charge: '0',
  receipt_header: 'Thank you for choosing us!',
  receipt_footer: 'Please keep this receipt for warranty or service reference.',
  webhook_url: '',
  backup_frequency: 'daily',
  theme: 'light',
  language: 'en',
  whatsapp_diagnosing: 'Hello {customer}, we have started diagnosing your job {job_number}. We will update you when work begins.',
  whatsapp_repairing: 'Hello {customer}, work is now in progress on your job {job_number}.',
  whatsapp_ready: 'Hello {customer}, your job {job_number} is complete and ready for collection.',
  whatsapp_completed: 'Hello {customer}, job {job_number} has been delivered. Thank you for choosing us.',
  whatsapp_warranty_checking: 'Hello {customer}, we are now checking your warranty return {job_number}. We will update you with the result.',
  whatsapp_warranty_ready: 'Hello {customer}, your warranty return {job_number} has been checked and is ready for collection. Please contact us if you need more information.',
  whatsapp_warranty_returned: 'Hello {customer}, warranty return {job_number} has been returned to you. Thank you.',
}

const storageKey = 'computer_shop_settings'

export function loadShopSettings(): ShopSettings {
  if (typeof window === 'undefined') return defaultShopSettings
  try {
    const stored = JSON.parse(localStorage.getItem(storageKey) || '{}')
    return { ...defaultShopSettings, ...stored }
  } catch {
    return defaultShopSettings
  }
}

export function saveShopSettings(settings: ShopSettings) {
  localStorage.setItem(storageKey, JSON.stringify(settings))
  window.dispatchEvent(new CustomEvent('shop-settings-changed'))
}

export function formatMoney(value: number) {
  const { currency } = loadShopSettings()
  return new Intl.NumberFormat('en-LK', { style: 'currency', currency }).format(value)
}

export function renderWhatsAppTemplate(template: string, values: { customer: string; jobNumber: string }) {
  return template
    .split('{customer}').join(values.customer)
    .split('{job_number}').join(values.jobNumber)
}

export function buildWhatsAppUrl(phone: string, message: string) {
  const normalizedPhone = phone.replace(/\D/g, '').replace(/^0/, '94')
  return `https://wa.me/${normalizedPhone}?text=${encodeURIComponent(message)}`
}
