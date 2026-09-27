import { useState } from 'react'
import { Bell, Database, DollarSign, Globe, MessageCircle, Printer, RotateCcw, Save } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { defaultShopSettings, loadShopSettings, saveShopSettings } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'

export function AdminSettings() {
  const [settings, setSettings] = useState(loadShopSettings)

  const update = (field: keyof typeof settings, value: string) => setSettings((current) => ({ ...current, [field]: value }))

  const handleSave = () => {
    saveShopSettings(settings)
    toastHelpers.apiSuccess('Settings', 'Shop preferences saved on this device')
  }

  const handleReset = () => setSettings(defaultShopSettings)

  return (
    <div className="max-w-5xl space-y-6 p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">System Settings</h1>
          <p className="text-muted-foreground">Configure your computer shop, money display, and customer messages.</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleReset}><RotateCcw className="mr-2 h-4 w-4" />Reset</Button>
          <Button onClick={handleSave}><Save className="mr-2 h-4 w-4" />Save changes</Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2"><Globe className="h-5 w-5" />Shop Information</CardTitle><CardDescription>The final business name can be changed later.</CardDescription></CardHeader>
          <CardContent className="space-y-4">
            <Field label="Shop Name"><Input value={settings.shop_name} onChange={(event) => update('shop_name', event.target.value)} /></Field>
            <Field label="Language"><select className="w-full rounded-md border border-input bg-background p-2" value={settings.language} onChange={(event) => update('language', event.target.value)}><option value="en">English</option></select></Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2"><DollarSign className="h-5 w-5" />Financial Settings</CardTitle><CardDescription>LKR is the default. Tax and service charge may stay empty or zero.</CardDescription></CardHeader>
          <CardContent className="space-y-4">
            <Field label="Currency"><select className="w-full rounded-md border border-input bg-background p-2" value={settings.currency} onChange={(event) => update('currency', event.target.value)}><option value="LKR">Sri Lankan Rupee (LKR / Rs.)</option><option value="USD">US Dollar (USD / $)</option></select></Field>
            <div className="grid grid-cols-2 gap-4">
              <Field label="Tax Rate % (optional)"><Input type="number" min="0" step="0.01" placeholder="0" value={settings.tax_rate} onChange={(event) => update('tax_rate', event.target.value)} /></Field>
              <Field label="Service Charge % (optional)"><Input type="number" min="0" step="0.01" placeholder="0" value={settings.service_charge} onChange={(event) => update('service_charge', event.target.value)} /></Field>
            </div>
            <p className="text-xs text-muted-foreground">New transactions currently use zero tax and zero service charge.</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2"><Printer className="h-5 w-5" />Receipt Text</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <Field label="Receipt Header"><Input value={settings.receipt_header} onChange={(event) => update('receipt_header', event.target.value)} /></Field>
            <Field label="Receipt Footer"><Input value={settings.receipt_footer} onChange={(event) => update('receipt_footer', event.target.value)} /></Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2"><Bell className="h-5 w-5" />Discord / Webhook Notifications</CardTitle><CardDescription>Saved now for the planned Discord integration. Automatic webhook sending will be wired in a later pass.</CardDescription></CardHeader>
          <CardContent><Field label="Webhook URL (optional)"><Input type="url" placeholder="https://discord.com/api/webhooks/..." value={settings.webhook_url} onChange={(event) => update('webhook_url', event.target.value)} /></Field></CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><MessageCircle className="h-5 w-5" />WhatsApp Message Templates</CardTitle><CardDescription>Use {'{customer}'} and {'{job_number}'} as placeholders. Repair Tickets opens WhatsApp Web with the chosen message already typed.</CardDescription></CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <Template label="Diagnosing" value={settings.whatsapp_diagnosing} onChange={(value) => update('whatsapp_diagnosing', value)} />
          <Template label="In progress" value={settings.whatsapp_repairing} onChange={(value) => update('whatsapp_repairing', value)} />
          <Template label="Waiting for customer" value={settings.whatsapp_ready} onChange={(value) => update('whatsapp_ready', value)} />
          <Template label="Delivered / completed" value={settings.whatsapp_completed} onChange={(value) => update('whatsapp_completed', value)} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Database className="h-5 w-5" />System Status</CardTitle></CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {['Database connected', 'API online', 'Daily backups enabled', 'Local self-host'].map((item) => <Badge key={item} variant="outline" className="justify-center py-2">{item}</Badge>)}
        </CardContent>
      </Card>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div><label className="mb-2 block text-sm font-medium">{label}</label>{children}</div>
}

function Template({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <Field label={label}><Textarea rows={4} value={value} onChange={(event) => onChange(event.target.value)} /></Field>
}
