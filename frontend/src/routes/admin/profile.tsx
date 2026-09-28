import { useEffect, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ImagePlus, Save, Trash2, UserRound } from 'lucide-react'
import apiClient from '@/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { getMediaUrl } from '@/lib/media'
import { toastHelpers } from '@/lib/toast-helpers'

export const Route = createFileRoute('/admin/profile')({ component: ProfilePage })

function ProfilePage() {
  const queryClient = useQueryClient()
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [imageURL, setImageURL] = useState<string | undefined>()
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState('')

  const { data: user, isLoading } = useQuery({
    queryKey: ['current-user'],
    queryFn: () => apiClient.getCurrentUser().then((response) => response.data),
  })

  useEffect(() => {
    if (!user) return
    setFirstName(user.first_name)
    setLastName(user.last_name)
    setUsername(user.username)
    setEmail(user.email)
    setImageURL(user.profile_image_url)
  }, [user])

  const save = useMutation({
    mutationFn: async () => {
      let nextImageURL = imageURL
      if (imageFile) nextImageURL = (await apiClient.uploadProductImage(imageFile)).data?.url
      return apiClient.updateCurrentUser({
        first_name: firstName.trim(), last_name: lastName.trim(), username: username.trim(),
        email: email.trim(), profile_image_url: nextImageURL,
      })
    },
    onSuccess: (response) => {
      if (!response.data) return
      localStorage.setItem('pos_user', JSON.stringify(response.data))
      queryClient.setQueryData(['current-user'], response.data)
      window.dispatchEvent(new CustomEvent('user-profile-changed', { detail: response.data }))
      setImageFile(null)
      setImagePreview('')
      toastHelpers.apiSuccess('Profile', 'Your profile was updated')
    },
    onError: (error) => toastHelpers.apiError('Update profile', error),
  })

  if (isLoading) return <div className="p-6 text-muted-foreground">Loading profile…</div>

  return (
    <div className="max-w-3xl space-y-6 p-6">
      <div><h1 className="text-3xl font-bold tracking-tight">Your Profile</h1><p className="text-muted-foreground">Change the owner name, sign-in username, email, and profile photo.</p></div>
      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><UserRound className="h-5 w-5" />Owner Details</CardTitle><CardDescription>These details appear in the account menu. Your password is not changed here.</CardDescription></CardHeader>
        <CardContent className="space-y-5">
          <div className="flex items-center gap-4">
            <div className="flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-full border bg-muted">{imagePreview || imageURL ? <img src={imagePreview || getMediaUrl(imageURL)} alt="Profile preview" className="h-full w-full object-cover" /> : <ImagePlus className="h-7 w-7 text-muted-foreground" />}</div>
            <div className="min-w-0 flex-1 space-y-2"><Input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(event) => { const file = event.target.files?.[0]; if (!file) return; setImageFile(file); setImagePreview(URL.createObjectURL(file)) }} />{(imageURL || imagePreview) && <Button type="button" size="sm" variant="outline" onClick={() => { setImageURL(undefined); setImageFile(null); setImagePreview('') }}><Trash2 className="mr-1.5 h-3.5 w-3.5" />Remove photo</Button>}</div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2"><Field label="First Name"><Input value={firstName} onChange={(event) => setFirstName(event.target.value)} maxLength={50} /></Field><Field label="Last Name"><Input value={lastName} onChange={(event) => setLastName(event.target.value)} maxLength={50} /></Field></div>
          <div className="grid gap-4 sm:grid-cols-2"><Field label="Username"><Input value={username} onChange={(event) => setUsername(event.target.value)} maxLength={50} /></Field><Field label="Email"><Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} maxLength={100} /></Field></div>
          <Button onClick={() => save.mutate()} disabled={save.isPending || !firstName.trim() || !username.trim() || !email.trim()}><Save className="mr-2 h-4 w-4" />{save.isPending ? 'Saving…' : 'Save profile'}</Button>
        </CardContent>
      </Card>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="space-y-2"><label className="text-sm font-medium">{label}</label>{children}</div>
}
