import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Alert, Avatar, Box, Card, CardContent, Chip, Grid, Skeleton, Typography,
} from '@mui/material'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { format, parseISO } from 'date-fns'
import {
  ArrowLeft24Regular, Clock24Regular, People24Regular,
  Play24Regular, Star24Regular,
} from '@fluentui/react-icons'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import DataTable from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import type { ItemDetails, ItemWatchHistory, ItemWatchUser } from '@/shared/types/library'

const userCol = createColumnHelper<ItemWatchUser>()
const historyCol = createColumnHelper<ItemWatchHistory>()

function posterUrl(itemId: string): string {
  return `/proxy/Items/Images/Primary/?id=${encodeURIComponent(itemId)}&fillWidth=420&quality=95`
}

function formatDate(value?: string | null): string {
  if (!value) return '—'
  try { return format(parseISO(value), 'dd/MM/yyyy HH:mm') } catch { return value }
}

export default function ItemDetailPage() {
  const { itemId, libraryId } = useParams<{ itemId: string; libraryId: string }>()
  const navigate = useNavigate()
  const [details, setDetails] = useState<ItemDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!itemId) return
    const load = (showLoading = true) => {
      if (showLoading) setLoading(true)
      api.get(`/stats/getItemDetails?itemId=${itemId}`)
        .then((res) => setDetails(res.data))
        .catch(() => setError('Impossible de charger le média'))
        .finally(() => setLoading(false))
    }

    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [itemId])

  const item = details?.item

  const userColumns: ColumnDef<ItemWatchUser, any>[] = [
    userCol.accessor('UserName', {
      header: 'Utilisateur',
      cell: (info) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Avatar sx={{ width: 28, height: 28, bgcolor: 'primary.main', fontSize: 12 }}>
            {(info.getValue() || '?').charAt(0).toUpperCase()}
          </Avatar>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>{info.getValue()}</Typography>
          {info.row.original.IsActive && <Chip label="En cours" size="small" color="primary" sx={{ height: 20, fontSize: 11 }} />}
        </Box>
      ),
    }),
    userCol.accessor('PlayCount', { header: 'Lectures' }),
    userCol.accessor('TotalWatchTime', { header: 'Temps total', cell: (info) => formatWatchTime(info.getValue()) }),
    userCol.accessor('LastWatched', { header: 'Dernière lecture', cell: (info) => formatDate(info.getValue()) }),
  ]

  const historyColumns: ColumnDef<ItemWatchHistory, any>[] = [
    historyCol.accessor('UserName', { header: 'Utilisateur' }),
    historyCol.accessor('ActivityDateInserted', { header: 'Quand', cell: (info) => formatDate(info.getValue()) }),
    historyCol.accessor('PlaybackDuration', { header: 'Durée', cell: (info) => formatWatchTime(info.getValue()) }),
    historyCol.accessor('Client', { header: 'Client', cell: (info) => info.getValue() ?? '—' }),
    historyCol.accessor('DeviceName', { header: 'Appareil', cell: (info) => info.getValue() ?? '—' }),
    historyCol.accessor('PlayMethod', { header: 'Méthode', cell: (info) => info.getValue() ?? '—' }),
    historyCol.accessor('IsActive', {
      header: 'Statut',
      cell: (info) => info.getValue()
        ? <Chip label="En cours" size="small" color="primary" sx={{ height: 20, fontSize: 11 }} />
        : <Chip label="Terminé" size="small" sx={{ height: 20, fontSize: 11 }} />,
    }),
  ]

  return (
    <>
      <Box
        component="button"
        onClick={() => navigate(libraryId ? `/libraries/${libraryId}` : '/libraries')}
        style={{ all: 'unset', cursor: 'pointer' }}
      >
        <Typography variant="body2" color="primary.main" sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 2 }}>
          <ArrowLeft24Regular style={{ fontSize: 18 }} />
          Retour à la bibliothèque
        </Typography>
      </Box>

      <PageHeader title={item?.Name ?? 'Média'} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 3 }}>
          {loading ? (
            <Skeleton variant="rectangular" height={360} sx={{ borderRadius: 2 }} />
          ) : item ? (
            <Card sx={{ overflow: 'hidden' }}>
              <Box
                component="img"
                src={posterUrl(item.Id)}
                alt={item.Name}
                sx={{ width: '100%', aspectRatio: '2 / 3', objectFit: 'cover', display: 'block' }}
              />
            </Card>
          ) : null}
        </Grid>

        <Grid size={{ xs: 12, md: 9 }}>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label="Lectures" value={details?.stats.TotalPlays ?? '—'} icon={<Play24Regular />} loading={loading} />
            </Grid>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label="Temps total" value={details ? formatWatchTime(details.stats.TotalWatchTime) : '—'} icon={<Clock24Regular />} loading={loading} />
            </Grid>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label="Utilisateurs" value={details?.stats.UniqueUsers ?? '—'} icon={<People24Regular />} loading={loading} />
            </Grid>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label="Note" value={item?.CommunityRating ? `★ ${item.CommunityRating.toFixed(1)}` : '—'} icon={<Star24Regular />} loading={loading} />
            </Grid>
          </Grid>

          <Card>
            <CardContent>
              {loading ? (
                <>
                  <Skeleton width="50%" height={28} />
                  <Skeleton width="80%" />
                  <Skeleton width="70%" />
                </>
              ) : item ? (
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, mb: 1 }}>{item.Name}</Typography>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 2 }}>
                    {item.Type && <Chip label={item.Type} size="small" />}
                    {item.ProductionYear && <Chip label={item.ProductionYear} size="small" />}
                    {details?.stats.IsActive && <Chip label="Lecture en cours" color="primary" size="small" />}
                    {item.Genres?.map((genre) => <Chip key={genre} label={genre} size="small" variant="outlined" />)}
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    Dernière lecture : {formatDate(details?.stats.LastWatched)}
                  </Typography>
                  {item.Path && (
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1, wordBreak: 'break-all' }}>
                      {item.Path}
                    </Typography>
                  )}
                </Box>
              ) : (
                <Typography variant="body2" color="text.secondary">Média introuvable.</Typography>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>Qui a regardé ?</Typography>
          <DataTable data={details?.users ?? []} columns={userColumns} loading={loading} searchable={false} />
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>Historique des lectures</Typography>
          <DataTable data={details?.history ?? []} columns={historyColumns} loading={loading} searchPlaceholder="Rechercher dans l'historique..." />
        </CardContent>
      </Card>
    </>
  )
}
