import { useEffect, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import apiClient from '@/api/client'
import { getMediaUrl } from '@/lib/media'
import { toastHelpers } from '@/lib/toast-helpers'
import type { Customer, CustomerInput } from '@/types'
import { Trash2, UserRound, X } from 'lucide-react'

interface ClientFormProps {
  customer?: Customer
  onSuccess: () => void
  onCancel: () => void
}

export function ClientForm({ customer, onSuccess, onCancel }: ClientFormProps) {
  const [name, setName] = useState(customer?.name || '')
  const [phone, setPhone] = useState(customer?.phone || '')
  const [email, setEmail] = useState(customer?.email || '')
  const [notes, setNotes] = useState(customer?.notes || '')
  const [imageUrl, setImageUrl] = useState(customer?.image_url || '')
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState('')
  const [imageError, setImageError] = useState('')
  const queryClient = useQueryClient()

  useEffect(() => () => {
    if (imagePreview.startsWith('blob:')) URL.revokeObjectURL(imagePreview)
  }, [imagePreview])

  const saveClient = useMutation({
    mutationFn: async () => {
      let savedImageUrl = imageUrl
      if (imageFile) {
        const upload = await apiClient.uploadProductImage(imageFile)
        savedImageUrl = upload.data?.url || ''
      }
      const payload: CustomerInput = {
        name: name.trim(),
        phone: phone.trim(),
        email: email.trim() || undefined,
        notes: notes.trim() || undefined,
        image_url: savedImageUrl || undefined,
      }
      return customer ? apiClient.updateCustomer(customer.id, payload) : apiClient.createCustomer(payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] })
      queryClient.invalidateQueries({ queryKey: ['customer-suggestions'] })
      toastHelpers.apiSuccess(customer ? 'Update' : 'Create', `Client "${name.trim()}"`)
      onSuccess()
    },
    onError: (error) => toastHelpers.apiError(customer ? 'Update client' : 'Create client', error),
  })

  const handleImageChange = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return
    const acceptedTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
    if (!acceptedTypes.includes(file.type)) {
      setImageError('Use a JPG, PNG, WebP, or GIF image.')
      event.target.value = ''
      return
    }
    if (file.size > 5 * 1024 * 1024) {
      setImageError('Image must be 5 MB or smaller.')
      event.target.value = ''
      return
    }
    setImageError('')
    setImageFile(file)
    setImagePreview(URL.createObjectURL(file))
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    if (!name.trim() || !phone.trim()) {
      toastHelpers.validationError('Client name and WhatsApp/contact number are required.')
      return
    }
    saveClient.mutate()
  }

  return (
    <Card className="mx-auto max-w-2xl">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{customer ? 'Edit Client' : 'Add Client'}</CardTitle>
        <Button variant="ghost" size="icon" onClick={onCancel}><X className="h-4 w-4" /></Button>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="space-y-5">
          <div className="space-y-2">
            <label className="text-sm font-medium">Client photo (optional)</label>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
              <div className="flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-full border bg-muted">
                {imagePreview || imageUrl ? (
                  <img src={imagePreview || getMediaUrl(imageUrl)} alt="Client preview" className="h-full w-full object-cover" />
                ) : (
                  <UserRound className="h-10 w-10 text-muted-foreground" />
                )}
              </div>
              <div className="flex-1 space-y-2">
                <Input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={handleImageChange} />
                <p className="text-xs text-muted-foreground">JPG, PNG, WebP, or GIF; maximum 5 MB.</p>
                {(imagePreview || imageUrl) && (
                  <Button type="button" variant="outline" size="sm" onClick={() => { setImageFile(null); setImagePreview(''); setImageUrl('') }}>
                    <Trash2 className="mr-2 h-4 w-4" /> Remove photo
                  </Button>
                )}
                {imageError && <p className="text-sm text-destructive">{imageError}</p>}
              </div>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="client-name" className="text-sm font-medium">Client name *</label>
              <Input id="client-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="Example: Tony" />
            </div>
            <div className="space-y-2">
              <label htmlFor="client-phone" className="text-sm font-medium">WhatsApp / contact number *</label>
              <Input id="client-phone" value={phone} onChange={(event) => setPhone(event.target.value)} placeholder="Example: 0771234567" inputMode="tel" />
            </div>
          </div>

          <div className="space-y-2">
            <label htmlFor="client-email" className="text-sm font-medium">Email (optional)</label>
            <Input id="client-email" type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="client@example.com" />
          </div>

          <div className="space-y-2">
            <label htmlFor="client-notes" className="text-sm font-medium">Client notes (optional)</label>
            <Textarea id="client-notes" value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Preferences, device details, or useful reminders..." rows={4} />
          </div>

          <div className="flex gap-3 pt-2">
            <Button type="submit" className="flex-1" disabled={saveClient.isPending}>
              {saveClient.isPending ? 'Saving...' : customer ? 'Update Client' : 'Add Client'}
            </Button>
            <Button type="button" variant="outline" className="flex-1" onClick={onCancel}>Cancel</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
