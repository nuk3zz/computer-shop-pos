import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Bell, DollarSign, Globe, ImagePlus, MessageCircle, Printer, RotateCcw, Save, Trash2 } from 'lucide-react'
import apiClient from '@/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { defaultShopSettings, loadShopSettings, saveShopSettings } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import { SystemMaintenance } from '@/components/admin/SystemMaintenance'
import { getMediaUrl } from '@/lib/media'

export function AdminSettings() {
  const [settings, setSettings] = useState(loadShopSettings)
	const [companyName, setCompanyName] = useState('')
	const [description, setDescription] = useState('')
	const [logoURL, setLogoURL] = useState<string | undefined>()
	const [logoFile, setLogoFile] = useState<File | null>(null)
	const [logoPreview, setLogoPreview] = useState('')
	const queryClient = useQueryClient()
	const { data: profile } = useQuery({
		queryKey: ['shop-profile'],
		queryFn: () => apiClient.getShopProfile().then((response) => response.data),
	})

	useEffect(() => {
		if (!profile) return
		setCompanyName(profile.company_name)
		setDescription(profile.description || '')
		setLogoURL(profile.logo_url)
	}, [profile])

  const update = (field: keyof typeof settings, value: string) => setSettings((current) => ({ ...current, [field]: value }))

	const save = useMutation({
		mutationFn: async () => {
			let nextLogoURL = logoURL
			if (logoFile) nextLogoURL = (await apiClient.uploadProductImage(logoFile)).data?.url
			return apiClient.updateShopProfile({ company_name: companyName.trim(), description: description.trim(), logo_url: nextLogoURL })
		},
		onSuccess: (response) => {
			saveShopSettings({ ...settings, shop_name: companyName.trim() })
			if (response.data) queryClient.setQueryData(['shop-profile'], response.data)
			setLogoFile(null)
			setLogoPreview('')
			toastHelpers.apiSuccess('Settings', 'Shop identity and preferences saved')
		},
		onError: (error) => toastHelpers.apiError('Save settings', error),
	})

	const handleSave = () => {
		if (!companyName.trim()) return toastHelpers.apiError('Save settings', new Error('Shop name is required'))
		save.mutate()
	}

  const handleReset = () => setSettings(defaultShopSettings)

  return (
    <div className="max-w-5xl space-y-6 p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">System Settings</h1>
          <p className="text-muted-foreground">Configure your business, money display, and customer messages.</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleReset}><RotateCcw className="mr-2 h-4 w-4" />Reset</Button>
          <Button onClick={handleSave} disabled={save.isPending}><Save className="mr-2 h-4 w-4" />{save.isPending ? 'Saving…' : 'Save changes'}</Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2"><Globe className="h-5 w-5" />Shop Information</CardTitle><CardDescription>The final business name can be changed later.</CardDescription></CardHeader>
          <CardContent className="space-y-4">
            <Field label="Shop Name"><Input value={companyName} onChange={(event) => setCompanyName(event.target.value)} maxLength={150} /></Field>
			<Field label="Sidebar Description"><Input value={description} onChange={(event) => setDescription(event.target.value)} maxLength={200} placeholder="Sales and repair management" /></Field>
			<Field label="Shop Logo">
				<div className="flex items-center gap-3">
					<div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted">{logoPreview || logoURL ? <img src={logoPreview || getMediaUrl(logoURL)} alt="Logo preview" className="h-full w-full object-contain" /> : <ImagePlus className="h-6 w-6 text-muted-foreground" />}</div>
					<div className="min-w-0 flex-1 space-y-2"><Input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(event) => { const file = event.target.files?.[0]; if (!file) return; setLogoFile(file); setLogoPreview(URL.createObjectURL(file)) }} />{(logoURL || logoPreview) && <Button type="button" size="sm" variant="outline" onClick={() => { setLogoURL(undefined); setLogoFile(null); setLogoPreview('') }}><Trash2 className="mr-1.5 h-3.5 w-3.5" />Remove</Button>}</div>
				</div>
			</Field>
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
          <Template label="Warranty: checking" value={settings.whatsapp_warranty_checking} onChange={(value) => update('whatsapp_warranty_checking', value)} />
          <Template label="Warranty: ready for customer" value={settings.whatsapp_warranty_ready} onChange={(value) => update('whatsapp_warranty_ready', value)} />
          <Template label="Warranty: returned" value={settings.whatsapp_warranty_returned} onChange={(value) => update('whatsapp_warranty_returned', value)} />
        </CardContent>
      </Card>

      <SystemMaintenance />
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div><label className="mb-2 block text-sm font-medium">{label}</label>{children}</div>
}

function Template({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <Field label={label}><Textarea rows={4} value={value} onChange={(event) => onChange(event.target.value)} /></Field>
}
