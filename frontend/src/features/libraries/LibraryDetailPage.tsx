import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Alert, Box, Skeleton, Typography } from '@mui/material'
import { ArrowLeft24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import api from '@/lib/axios'
import MediaLibraryContent from './MediaLibraryContent'
import MusicLibraryContent from './MusicLibraryContent'

interface LibraryInfo {
  Id: string
  Name: string
  CollectionType: string
}

export default function LibraryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [library, setLibrary] = useState<LibraryInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    api.get(`/api/libraries/${encodeURIComponent(id)}`)
      .then((res) => setLibrary(res.data))
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [id, t])

  if (loading) {
    return (
      <>
        <Skeleton variant="text" width={80} height={24} sx={{ mb: 2 }} />
        <Skeleton variant="text" width={250} height={36} sx={{ mb: 3 }} />
        <Skeleton variant="rectangular" height={400} sx={{ borderRadius: 2 }} />
      </>
    )
  }

  if (error || !library) {
    return <Alert severity="error">{error ?? t('common.loadError')}</Alert>
  }

  const isMusic = library.CollectionType?.toLowerCase() === 'music'

  return (
    <>
      <Box component="button" onClick={() => navigate(-1 as any)} style={{ all: 'unset', cursor: 'pointer' }}>
        <Typography variant="body2" color="primary.main" sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 2 }}>
          <ArrowLeft24Regular style={{ fontSize: 18 }} />
          {t('common.back', 'Back')}
        </Typography>
      </Box>
      <PageHeader title={library.Name} />

      {isMusic
        ? <MusicLibraryContent libraryId={library.Id} libraryName={library.Name} />
        : <MediaLibraryContent libraryId={library.Id} libraryName={library.Name} />
      }
    </>
  )
}
