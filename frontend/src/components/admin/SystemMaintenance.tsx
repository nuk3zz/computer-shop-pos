import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, Archive, CheckCircle2, Download, ExternalLink, HardDrive, Network, RefreshCw, RotateCcw, Trash2, Upload } from 'lucide-react'
import apiClient from '@/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { toastHelpers } from '@/lib/toast-helpers'
import { defaultShopSettings, saveShopSettings } from '@/lib/shop-settings'

export function SystemMaintenance() {
  const queryClient = useQueryClient()
  const [networkMode, setNetworkMode] = useState<'local' | 'lan'>('local')
  const [autoBackup, setAutoBackup] = useState(true)
  const [backupTime, setBackupTime] = useState('02:30')
	const [showFreshStart, setShowFreshStart] = useState(false)
	const [freshStartConfirmation, setFreshStartConfirmation] = useState('')

  const { data: info } = useQuery({
    queryKey: ['system-info'],
    queryFn: () => apiClient.getSystemInfo().then((response) => response.data),
  })
  const { data: backups = [] } = useQuery({
    queryKey: ['system-backups'],
    queryFn: () => apiClient.getBackups().then((response) => response.data || []),
    enabled: Boolean(info?.native),
  })

  useEffect(() => {
    if (!info) return
    setNetworkMode(info.network_mode)
    setAutoBackup(info.auto_backup)
    setBackupTime(info.backup_time)
  }, [info])

  const preferences = useMutation({
    mutationFn: () => apiClient.updateSystemPreferences({ network_mode: networkMode, auto_backup: autoBackup, backup_time: backupTime }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['system-info'] })
      toastHelpers.apiSuccess('Server settings', 'Network and backup schedule saved')
    },
    onError: (error) => toastHelpers.apiError('Save server settings', error),
  })
  const createBackup = useMutation({
    mutationFn: () => apiClient.createBackup(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['system-backups'] })
      toastHelpers.apiSuccess('Backup', 'A verified backup archive was created')
    },
    onError: (error) => toastHelpers.apiError('Create backup', error),
  })
  const uploadBackup = useMutation({
    mutationFn: (file: File) => apiClient.uploadBackup(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['system-backups'] })
      toastHelpers.apiSuccess('Backup', 'Backup uploaded and verified')
    },
    onError: (error) => toastHelpers.apiError('Upload backup', error),
  })
  const restoreBackup = useMutation({
    mutationFn: (name: string) => apiClient.restoreBackup(name),
    onSuccess: (response) => toastHelpers.apiSuccess('Restore staged', response.message),
    onError: (error) => toastHelpers.apiError('Restore backup', error),
  })
  const checkUpdates = useMutation({
    mutationFn: () => apiClient.checkForUpdates(),
    onSuccess: (response) => {
      const update = response.data
      if (!update?.update_available) toastHelpers.apiSuccess('Updates', response.message || 'You are up to date')
    },
    onError: (error) => toastHelpers.apiError('Check for updates', error),
  })
	const startFresh = useMutation({
		mutationFn: () => apiClient.startFresh(freshStartConfirmation),
		onSuccess: () => {
			saveShopSettings(defaultShopSettings)
			queryClient.clear()
			window.location.href = '/setup'
		},
		onError: (error) => toastHelpers.apiError('Start fresh', error),
	})

  const confirmRestore = (name: string) => {
    if (window.confirm(`Restore ${name}? A safety backup will be created first. The POS service must then be restarted.`)) {
      restoreBackup.mutate(name)
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Network className="h-5 w-5" />Server Access</CardTitle><CardDescription>Choose whether the POS is limited to this computer or available to trusted devices on the same Wi-Fi/LAN.</CardDescription></CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <Choice selected={networkMode === 'local'} onClick={() => setNetworkMode('local')} title="Only this computer" text="Use localhost on the server." />
            <Choice selected={networkMode === 'lan'} onClick={() => setNetworkMode('lan')} title="Same Wi-Fi / LAN" text="Allow phones and laptops on the trusted local network." />
          </div>
          {info?.addresses && <div className="rounded-md bg-slate-50 p-3 text-sm"><div className="font-medium">Available addresses</div>{info.addresses.map((address) => <div key={address} className="mt-1 break-all font-mono text-xs text-muted-foreground">{address}</div>)}</div>}
          <Button onClick={() => preferences.mutate()} disabled={preferences.isPending}>Save server access</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Archive className="h-5 w-5" />Backup & Restore</CardTitle><CardDescription>{info?.native ? `Data: ${info.data_dir}` : 'This Docker installation continues using its container backup schedule. Native installer controls appear in the standalone edition.'}</CardDescription></CardHeader>
        {info?.native && <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between">
            <div><div className="flex items-center gap-2 font-medium"><Switch checked={autoBackup} onCheckedChange={setAutoBackup} />Automatic daily backup</div><p className="mt-1 text-xs text-muted-foreground">Stored separately in {info.backup_dir}</p></div>
            <div className="flex items-center gap-2"><Input type="time" value={backupTime} onChange={(event) => setBackupTime(event.target.value)} className="w-32" /><Button variant="outline" onClick={() => preferences.mutate()} disabled={preferences.isPending}>Save</Button></div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => createBackup.mutate()} disabled={createBackup.isPending}><HardDrive className="mr-2 h-4 w-4" />{createBackup.isPending ? 'Creating…' : 'Back up now'}</Button>
            <label className="inline-flex cursor-pointer items-center rounded-md border px-4 py-2 text-sm font-medium hover:bg-slate-50"><Upload className="mr-2 h-4 w-4" />Upload backup<input type="file" accept=".urposbackup,.cspbackup" className="hidden" onChange={(event) => { const file = event.target.files?.[0]; if (file) uploadBackup.mutate(file); event.target.value = '' }} /></label>
          </div>
          <div className="divide-y rounded-lg border">
            {backups.length === 0 ? <div className="p-4 text-sm text-muted-foreground">No backup archives yet.</div> : backups.map((backup) => <div key={backup.name} className="flex flex-col gap-2 p-3 sm:flex-row sm:items-center sm:justify-between"><div className="min-w-0"><div className="truncate text-sm font-medium">{backup.name}</div><div className="text-xs text-muted-foreground">{backup.kind} · {formatBytes(backup.size)} · {new Date(backup.created_at).toLocaleString()}</div></div><div className="flex gap-2"><Button variant="outline" size="sm" onClick={() => apiClient.downloadBackup(backup.name)}><Download className="mr-1.5 h-3.5 w-3.5" />Download</Button><Button variant="outline" size="sm" onClick={() => confirmRestore(backup.name)} disabled={restoreBackup.isPending}><RotateCcw className="mr-1.5 h-3.5 w-3.5" />Restore</Button></div></div>)}
          </div>
        </CardContent>}
      </Card>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><RefreshCw className="h-5 w-5" />Software Updates</CardTitle><CardDescription>Installing a newer package replaces the application only. Your Documents data and backups are preserved.</CardDescription></CardHeader>
        <CardContent className="space-y-3">
          <div className="flex flex-wrap items-center gap-3"><Badge variant="outline">Version {info?.version || '…'}</Badge><Button variant="outline" onClick={() => checkUpdates.mutate()} disabled={checkUpdates.isPending}><RefreshCw className={`mr-2 h-4 w-4 ${checkUpdates.isPending ? 'animate-spin' : ''}`} />Check for updates</Button></div>
          {checkUpdates.data?.data?.update_available && <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-sm"><div className="flex items-center gap-2 font-medium text-emerald-800"><CheckCircle2 className="h-4 w-4" />{checkUpdates.data.data.latest_version} is available</div>{checkUpdates.data.data.release_url && <a href={checkUpdates.data.data.release_url} target="_blank" rel="noreferrer" className="mt-2 inline-flex items-center text-emerald-800 underline">Open installer download <ExternalLink className="ml-1 h-3.5 w-3.5" /></a>}</div>}
        </CardContent>
      </Card>

		<Card className="border-red-200">
			<CardHeader><CardTitle className="flex items-center gap-2 text-red-700"><AlertTriangle className="h-5 w-5" />Start Fresh</CardTitle><CardDescription>Remove test data and return to first-time setup. Your current owner login, server settings, and existing backups are retained.</CardDescription></CardHeader>
			<CardContent className="space-y-4">
				<div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">This permanently removes products, categories, stock, sales, repair tickets, clients, staff accounts, and uploaded photos. A safety backup is created automatically in the standalone edition.</div>
				{showFreshStart ? <div className="max-w-md space-y-3"><label className="block text-sm font-medium">Type <span className="font-mono">START FRESH</span> to confirm</label><Input value={freshStartConfirmation} onChange={(event) => setFreshStartConfirmation(event.target.value)} autoComplete="off" /><div className="flex gap-2"><Button variant="destructive" disabled={freshStartConfirmation !== 'START FRESH' || startFresh.isPending} onClick={() => startFresh.mutate()}><Trash2 className="mr-2 h-4 w-4" />{startFresh.isPending ? 'Clearing…' : 'Delete data and start fresh'}</Button><Button variant="outline" onClick={() => { setShowFreshStart(false); setFreshStartConfirmation('') }}>Cancel</Button></div></div> : <Button variant="outline" className="border-red-300 text-red-700 hover:bg-red-50 hover:text-red-800" onClick={() => setShowFreshStart(true)}><Trash2 className="mr-2 h-4 w-4" />Start Fresh…</Button>}
			</CardContent>
		</Card>
    </div>
  )
}

function Choice({ selected, onClick, title, text }: { selected: boolean; onClick: () => void; title: string; text: string }) {
  return <button type="button" onClick={onClick} className={`rounded-lg border p-3 text-left ${selected ? 'border-slate-900 bg-slate-50 ring-1 ring-slate-900' : 'hover:border-slate-400'}`}><div className="font-medium">{title}</div><div className="mt-1 text-xs text-muted-foreground">{text}</div></button>
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
