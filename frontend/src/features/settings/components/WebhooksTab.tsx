import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Button, Typography, IconButton, Tooltip,
  Dialog, DialogTitle, DialogContent, DialogActions, Chip,
  TextField, FormControlLabel, Switch, Checkbox, FormGroup,
  CircularProgress, Alert,
} from '@mui/material'
import {
  Delete24Regular, Add24Regular, Send24Regular, Edit24Regular,
} from '@fluentui/react-icons'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import { format, parseISO } from 'date-fns'
import SkeletonList from '@/shared/components/SkeletonList/SkeletonList'
import ConfirmDialog from '@/shared/components/ConfirmDialog/ConfirmDialog'
import api from '@/lib/axios'

// ─── Types ────────────────────────────────────────────────────────────────────

interface DiscordWebhook {
  id: number
  name: string
  url: string
  bot_username: string
  bot_avatar_url: string
  discord_events: string[]
  enabled: boolean
  last_triggered?: string | null
}

// ─── Event definitions ────────────────────────────────────────────────────────

const EVENTS = [
  { id: 'task_start',      label: 'Task Started',      emoji: '▶️', color: '#3498DB' },
  { id: 'task_complete',   label: 'Task Completed',     emoji: '✅', color: '#57F287' },
  { id: 'task_failed',     label: 'Task Failed',        emoji: '❌', color: '#E74C3C' },
  { id: 'backup_complete', label: 'Backup Completed',   emoji: '💾', color: '#1ABC9C' },
  { id: 'api_key_created', label: 'API Key Created',    emoji: '🔑', color: '#FECB28' },
  { id: 'api_key_deleted', label: 'API Key Deleted',    emoji: '🗑️', color: '#E67E22' },
] as const

type EventId = typeof EVENTS[number]['id']

function getEvent(id: string) {
  return EVENTS.find((e) => e.id === id) ?? EVENTS[0]
}

// ─── Zod schema ───────────────────────────────────────────────────────────────

const schema = z.object({
  name: z.string().min(1, 'Name is required'),
  url: z.string().url('Must be a valid URL').refine(
    (v) => v.includes('discord.com/api/webhooks'),
    { message: 'Must be a Discord webhook URL' },
  ),
  bot_username: z.string().min(1, 'Bot username is required'),
  bot_avatar_url: z.string(),
  discord_events: z.array(z.string()).min(1, 'Select at least one event'),
})
type FormValues = z.infer<typeof schema>

// ─── Discord Preview ──────────────────────────────────────────────────────────

function DiscordPreview({
  botName, avatarUrl, previewEvent,
}: {
  botName: string
  avatarUrl?: string
  previewEvent: EventId
}) {
  const evt = getEvent(previewEvent)
  const displayName = botName?.trim() || 'jellystics_bot'
  const [avatarError, setAvatarError] = useState(false)

  const timeStr = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })

  const sampleData: Record<string, { desc?: string; fields?: { name: string; value: string }[] }> = {
    task_start:      { fields: [{ name: 'Task', value: 'Full Jellyfin Sync' }] },
    task_complete:   { fields: [{ name: 'Task', value: 'Full Jellyfin Sync' }, { name: 'Duration', value: '1234ms' }] },
    task_failed:     { fields: [{ name: 'Task', value: 'Full Jellyfin Sync' }, { name: 'Duration', value: '300ms' }] },
    backup_complete: { fields: [{ name: 'Task', value: 'Backup' }] },
    api_key_created: { fields: [{ name: 'Key Name', value: 'my-api-key' }] },
    api_key_deleted: { fields: [{ name: 'Key Name', value: 'my-api-key' }] },
  }
  const sample = sampleData[previewEvent] ?? {}

  return (
    <Box
      sx={{
        bgcolor: '#313338',
        borderRadius: 2,
        p: 2,
        fontFamily: '"gg sans", "Noto Sans", ui-sans-serif, sans-serif',
        userSelect: 'none',
        minWidth: 0,
      }}
    >
      {/* Header label */}
      <Typography sx={{ color: '#949ba4', fontSize: 11, fontWeight: 600, mb: 1.5, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
        Preview
      </Typography>

      {/* Message row */}
      <Box sx={{ display: 'flex', gap: 1.5 }}>
        {/* Bot avatar — Jellystics logo by default */}
        <Box
          sx={{
            width: 40, height: 40, borderRadius: '50%', flexShrink: 0,
            bgcolor: '#fff',
            overflow: 'hidden',
          }}
        >
          <Box
            component="img"
            src={avatarUrl && !avatarError ? avatarUrl : '/logo.svg'}
            onError={() => setAvatarError(true)}
            sx={{ width: '100%', height: '100%', objectFit: 'contain', p: '6px' }}
          />
        </Box>

        {/* Right side */}
        <Box sx={{ flex: 1, minWidth: 0 }}>
          {/* Name + badge + time */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mb: 0.5, flexWrap: 'wrap' }}>
            <Typography sx={{ color: '#fff', fontWeight: 600, fontSize: 14, lineHeight: 1 }}>
              {displayName}
            </Typography>
            <Box
              sx={{
                bgcolor: '#5865F2', color: '#fff', fontSize: 9, fontWeight: 700,
                px: 0.5, py: 0.15, borderRadius: 0.5, lineHeight: 1.4,
                textTransform: 'uppercase', letterSpacing: '0.03em',
              }}
            >
              APP
            </Box>
            <Typography sx={{ color: '#949ba4', fontSize: 12 }}>
              Today at {timeStr}
            </Typography>
          </Box>

          {/* Embed */}
          <Box
            sx={{
              bgcolor: '#2b2d31',
              borderLeft: `4px solid ${evt.color}`,
              borderRadius: '0 4px 4px 0',
              p: 1.5,
              maxWidth: 420,
              mt: 0.25,
            }}
          >
            {/* Title */}
            <Typography sx={{ color: '#fff', fontWeight: 600, fontSize: 14, mb: sample.fields?.length ? 1 : 0 }}>
              {evt.label}
            </Typography>

            {/* Fields */}
            {sample.fields && sample.fields.length > 0 && (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5, mb: 0.75 }}>
                {sample.fields.map((f) => (
                  <Box key={f.name}>
                    <Typography sx={{ color: '#b5bac1', fontSize: 11, fontWeight: 700 }}>{f.name}</Typography>
                    <Typography sx={{ color: '#dbdee1', fontSize: 13 }}>{f.value}</Typography>
                  </Box>
                ))}
              </Box>
            )}

            {/* Footer with logo */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 1 }}>
              <Box
                component="img"
                src="/logo.svg"
                sx={{ width: 16, height: 16, objectFit: 'contain', opacity: 0.7 }}
              />
              <Typography sx={{ color: '#949ba4', fontSize: 11 }}>
                Jellystics · {new Date().toLocaleDateString()}
              </Typography>
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}

// ─── Webhook card ─────────────────────────────────────────────────────────────

function WebhookCard({
  webhook,
  onEdit,
  onDelete,
  onToggle,
  onTest,
}: {
  webhook: DiscordWebhook
  onEdit: () => void
  onDelete: () => void
  onToggle: () => void
  onTest: () => void
}) {
  const truncateUrl = (url: string) => {
    try {
      const u = new URL(url)
      return u.pathname.slice(0, 40) + (u.pathname.length > 40 ? '…' : '')
    } catch {
      return url.slice(0, 40)
    }
  }

  return (
    <Card variant="outlined" sx={{ mb: 1.5, opacity: webhook.enabled ? 1 : 0.6, transition: 'opacity 0.2s' }}>
      <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
          {/* Info */}
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.25 }}>
              <Typography variant="body2" sx={{ fontWeight: 600 }}>{webhook.name}</Typography>
              <Typography variant="caption" color="text.secondary">·</Typography>
              <Typography variant="caption" color="text.secondary">{webhook.bot_username || 'jellystics_bot'}</Typography>
            </Box>
            <Typography variant="caption" color="text.disabled" sx={{ fontFamily: 'monospace', display: 'block', mb: 0.75 }}>
              discord.com{truncateUrl(webhook.url)}
            </Typography>

            {/* Events */}
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 0.5 }}>
              {(webhook.discord_events ?? []).map((eid) => {
                const ev = getEvent(eid)
                return (
                  <Chip
                    key={eid}
                    label={ev.label}
                    size="small"
                    sx={{
                      height: 20, fontSize: 11,
                    }}
                  />
                )
              })}
            </Box>

            {webhook.last_triggered && (
              <Typography variant="caption" color="text.disabled">
                Last triggered: {format(parseISO(webhook.last_triggered), 'dd/MM/yyyy HH:mm')}
              </Typography>
            )}
          </Box>

          {/* Actions */}
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, flexShrink: 0 }}>
            <Switch
              size="small"
              checked={webhook.enabled}
              onChange={onToggle}
              sx={{ mr: 0.5 }}
            />
            <Tooltip title="Send test">
              <IconButton size="small" onClick={onTest}>
                <Send24Regular style={{ fontSize: 16 }} />
              </IconButton>
            </Tooltip>
            <Tooltip title="Edit">
              <IconButton size="small" onClick={onEdit}>
                <Edit24Regular style={{ fontSize: 16 }} />
              </IconButton>
            </Tooltip>
            <Tooltip title="Delete">
              <IconButton size="small" color="error" onClick={onDelete}>
                <Delete24Regular style={{ fontSize: 16 }} />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>
      </CardContent>
    </Card>
  )
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function WebhooksTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()

  const [webhooks, setWebhooks] = useState<DiscordWebhook[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)
  const [previewEvent, setPreviewEvent] = useState<EventId>('task_complete')

  const {
    register, handleSubmit, control, reset, watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      url: '',
      bot_username: 'jellystics_bot',
      bot_avatar_url: '',
      discord_events: [],
    },
  })

  const watchedValues = watch()
  const selectedEvents = watchedValues.discord_events ?? []

  const load = () => {
    api.get('/webhooks/').then((r) => setWebhooks(r.data ?? [])).catch(() => {}).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const openAdd = () => {
    setEditingId(null)
    reset({ name: '', url: '', bot_username: 'jellystics_bot', bot_avatar_url: '', discord_events: [] })
    setPreviewEvent('task_complete')
    setDialogOpen(true)
  }

  const openEdit = (wh: DiscordWebhook) => {
    setEditingId(wh.id)
    reset({
      name: wh.name,
      url: wh.url,
      bot_username: wh.bot_username || 'jellystics_bot',
      bot_avatar_url: wh.bot_avatar_url || '',
      discord_events: wh.discord_events ?? [],
    })
    const firstEvent = (wh.discord_events ?? [])[0] as EventId | undefined
    setPreviewEvent(firstEvent ?? 'task_complete')
    setDialogOpen(true)
  }

  const onSave = async (data: FormValues) => {
    const payload = {
      name: data.name,
      url: data.url,
      bot_username: data.bot_username || 'jellystics_bot',
      bot_avatar_url: data.bot_avatar_url || '',
      discord_events: data.discord_events,
      trigger_type: 'discord',
      enabled: true,
      method: 'POST',
    }
    try {
      if (editingId !== null) {
        await api.put(`/webhooks/${editingId}`, payload)
        enqueueSnackbar('Webhook updated', { variant: 'success' })
      } else {
        await api.post('/webhooks/', payload)
        enqueueSnackbar(t('settings.webhookAdded'), { variant: 'success' })
      }
      setDialogOpen(false)
      load()
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
  }

  const onDelete = async (id: number) => {
    try {
      await api.delete(`/webhooks/${id}`)
      enqueueSnackbar(t('settings.webhookDeleted'), { variant: 'success' })
      setWebhooks((prev) => prev.filter((w) => w.id !== id))
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setDeleteTarget(null)
    }
  }

  const onToggle = async (wh: DiscordWebhook) => {
    try {
      await api.put(`/webhooks/${wh.id}`, { ...wh, enabled: !wh.enabled })
      setWebhooks((prev) => prev.map((w) => w.id === wh.id ? { ...w, enabled: !w.enabled } : w))
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
  }

  const onTest = async (wh: DiscordWebhook) => {
    try {
      await api.post(`/webhooks/${wh.id}/test`, {})
      enqueueSnackbar('Test notification sent', { variant: 'success' })
    } catch {
      enqueueSnackbar('Test failed', { variant: 'error' })
    }
  }

  return (
    <Box sx={{ maxWidth: 720 }}>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Box>
              <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>Discord Webhooks</Typography>
              <Typography variant="caption" color="text.secondary">
                Send notifications to Discord channels for specific events
              </Typography>
            </Box>
            <Button variant="contained" size="small" startIcon={<Add24Regular />} onClick={openAdd}>
              {t('settings.addWebhook')}
            </Button>
          </Box>

          {loading ? (
            <SkeletonList count={2} height={88} />
          ) : webhooks.length === 0 ? (
            <Box sx={{ py: 4, textAlign: 'center' }}>
              <Typography color="text.secondary" sx={{ mb: 1 }}>
                {t('settings.noWebhooks')}
              </Typography>
              <Typography variant="caption" color="text.disabled">
                Add a Discord webhook URL to get started
              </Typography>
            </Box>
          ) : (
            webhooks.map((wh) => (
              <WebhookCard
                key={wh.id}
                webhook={wh}
                onEdit={() => openEdit(wh)}
                onDelete={() => setDeleteTarget(wh.id)}
                onToggle={() => onToggle(wh)}
                onTest={() => onTest(wh)}
              />
            ))
          )}
        </CardContent>
      </Card>

      {/* Add / Edit Dialog */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          {editingId !== null ? 'Edit Discord Webhook' : 'Add Discord Webhook'}
        </DialogTitle>
        <DialogContent>
          <Box
            component="form"
            id="webhook-form"
            onSubmit={handleSubmit(onSave)}
            noValidate
            sx={{ display: 'flex', gap: 3, pt: 1, flexDirection: { xs: 'column', md: 'row' } }}
          >
            {/* Left — form */}
            <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField
                {...register('name')}
                label="Webhook Name"
                fullWidth
                size="small"
                error={!!errors.name}
                helperText={errors.name?.message}
                autoFocus
              />

              <TextField
                {...register('url')}
                label="Discord Webhook URL"
                fullWidth
                size="small"
                error={!!errors.url}
                helperText={errors.url?.message ?? 'https://discord.com/api/webhooks/…'}
                placeholder="https://discord.com/api/webhooks/..."
              />

              <Box sx={{ display: 'flex', gap: 1.5 }}>
                <TextField
                  {...register('bot_username')}
                  label="Bot Username"
                  fullWidth
                  size="small"
                  error={!!errors.bot_username}
                  helperText={errors.bot_username?.message}
                  placeholder="jellystics_bot"
                />
                <TextField
                  {...register('bot_avatar_url')}
                  label="Bot Avatar URL"
                  fullWidth
                  size="small"
                  error={!!errors.bot_avatar_url}
                  helperText="Optional — leave empty for default"
                />
              </Box>

              {/* Events */}
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.75 }}>
                  Notifications
                </Typography>
                {errors.discord_events && (
                  <Alert severity="error" sx={{ mb: 1, py: 0.25, fontSize: 12 }}>
                    {errors.discord_events.message as string}
                  </Alert>
                )}
                <Controller
                  name="discord_events"
                  control={control}
                  render={({ field }) => (
                    <FormGroup sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0 }}>
                      {EVENTS.map((ev) => {
                        const checked = field.value.includes(ev.id)
                        return (
                          <FormControlLabel
                            key={ev.id}
                            label={<Typography variant="body2">{ev.label}</Typography>}
                            control={
                              <Checkbox
                                size="small"
                                checked={checked}
                                onChange={(e) => {
                                  const next = e.target.checked
                                    ? [...field.value, ev.id]
                                    : field.value.filter((v) => v !== ev.id)
                                  field.onChange(next)
                                  if (e.target.checked) setPreviewEvent(ev.id as EventId)
                                }}
                              />
                            }
                            sx={{ ml: 0, mr: 0 }}
                          />
                        )
                      })}
                    </FormGroup>
                  )}
                />
              </Box>
            </Box>

            {/* Right — preview */}
            <Box sx={{ width: { xs: '100%', md: 320 }, flexShrink: 0 }}>
              <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>
                Preview
              </Typography>

              {/* Event selector for preview */}
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 1.5 }}>
                {EVENTS.map((ev) => (
                  <Chip
                    key={ev.id}
                    label={ev.label}
                    size="small"
                    variant={previewEvent === ev.id ? 'filled' : 'outlined'}
                    onClick={() => setPreviewEvent(ev.id as EventId)}
                    sx={{
                      cursor: 'pointer',
                    }}
                  />
                ))}
              </Box>

              <DiscordPreview
                botName={watchedValues.bot_username || 'jellystics_bot'}
                avatarUrl={watchedValues.bot_avatar_url || undefined}
                previewEvent={
                  selectedEvents.includes(previewEvent)
                    ? previewEvent
                    : (selectedEvents[0] as EventId | undefined) ?? previewEvent
                }
              />

              {selectedEvents.length === 0 && (
                <Typography variant="caption" color="text.disabled" sx={{ mt: 1, display: 'block', textAlign: 'center' }}>
                  Select events above to see how they look
                </Typography>
              )}
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>{t('common.cancel')}</Button>
          <Button
            type="submit"
            form="webhook-form"
            variant="contained"
            disabled={isSubmitting}
            startIcon={isSubmitting ? <CircularProgress size={16} /> : undefined}
          >
            {editingId !== null ? 'Save Changes' : t('common.add')}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete confirm */}
      <ConfirmDialog
        open={deleteTarget !== null}
        title={t('settings.deleteWebhook')}
        description={t('settings.deleteWebhookConfirm')}
        onConfirm={() => deleteTarget !== null && onDelete(deleteTarget)}
        onCancel={() => setDeleteTarget(null)}
        dangerous
      />
    </Box>
  )
}
