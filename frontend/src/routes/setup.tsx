import { useEffect, useState } from 'react'
import { createFileRoute, Navigate, useNavigate } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Check, ImagePlus, ShieldCheck } from 'lucide-react'
import apiClient from '@/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { getMediaUrl } from '@/lib/media'
import { loadShopSettings, saveShopSettings } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'

export const Route = createFileRoute('/setup')({ component: SetupPage })

function SetupPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [companyName, setCompanyName] = useState('')
  const [logoFile, setLogoFile] = useState<File | null>(null)
  const [logoPreview, setLogoPreview] = useState('')
  const [imageError, setImageError] = useState('')
  const authenticated = apiClient.isAuthenticated()

  const { data: profile, isLoading } = useQuery({
    queryKey: ['shop-profile'],
    queryFn: async () => {
      const response = await apiClient.getShopProfile()
      if (!response.data) throw new Error('Shop profile is missing')
      return response.data
    },
    enabled: authenticated,
  })

  useEffect(() => {
    if (profile && !companyName) setCompanyName(profile.company_name === 'Computer Shop POS' ? '' : profile.company_name)
  }, [companyName, profile])

  const completeSetup = useMutation({
    mutationFn: async () => {
      let logoURL = profile?.logo_url
      if (logoFile) {
        const upload = await apiClient.uploadProductImage(logoFile)
        logoURL = upload.data?.url
      }
      const response = await apiClient.updateShopProfile({ company_name: companyName.trim(), logo_url: logoURL })
      if (!response.data) throw new Error('The saved shop profile was not returned')
      return response.data
    },
    onSuccess: (savedProfile) => {
      saveShopSettings({ ...loadShopSettings(), shop_name: savedProfile.company_name })
      queryClient.setQueryData(['shop-profile'], savedProfile)
      toastHelpers.apiSuccess('Setup complete', `${savedProfile.company_name} is ready`)
      navigate({ to: '/admin/dashboard' })
    },
    onError: (error) => toastHelpers.apiError('Complete setup', error),
  })

  if (!authenticated) return <Navigate to="/login" />
  if (isLoading) return <div className="flex min-h-screen items-center justify-center text-muted-foreground">Loading setup…</div>
  if (profile?.setup_completed) return <Navigate to="/admin/dashboard" />

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50 p-4">
      <Card className="w-full max-w-xl shadow-lg">
        <CardHeader className="space-y-3 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-slate-900 text-white"><Building2 className="h-6 w-6" /></div>
          <div><CardTitle className="text-2xl">Set up your shop</CardTitle><CardDescription className="mt-1">Add your business identity. It will be saved permanently on this POS server.</CardDescription></div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-2">
            <label htmlFor="company-name" className="text-sm font-medium">Company name</label>
            <Input id="company-name" value={companyName} onChange={(event) => setCompanyName(event.target.value)} placeholder="Your computer shop name" maxLength={150} autoFocus />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium">Company logo <span className="font-normal text-muted-foreground">(optional)</span></label>
            <div className="flex items-center gap-4">
              <div className="flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-xl border bg-slate-100">
                {logoPreview || profile?.logo_url ? <img src={logoPreview || getMediaUrl(profile?.logo_url)} alt="Logo preview" className="h-full w-full object-contain" /> : <ImagePlus className="h-8 w-8 text-slate-400" />}
              </div>
              <div className="flex-1 space-y-2">
                <Input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(event) => {
                  const file = event.target.files?.[0]
                  if (!file) return
                  if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type) || file.size > 5 * 1024 * 1024) {
                    setImageError('Use a JPG, PNG, WebP, or GIF no larger than 5 MB.')
                    return
                  }
                  if (logoPreview.startsWith('blob:')) URL.revokeObjectURL(logoPreview)
                  setImageError('')
                  setLogoFile(file)
                  setLogoPreview(URL.createObjectURL(file))
                }} />
                <p className="text-xs text-muted-foreground">A square PNG or WebP works best.</p>
                {imageError && <p className="text-sm text-destructive">{imageError}</p>}
              </div>
            </div>
          </div>
          <div className="rounded-lg border bg-slate-50 p-3 text-sm text-slate-600">
            <div className="flex items-center gap-2 font-medium text-slate-900"><ShieldCheck className="h-4 w-4" />Stored on your self-hosted server</div>
            <p className="mt-1 text-xs">Your company details remain after restarts and are included in the normal database and uploads backups.</p>
          </div>
          <Button className="w-full" size="lg" disabled={!companyName.trim() || completeSetup.isPending} onClick={() => completeSetup.mutate()}>
            <Check className="mr-2 h-4 w-4" />{completeSetup.isPending ? 'Saving setup…' : 'Finish setup'}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
