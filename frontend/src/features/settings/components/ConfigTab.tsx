import { useState, useEffect, useRef } from 'react'
import {
  Box, Button, TextField, Typography, Card, CardContent,
  CircularProgress, FormControl, InputLabel, Select, MenuItem,
  Dialog, DialogTitle, DialogContent, DialogActions, Tooltip, IconButton,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { Add24Regular, Delete24Regular } from '@fluentui/react-icons'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import { useSnackbar } from 'notistack'
import api from '@/lib/axios'
import { usePalette } from '@/lib/PaletteContext'
import {
  type ColorPalette,
  getAllPalettes,
  getCustomPalettes,
  saveCustomPalettes,
} from '@/lib/palette'
import { getCustomFavicon, uploadFavicon, resetFavicon } from '@/lib/favicon'
import { useLogo } from '@/lib/FaviconContext'

const schema = z.object({
  JellyfinUrl: z.string().url(),
  ApiKey: z.string(),
  AppUrl: z.string(),
})

type FormData = z.infer<typeof schema>


const languages = [
  { code: 'en', displayName: 'English' },
  { code: 'fr', displayName: 'Français' },
  { code: 'es', displayName: 'Español' },
]

function SiteIconSection() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [preview, setPreview] = useState<string | null>(getCustomFavicon)
  const inputRef = useRef<HTMLInputElement>(null)
  const { refreshLogo } = useLogo()

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    try {
      await uploadFavicon(file)
      setPreview(getCustomFavicon())
      refreshLogo()
      enqueueSnackbar(t('settings.iconSaved'), { variant: 'success' })
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
    e.target.value = ''
  }

  const handleReset = () => {
    resetFavicon()
    setPreview(null)
    refreshLogo()
  }

  return (
    <Card sx={{ mb: 3 }}>
      <CardContent>
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 2.5 }}>{t('settings.siteIcon')}</Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 3 }}>
          <Box
            sx={{
              width: 64,
              height: 64,
              borderRadius: 2,
              border: '1px solid',
              borderColor: 'divider',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'background.default',
              flexShrink: 0,
              overflow: 'hidden',
            }}
          >
            <Box
              component="img"
              src={preview ?? '/logo.svg'}
              alt="favicon"
              sx={{ width: 40, height: 40, objectFit: 'contain' }}
            />
          </Box>

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button
                variant="outlined"
                size="small"
                onClick={() => inputRef.current?.click()}
                sx={{ borderRadius: 2 }}
              >
                {t('settings.uploadIcon')}
              </Button>
              {preview && (
                <Button
                  variant="text"
                  size="small"
                  color="error"
                  onClick={handleReset}
                  sx={{ borderRadius: 2 }}
                >
                  {t('common.reset')}
                </Button>
              )}
            </Box>
            <Typography variant="caption" color="text.secondary">
              {t('settings.iconHint')}
            </Typography>
          </Box>
        </Box>

        <input
          ref={inputRef}
          type="file"
          accept="image/*"
          onChange={handleFile}
          style={{ display: 'none' }}
        />
      </CardContent>
    </Card>
  )
}

interface PaletteSwatchProps {
  palette: ColorPalette
  selected: boolean
  isDark: boolean
  bgColor: string
  onSelect: () => void
  onDelete?: () => void
}

function PaletteSwatch({ palette: p, selected, isDark, bgColor, onSelect, onDelete }: PaletteSwatchProps) {
  const [hovered, setHovered] = useState(false)
  const color = isDark ? p.dark : p.light

  return (
    <Box
      sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 0.75 }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Box sx={{ position: 'relative' }}>
        <Tooltip title={p.label} placement="top">
          <Box
            onClick={onSelect}
            sx={{
              width: 34,
              height: 34,
              borderRadius: '50%',
              background: `linear-gradient(135deg, ${p.light} 50%, ${p.dark} 50%)`,
              cursor: 'pointer',
              boxShadow: selected
                ? `0 0 0 2px ${bgColor}, 0 0 0 4px ${color}`
                : 'none',
              transition: 'transform 0.15s, box-shadow 0.15s',
              '&:hover': { transform: 'scale(1.12)' },
            }}
          />
        </Tooltip>
        {onDelete && (
          <IconButton
            size="small"
            onClick={onDelete}
            sx={{
              position: 'absolute',
              top: -6,
              right: -6,
              width: 16,
              height: 16,
              p: 0,
              bgcolor: 'error.main',
              color: '#fff',
              opacity: hovered ? 1 : 0,
              transition: 'opacity 0.15s',
              '&:hover': { bgcolor: 'error.dark' },
            }}
          >
            <Delete24Regular style={{ fontSize: 10 }} />
          </IconButton>
        )}
      </Box>
      <Typography
        variant="caption"
        color={selected ? 'primary' : 'text.secondary'}
        sx={{ fontWeight: selected ? 600 : 400, lineHeight: 1 }}
      >
        {p.label}
      </Typography>
    </Box>
  )
}

function ColorInput({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  const [hexText, setHexText] = useState(value)

  // Keep local text in sync when parent value changes (e.g. picker)
  useEffect(() => setHexText(value), [value])

  const handleText = (raw: string) => {
    setHexText(raw)
    const normalized = raw.startsWith('#') ? raw : `#${raw}`
    if (/^#[0-9a-fA-F]{6}$/.test(normalized)) {
      onChange(normalized.toLowerCase())
    }
  }

  return (
    <Box sx={{ flex: 1 }}>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.75, fontWeight: 600 }}>
        {label}
      </Typography>
      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
        <Box
          component="input"
          type="color"
          value={value}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
          sx={{
            width: 36,
            height: 36,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1.5,
            cursor: 'pointer',
            p: '3px',
            bgcolor: 'transparent',
            flexShrink: 0,
          }}
        />
        <TextField
          size="small"
          value={hexText}
          onChange={(e) => handleText(e.target.value)}
          slotProps={{ htmlInput: { maxLength: 7, spellCheck: false, style: { fontFamily: 'monospace', fontSize: 13 } } }}
          placeholder="#000000"
          sx={{ flex: 1 }}
          error={hexText.length > 0 && !/^#?[0-9a-fA-F]{0,6}$/.test(hexText)}
        />
      </Box>
    </Box>
  )
}

function PaletteSection() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const theme = useTheme()
  const { paletteId, setPaletteId } = usePalette()
  const [palettes, setPalettes] = useState<ColorPalette[]>(getAllPalettes)
  const [addOpen, setAddOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newLight, setNewLight] = useState('#6366f1')
  const [newDark, setNewDark] = useState('#818cf8')
  const isDark = theme.palette.mode === 'dark'

  const handleAdd = () => {
    if (!newName.trim()) return
    const id = `custom-${Date.now()}`
    const custom = getCustomPalettes()
    const updated = [...custom, { id, label: newName.trim(), light: newLight, dark: newDark }]
    saveCustomPalettes(updated)
    setPalettes(getAllPalettes())
    setPaletteId(id)
    setAddOpen(false)
    setNewName('')
    enqueueSnackbar(t('settings.paletteAdded'), { variant: 'success' })
  }

  const handleDelete = (id: string) => {
    const updated = getCustomPalettes().filter((p) => p.id !== id)
    saveCustomPalettes(updated)
    setPalettes(getAllPalettes())
    if (paletteId === id) setPaletteId('slate')
    enqueueSnackbar(t('settings.paletteDeleted'), { variant: 'success' })
  }

  return (
    <Card sx={{ mb: 3 }}>
      <CardContent>
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 2.5 }}>{t('settings.accentColor')}</Typography>

        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5, alignItems: 'center' }}>
          {palettes.map((p) => (
            <PaletteSwatch
              key={p.id}
              palette={p}
              selected={paletteId === p.id}
              isDark={isDark}
              bgColor={theme.palette.background.paper}
              onSelect={() => setPaletteId(p.id)}
              onDelete={!p.builtIn ? () => handleDelete(p.id) : undefined}
            />
          ))}

          <Tooltip title={t('settings.addPalette')} placement="top">
            <Box
              onClick={() => setAddOpen(true)}
              sx={{
                width: 34,
                height: 34,
                borderRadius: '50%',
                border: '2px dashed',
                borderColor: 'divider',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                cursor: 'pointer',
                color: 'text.disabled',
                transition: 'border-color 0.15s, color 0.15s, transform 0.15s',
                '&:hover': { borderColor: 'primary.main', color: 'primary.main', transform: 'scale(1.12)' },
              }}
            >
              <Add24Regular style={{ fontSize: 16 }} />
            </Box>
          </Tooltip>
        </Box>
      </CardContent>

      <Dialog open={addOpen} onClose={() => setAddOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>{t('settings.addPalette')}</DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 0.5, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              label={t('settings.paletteName')}
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              fullWidth
              size="small"
              autoFocus
            />
            <Box sx={{ display: 'flex', gap: 2 }}>
              <ColorInput label={t('settings.paletteLight')} value={newLight} onChange={setNewLight} />
              <ColorInput label={t('settings.paletteDark')} value={newDark} onChange={setNewDark} />
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
              <Typography variant="caption" color="text.secondary">{t('common.preview')}:</Typography>
              <Box sx={{
                width: 34, height: 34, borderRadius: '50%',
                background: `linear-gradient(135deg, ${newLight} 50%, ${newDark} 50%)`,
                border: '1px solid', borderColor: 'divider',
              }} />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setAddOpen(false)}>{t('common.cancel')}</Button>
          <Button variant="contained" onClick={handleAdd} disabled={!newName.trim()}>
            {t('common.add')}
          </Button>
        </DialogActions>
      </Dialog>
    </Card>
  )
}

export default function ConfigTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [lang, setLang] = useState(i18next.language)

  const {
    control,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({ resolver: zodResolver(schema) })

  useEffect(() => {
    api.get('/api/getconfig').then((r) => {
      const data = r.data
      reset({
        JellyfinUrl: data.JF_HOST ?? '',
        ApiKey: '',
        AppUrl: data.app_url ?? '',
      })
    }).catch(() => {})
  }, [reset])

  const onSubmit = async (data: FormData) => {
    try {
      await api.post('/api/setconfig', { JF_HOST: data.JellyfinUrl, JF_API_KEY: data.ApiKey, app_url: data.AppUrl })
      enqueueSnackbar(t('settings.configSaved'), { variant: 'success' })
    } catch {
      enqueueSnackbar(t('common.saveError'), { variant: 'error' })
    }
  }

  const handleLanguageChange = (code: string) => {
    setLang(code)
    i18next.changeLanguage(code)
  }

  return (
    <Box sx={{ maxWidth: 640 }}>
      <Card sx={{ mb: 3, borderRadius: 3, border: '1px solid', borderColor: 'divider', boxShadow: 'none' }}>
        <CardContent sx={{ p: 3 }}>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 2.5 }}>{t('settings.jellyfinConfig')}</Typography>
          <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
            <Controller
              name="JellyfinUrl"
              control={control}
              render={({ field }) => (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.75, fontWeight: 600 }}>
                    {t('setup.jellyfinUrl')}
                  </Typography>
                  <TextField
                    {...field}
                    placeholder={t('placeholder.jellyfinUrl')}
                    fullWidth
                    size="small"
                    error={!!errors.JellyfinUrl}
                    helperText={errors.JellyfinUrl?.message}
                  />
                </Box>
              )}
            />
            <Controller
              name="ApiKey"
              control={control}
              render={({ field }) => (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.75, fontWeight: 600 }}>
                    {t('setup.apiKey')}
                  </Typography>
                  <TextField
                    {...field}
                    placeholder={t('setup.apiKey')}
                    fullWidth
                    size="small"
                    error={!!errors.ApiKey}
                    helperText={errors.ApiKey?.message}
                  />
                </Box>
              )}
            />
            <Controller
              name="AppUrl"
              control={control}
              render={({ field }) => (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.75, fontWeight: 600 }}>
                    Public Jellystics URL
                  </Typography>
                  <TextField
                    {...field}
                    placeholder="https://jellystics.example.com"
                    fullWidth
                    size="small"
                    helperText="Used as the default bot avatar in Discord notifications"
                  />
                </Box>
              )}
            />
            <Button type="submit" variant="contained" disabled={isSubmitting} sx={{ px: 3, borderRadius: 2 }}>
              {isSubmitting ? <CircularProgress size={18} /> : t('common.save')}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('frame.language')}</Typography>
          <FormControl size="small" sx={{ minWidth: 200 }}>
            <InputLabel>{t('frame.language')}</InputLabel>
            <Select value={lang} label={t('frame.language')} onChange={(e) => handleLanguageChange(e.target.value)}>
              {languages.map((l) => (
                <MenuItem key={l.code} value={l.code}>{l.displayName}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </CardContent>
      </Card>

      <PaletteSection />
      <SiteIconSection />

    </Box>
  )
}
