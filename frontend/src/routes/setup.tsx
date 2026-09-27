import { useEffect, useState } from 'react'
import { createFileRoute, Navigate, useNavigate } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Check, ChevronLeft, ChevronRight, Eye, EyeOff, ImagePlus, Laptop, Network, ShieldCheck, UserRound } from 'lucide-react'
import apiClient from '@/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { getMediaUrl } from '@/lib/media'
import { defaultShopSettings, saveShopSettings } from '@/lib/shop-settings'
import { toastHelpers } from '@/lib/toast-helpers'
import type { InitialSetupInput } from '@/types'

export const Route = createFileRoute('/setup')({ component: SetupPage })

function SetupPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [step, setStep] = useState(1)
  const [companyName, setCompanyName] = useState('')
  const [logoFile, setLogoFile] = useState<File | null>(null)
  const [logoPreview, setLogoPreview] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [networkMode, setNetworkMode] = useState<'local' | 'lan'>('local')
  const [formError, setFormError] = useState('')

  const { data: profile, isLoading } = useQuery({
    queryKey: ['setup-status'],
    queryFn: async () => {
      const response = await apiClient.getSetupStatus()
      if (!response.data) throw new Error('Setup status is missing')
      return response.data
    },
  })

  useEffect(() => {
    if (!profile || companyName) return
    setCompanyName(profile.company_name === 'Computer Shop POS' ? '' : profile.company_name)
    setNetworkMode(profile.network_mode || 'local')
  }, [companyName, profile])

  const completeSetup = useMutation({
    mutationFn: async () => {
      let logoURL = profile?.logo_url
      if (logoFile) {
        const upload = apiClient.isAuthenticated()
          ? await apiClient.uploadProductImage(logoFile)
          : await apiClient.uploadSetupLogo(logoFile)
        logoURL = upload.data?.url
      }
      const input: InitialSetupInput = {
        company_name: companyName.trim(),
        logo_url: logoURL,
        first_name: firstName.trim(),
        last_name: lastName.trim() || undefined,
        username: username.trim(),
        email: email.trim() || undefined,
        password,
        network_mode: networkMode,
      }
      return apiClient.completeInitialSetup(input)
    },
    onSuccess: () => {
      saveShopSettings({ ...defaultShopSettings, shop_name: companyName.trim() })
      queryClient.clear()
      localStorage.removeItem('pos_token')
      localStorage.removeItem('pos_user')
      toastHelpers.apiSuccess('Setup complete', 'Sign in with your new owner username and password')
      navigate({ to: '/login' })
    },
    onError: (error) => toastHelpers.apiError('Complete setup', error),
  })

  const validateBusinessStep = () => {
    if (!companyName.trim()) {
      setFormError('Enter your company name.')
      return
    }
    setFormError('')
    setStep(2)
  }

  const submitOwner = () => {
    if (!firstName.trim() || !/^[A-Za-z0-9._-]{3,50}$/.test(username.trim())) {
      setFormError('Enter your name and a username with at least 3 letters or numbers.')
      return
    }
    if (password.length < 8) {
      setFormError('Password must contain at least 8 characters.')
      return
    }
    if (password !== confirmPassword) {
      setFormError('The passwords do not match.')
      return
    }
    setFormError('')
    completeSetup.mutate()
  }

  if (isLoading) return <div className="flex min-h-screen items-center justify-center text-muted-foreground">Loading setup…</div>
  if (profile?.setup_completed) return <Navigate to={apiClient.isAuthenticated() ? '/admin/dashboard' : '/login'} />

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50 p-4">
      <Card className="w-full max-w-2xl shadow-lg">
        <CardHeader className="space-y-3 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-slate-900 text-white">{step === 1 ? <Building2 className="h-6 w-6" /> : <UserRound className="h-6 w-6" />}</div>
          <div><CardTitle className="text-2xl">{step === 1 ? 'Set up your shop' : 'Create the owner account'}</CardTitle><CardDescription className="mt-1">Step {step} of 2 · {step === 1 ? 'Business identity and network access' : 'Choose your private sign-in details'}</CardDescription></div>
          <div className="mx-auto flex w-36 gap-2"><div className="h-1.5 flex-1 rounded-full bg-slate-900" /><div className={`h-1.5 flex-1 rounded-full ${step === 2 ? 'bg-slate-900' : 'bg-slate-200'}`} /></div>
        </CardHeader>

        <CardContent className="space-y-6">
          {step === 1 ? (
            <>
              <Field label="Company name"><Input value={companyName} onChange={(event) => setCompanyName(event.target.value)} placeholder="Your computer or phone repair shop" maxLength={150} autoFocus /></Field>
              <div className="space-y-2">
                <label className="text-sm font-medium">Company logo <span className="font-normal text-muted-foreground">(optional)</span></label>
                <div className="flex items-center gap-4">
                  <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-xl border bg-slate-100">{logoPreview || profile?.logo_url ? <img src={logoPreview || getMediaUrl(profile?.logo_url)} alt="Logo preview" className="h-full w-full object-contain" /> : <ImagePlus className="h-7 w-7 text-slate-400" />}</div>
                  <Input type="file" accept="image/jpeg,image/png,image/webp,image/gif" onChange={(event) => {
                    const file = event.target.files?.[0]
                    if (!file) return
                    if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type) || file.size > 5 * 1024 * 1024) {
                      setFormError('Use a JPG, PNG, WebP, or GIF no larger than 5 MB.')
                      return
                    }
                    setLogoFile(file)
                    setLogoPreview(URL.createObjectURL(file))
                    setFormError('')
                  }} />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Where can the POS be opened?</label>
                <div className="grid gap-3 sm:grid-cols-2">
                  <NetworkChoice selected={networkMode === 'local'} onClick={() => setNetworkMode('local')} icon={<Laptop className="h-5 w-5" />} title="Only this computer" description="Use localhost. Best while testing." />
                  <NetworkChoice selected={networkMode === 'lan'} onClick={() => setNetworkMode('lan')} icon={<Network className="h-5 w-5" />} title="Same Wi-Fi / LAN" description="Open it from your laptop or phone using the server URL." />
                </div>
                <p className="text-xs text-muted-foreground">LAN mode is for a trusted home or shop network only. It does not expose the POS to the internet.</p>
              </div>
            </>
          ) : (
            <>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Your name"><Input value={firstName} onChange={(event) => setFirstName(event.target.value)} placeholder="First name" autoFocus /></Field>
                <Field label="Last name (optional)"><Input value={lastName} onChange={(event) => setLastName(event.target.value)} placeholder="Last name" /></Field>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Username"><Input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="Example: isuru" autoComplete="username" /></Field>
                <Field label="Email (optional)"><Input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@example.com" autoComplete="email" /></Field>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Password"><div className="relative"><Input type={showPassword ? 'text' : 'password'} value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" className="pr-10" /><button type="button" onClick={() => setShowPassword((shown) => !shown)} className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground" aria-label="Show or hide password">{showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}</button></div></Field>
                <Field label="Confirm password"><Input type={showPassword ? 'text' : 'password'} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} autoComplete="new-password" /></Field>
              </div>
              <div className="rounded-lg border bg-slate-50 p-3 text-sm text-slate-600"><div className="flex items-center gap-2 font-medium text-slate-900"><ShieldCheck className="h-4 w-4" />No default administrator login</div><p className="mt-1 text-xs">These details become the only owner account. Your password is stored as a secure hash.</p></div>
            </>
          )}

          {formError && <p className="rounded-md bg-red-50 p-3 text-sm text-red-700">{formError}</p>}

          <div className="flex gap-3">
            {step === 2 && <Button variant="outline" className="flex-1" onClick={() => { setStep(1); setFormError('') }}><ChevronLeft className="mr-2 h-4 w-4" />Back</Button>}
            <Button className="flex-1" size="lg" disabled={completeSetup.isPending} onClick={step === 1 ? validateBusinessStep : submitOwner}>
              {step === 1 ? <><span>Continue</span><ChevronRight className="ml-2 h-4 w-4" /></> : <><Check className="mr-2 h-4 w-4" />{completeSetup.isPending ? 'Creating owner…' : 'Finish setup'}</>}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="space-y-2"><label className="text-sm font-medium">{label}</label>{children}</div>
}

function NetworkChoice({ selected, onClick, icon, title, description }: { selected: boolean; onClick: () => void; icon: React.ReactNode; title: string; description: string }) {
  return <button type="button" onClick={onClick} className={`rounded-lg border p-3 text-left transition-colors ${selected ? 'border-slate-900 bg-slate-50 ring-1 ring-slate-900' : 'hover:border-slate-400'}`}><div className="flex items-center gap-2 font-medium">{icon}{title}</div><p className="mt-1 text-xs text-muted-foreground">{description}</p></button>
}
